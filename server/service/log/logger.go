package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Level         string `yaml:"level"`
	InfoFilename  string `yaml:"infoFilename"`
	ErrorFilename string `yaml:"errorFilename"`
	MaxSize       int    `yaml:"maxsize"`
	MaxBackups    int    `yaml:"maxbackups"`
	MaxAge        int    `yaml:"maxage"`
	Compress      bool   `yaml:"compress"`
}

var logger *zap.Logger
var cfg Config

// InitLogger 初始化日志
func InitLogger(configPath string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read logger config: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse logger config: %v", err)
	}

	// Info 日志配置（包含 Debug/Info/Warn）
	infoWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.InfoFilename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	// Error 日志配置（Error 及以上）
	errorWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.ErrorFilename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	consoleWriter := zapcore.AddSync(os.Stdout)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.FunctionKey = "func"
	encoderConfig.MessageKey = "msg"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 选择最小写入等级
	var lvl zapcore.Level
	switch cfg.Level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "info":
		lvl = zapcore.InfoLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	// infoCore: 处理 Debug/Info/Warn （小于 Error 的都归 info）
	infoLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.DebugLevel && l < zapcore.ErrorLevel && l >= lvl
	})
	infoCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.NewMultiWriteSyncer(infoWriter, consoleWriter), infoLevelEnabler)

	// errorCore: 处理 Error 及以上
	errorLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.ErrorLevel && l >= lvl
	})
	errorCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.NewMultiWriteSyncer(errorWriter, consoleWriter), errorLevelEnabler)

	core := zapcore.NewTee(infoCore, errorCore)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// Info
func Info(msg string, playerID int64, serverId int32, serverType int32) {
	fields := []zap.Field{
		zap.Int64("player_id", playerID),
		zap.Int32("server_type", serverType),
		zap.Int32("server_id", serverId),
	}
	logger.Info(msg, fields...)
}

// Debug
func Debug(msg string, playerID int64, serverId int32, serverType int32) {
	fields := []zap.Field{
		zap.Int64("player_id", playerID),
		zap.Int32("server_type", serverType),
		zap.Int32("server_id", serverId),
	}
	logger.Debug(msg, fields...)
}

// Error
func Error(msg string, playerID int64, serverId int32, serverType int32) {
	fields := []zap.Field{
		zap.Int64("player_id", playerID),
		zap.Int32("server_type", serverType),
		zap.Int32("server_id", serverId),
	}
	logger.Error(msg, fields...)
}
