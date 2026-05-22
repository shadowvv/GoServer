package dbService

import (
	"errors"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type MySQLConfig struct {
	Dsn                string        `yaml:"dsn"`
	MaxIdleConnections int           `yaml:"maxIdleConnections"`
	MaxOpenConnections int           `yaml:"maxOpenConnections"`
	MaxLifetime        time.Duration `yaml:"maxLifetime"`
}

func InitMySQL(config *MySQLConfig, zapLogger *zap.Logger) (*gorm.DB, error) {
	if zapLogger == nil {
		return nil, errors.New("zapLogger is nil")
	}
	if config == nil {
		return nil, errors.New("mysqlConfig is nil")
	}

	gormLogger := NewZapGormLogger(zapLogger, logger.Info)

	db, err := gorm.Open(mysql.Open(config.Dsn), &gorm.Config{
		Logger:         gormLogger,
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	return db, nil
}
