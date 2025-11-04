package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var Logger *zap.Logger

type LoggerConfig struct {
	InfoFilename  string `yaml:"infoFilename"`
	ErrorFilename string `yaml:"errorFilename"`
	MaxSize       int    `yaml:"maxsize"`
	MaxBackups    int    `yaml:"maxbackups"`
	MaxAge        int    `yaml:"maxage"`
	Compress      bool   `yaml:"compress"`
}

func InitLoggerByConfig(config *LoggerConfig) error {

	// 获取当前日期用于文件名
	currentDate := time.Now().Format("2006-01-02")

	// Info 日志配置（包含 Debug/Info/Warn）
	infoWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fmt.Sprintf("%s-%s.log", config.InfoFilename, currentDate),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
		LocalTime:  true,
	})

	// Error 日志配置（Error 及以上）
	errorWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fmt.Sprintf("%s-%s.log", config.ErrorFilename, currentDate),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
		LocalTime:  true,
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
	errorEncoderConfig.FunctionKey = ""
	errorEncoderConfig.CallerKey = ""
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

	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}
