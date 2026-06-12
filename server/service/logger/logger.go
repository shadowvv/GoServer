package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var Logger *zap.Logger
var once sync.Once
var mu sync.RWMutex

type LoggerConfig struct {
	InfoFilename  string `yaml:"infoFilename"`
	ErrorFilename string `yaml:"errorFilename"`
	MaxSize       int    `yaml:"maxsize"`
	MaxBackups    int    `yaml:"maxBackups"`
	OnlyConsole   bool   `yaml:"onlyConsole"`
}

var currentConfig *LoggerConfig
var currentDate string
var currentClosers []io.Closer

func InitLoggerByConfig(config *LoggerConfig) error {
	var err error
	once.Do(func() {
		currentConfig = config
		err = initLogger(config)
		if err == nil && !config.OnlyConsole {
			startDailyRotateWatcher()
		}
	})
	return err
}

func initLogger(config *LoggerConfig) error {
	logger, dateStr, closers, err := buildLogger(config)
	if err != nil {
		return err
	}

	mu.Lock()
	Logger = logger
	currentDate = dateStr
	currentClosers = closers
	mu.Unlock()

	zap.ReplaceGlobals(logger)
	return nil
}

func buildLogger(config *LoggerConfig) (*zap.Logger, string, []io.Closer, error) {
	dateStr := time.Now().Format("2006-01-02")

	if config.OnlyConsole {
		consoleEncoderCfg := zap.NewProductionEncoderConfig()
		consoleEncoderCfg.TimeKey = "time"
		consoleEncoderCfg.LevelKey = "level"
		consoleEncoderCfg.FunctionKey = ""
		consoleEncoderCfg.CallerKey = "caller"
		consoleEncoderCfg.MessageKey = "msg"
		consoleEncoderCfg.StacktraceKey = "stacktrace"
		consoleEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderCfg),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)

		logger := zap.New(consoleCore, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
		return logger, dateStr, nil, nil
	}

	infoPath := buildDailyFilename(config.InfoFilename, dateStr)
	errorPath := buildDailyFilename(config.ErrorFilename, dateStr)

	infoFile := &lumberjack.Logger{
		Filename:   infoPath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     7,
		LocalTime:  true,
		Compress:   false,
	}
	infoWriter := zapcore.AddSync(infoFile)

	errorFile := &lumberjack.Logger{
		Filename:   errorPath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     7,
		LocalTime:  true,
		Compress:   false,
	}
	errorWriter := zapcore.AddSync(errorFile)
	// File logs use JSON encoding.
	infoFileEncoderCfg := zap.NewProductionEncoderConfig()
	infoFileEncoderCfg.TimeKey = "time"
	infoFileEncoderCfg.LevelKey = "level"
	infoFileEncoderCfg.FunctionKey = ""
	infoFileEncoderCfg.CallerKey = ""
	infoFileEncoderCfg.MessageKey = "msg"
	infoFileEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	infoFileEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	infoFileEncoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	errorFileEncoderCfg := zap.NewProductionEncoderConfig()
	errorFileEncoderCfg.TimeKey = "time"
	errorFileEncoderCfg.LevelKey = "level"
	errorFileEncoderCfg.FunctionKey = ""
	errorFileEncoderCfg.CallerKey = ""
	errorFileEncoderCfg.StacktraceKey = "stacktrace"
	errorFileEncoderCfg.MessageKey = "msg"
	errorFileEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	errorFileEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	errorFileEncoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	infoLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.DebugLevel && l < zapcore.ErrorLevel
	})
	errorLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.ErrorLevel
	})

	infoFileCore := zapcore.NewCore(zapcore.NewJSONEncoder(infoFileEncoderCfg), infoWriter, infoLevelEnabler)
	errorFileCore := zapcore.NewCore(zapcore.NewJSONEncoder(errorFileEncoderCfg), errorWriter, errorLevelEnabler)

	core := zapcore.NewTee(infoFileCore, errorFileCore)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, dateStr, []io.Closer{infoFile, errorFile}, nil
}

func buildDailyFilename(base string, dateStr string) string {
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	if ext == "" {
		ext = ".log"
	}
	return fmt.Sprintf("%s-%s%s", nameWithoutExt, dateStr, ext)
}

func closeLoggerOutputs(closers []io.Closer) {
	for _, closer := range closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil {
			fmt.Printf("[ERROR] close logger output failed: %v\n", err)
		}
	}
}

func startDailyRotateWatcher() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			today := time.Now().Format("2006-01-02")

			mu.RLock()
			needRotate := currentConfig != nil && today != currentDate
			cfg := currentConfig
			mu.RUnlock()

			if !needRotate {
				continue
			}

			newLogger, newDate, newClosers, err := buildLogger(cfg)
			if err != nil {
				fmt.Printf("[ERROR] rebuild logger failed: %v\n", err)
				continue
			}

			mu.Lock()
			oldLogger := Logger
			oldClosers := currentClosers
			Logger = newLogger
			currentDate = newDate
			currentClosers = newClosers
			mu.Unlock()

			zap.ReplaceGlobals(newLogger)

			if oldLogger != nil {
				_ = oldLogger.Sync()
			}
			closeLoggerOutputs(oldClosers)

			newLogger.Info(fmt.Sprintf("[logger] daily rotate success, new date=%s", newDate))
		}
	}()
}

func ensureLogger() {
	mu.RLock()
	if Logger != nil {
		mu.RUnlock()
		return
	}
	mu.RUnlock()

	once.Do(func() {
		fmt.Println("[WARN] logger not initialized, using default console logger")

		consoleEncoderCfg := zap.NewProductionEncoderConfig()
		consoleEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderCfg),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)

		logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

		mu.Lock()
		Logger = logger
		mu.Unlock()

		zap.ReplaceGlobals(logger)

		fmt.Println("[WARN] logger not initialized, using default console logger")
	})
}

func getLogger() *zap.Logger {
	ensureLogger()
	mu.RLock()
	defer mu.RUnlock()
	return Logger
}

func InfoWithSprintf(format string, args ...interface{}) {
	getLogger().Info(fmt.Sprintf(format, args...))
}

func InfoWithZapFields(format string, fields ...zap.Field) {
	getLogger().Info(format, fields...)
}

func ErrorBySprintf(format string, args ...interface{}) {
	getLogger().Error(fmt.Sprintf(format, args...))
}

func ErrorWithZapFields(msg string, fields ...zap.Field) {
	getLogger().Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	getLogger().Warn(msg, fields...)
}
