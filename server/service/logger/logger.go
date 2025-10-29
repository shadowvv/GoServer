package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

var logger *zap.Logger

type LoggerConfig struct {
	InfoFilename  string `yaml:"infoFilename"`
	ErrorFilename string `yaml:"errorFilename"`
	MaxSize       int    `yaml:"maxsize"`
	MaxBackups    int    `yaml:"maxbackups"`
	MaxAge        int    `yaml:"maxage"`
	Compress      bool   `yaml:"compress"`
}

func InitLoggerByConfigPath(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("[logger] Failed to read logger LoggerConfig: %v", err)
	}

	config := &LoggerConfig{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("[logger] Failed to parse logger LoggerConfig: %v", err)
	}
	return InitLoggerByConfig(config)
}

func InitLoggerByConfig(config *LoggerConfig) error {
	// Info 日志配置（包含 Debug/Info/Warn）
	infoWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.InfoFilename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	})

	// Error 日志配置（Error 及以上）
	errorWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.ErrorFilename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	})

	consoleWriter := zapcore.AddSync(os.Stdout)

	infoEncoderConfig := zap.NewProductionEncoderConfig()
	infoEncoderConfig.TimeKey = "time"
	infoEncoderConfig.LevelKey = "level"
	infoEncoderConfig.FunctionKey = ""
	infoEncoderConfig.CallerKey = ""
	infoEncoderConfig.MessageKey = "msg"
	infoEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	infoEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	infoEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	errorEncoderConfig := zap.NewProductionEncoderConfig()
	errorEncoderConfig.TimeKey = "time"
	errorEncoderConfig.LevelKey = "level"
	errorEncoderConfig.FunctionKey = "func"
	errorEncoderConfig.CallerKey = "caller"
	errorEncoderConfig.StacktraceKey = "stacktrace"
	errorEncoderConfig.MessageKey = "msg"
	errorEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	errorEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	errorEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// infoCore: 处理 Debug/Info/Warn （小于 Error 的都归 info）
	infoLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.DebugLevel && l < zapcore.ErrorLevel
	})
	infoCore := zapcore.NewCore(zapcore.NewJSONEncoder(infoEncoderConfig), zapcore.NewMultiWriteSyncer(infoWriter, consoleWriter), infoLevelEnabler)

	// errorCore: 处理 Error 及以上
	errorLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.ErrorLevel
	})
	errorCore := zapcore.NewCore(zapcore.NewJSONEncoder(errorEncoderConfig), zapcore.NewMultiWriteSyncer(errorWriter, consoleWriter), errorLevelEnabler)

	core := zapcore.NewTee(infoCore, errorCore)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}
