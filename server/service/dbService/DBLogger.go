package dbService

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var SLOW_THRESHOLD = 5 * time.Millisecond

type ZapGormLogger struct {
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

func NewZapGormLogger(_ *zap.Logger, level logger.LogLevel) *ZapGormLogger {
	return &ZapGormLogger{
		logLevel:      level,
		slowThreshold: SLOW_THRESHOLD,
	}
}

// LogMode implements gorm logger.Interface.
func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		zap.S().Infof(msg, data...)
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		zap.S().Warnf(msg, data...)
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		zap.S().Errorf(msg, data...)
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
		zap.L().Info("SQL Record Not Found",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case err != nil && l.logLevel >= logger.Error:
		zap.L().Error("SQL Error",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case elapsed > l.slowThreshold && l.logLevel >= logger.Warn:
		zap.L().Warn("Slow SQL",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)

	case l.logLevel >= logger.Info:
		zap.L().Info("SQL",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.String("caller", utils.FileWithLineNum()),
		)
	}
}
