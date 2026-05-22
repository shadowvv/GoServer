package dbService

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var SLOW_THRESHOLD = 5 * time.Millisecond

type ZapGormLogger struct {
	zapLogger     *zap.Logger
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

func NewZapGormLogger(zapLogger *zap.Logger, level logger.LogLevel) *ZapGormLogger {
	return &ZapGormLogger{
		zapLogger:     zapLogger,
		logLevel:      level,
		slowThreshold: SLOW_THRESHOLD, // 慢查询阈值
	}
}

// 实现接口方法
func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.zapLogger.Sugar().Infof(msg, data...)
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.zapLogger.Sugar().Warnf(msg, data...)
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.zapLogger.Sugar().Errorf(msg, data...)
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil && errors.Is(err, gorm.ErrRecordNotFound) && l.logLevel >= logger.Info:
		// 1️⃣ NoRecord → Info
		l.zapLogger.Info("SQL Record Not Found",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case err != nil && l.logLevel >= logger.Error:
		// 2️⃣ 其他错误 → Error
		l.zapLogger.Error("SQL Error",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case elapsed > l.slowThreshold && l.logLevel >= logger.Warn:
		// 3️⃣ 慢 SQL → Warn
		l.zapLogger.Warn("Slow SQL",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case l.logLevel >= logger.Info:
		// 4️⃣ 普通 SQL → Info
		l.zapLogger.Info("SQL",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)
	}

}
