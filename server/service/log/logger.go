package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"maxsize"`
	MaxBackups int    `yaml:"maxbackups"`
	MaxAge     int    `yaml:"maxage"`
	Compress   bool   `yaml:"compress"`
	ServerType string `yaml:"server_type"`
	ServerID   string `yaml:"server_id"`
}

var Logger *zap.Logger
var cfg Config

// InitLogger 初始化日志
func InitLogger(configPath string) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read logger config: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse logger config: %v", err)
	}

	writeSyncer := zapcore.AddSync(os.Stdout) // 控制台输出
	if cfg.Filename != "" {
		fileSyncer := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		writeSyncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileSyncer))
	}

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

	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// Info
func Info(msg string, playerID ...uint) {
	fields := []zap.Field{
		zap.String("server_type", cfg.ServerType),
		zap.String("server_id", cfg.ServerID),
	}
	if len(playerID) > 0 {
		fields = append(fields, zap.Uint("player_id", playerID[0]))
	}
	Logger.Info(msg, fields...)
}

// Debug
func Debug(msg string, playerID ...uint) {
	fields := []zap.Field{
		zap.String("server_type", cfg.ServerType),
		zap.String("server_id", cfg.ServerID),
	}
	if len(playerID) > 0 {
		fields = append(fields, zap.Uint("player_id", playerID[0]))
	}
	Logger.Debug(msg, fields...)
}

// Error
func Error(msg string, playerID ...uint) {
	fields := []zap.Field{
		zap.String("server_type", cfg.ServerType),
		zap.String("server_id", cfg.ServerID),
	}
	if len(playerID) > 0 {
		fields = append(fields, zap.Uint("player_id", playerID[0]))
	}
	Logger.Error(msg, fields...)
}
