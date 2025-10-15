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
	errorFilename string `yaml:"errorFilename"`
	MaxSize       int    `yaml:"maxsize"`
	MaxBackups    int    `yaml:"maxbackups"`
	MaxAge        int    `yaml:"maxage"`
	Compress      bool   `yaml:"compress"`
}

var logger *zap.Logger

// InitLogger 初始化日志
func InitLogger(configPath string) {

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read logger config: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse logger config: %v", err)
	}

	// Info 日志配置
	infoLogger := &lumberjack.Logger{
		Filename:   cfg.InfoFilename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// Error 日志配置
	errorLogger := &lumberjack.Logger{
		Filename:   cfg.errorFilename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	writeSyncerInfo := zapcore.AddSync(infoLogger)
	writeSyncerError := zapcore.AddSync(errorLogger)
	consoleWriter := zapcore.AddSync(os.Stdout)
	writeSyncer := zapcore.NewMultiWriteSyncer(consoleWriter, writeSyncerInfo, writeSyncerError)

	// 日志级别
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.MessageKey = "msg"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		level,
	)

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
