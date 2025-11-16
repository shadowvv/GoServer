package db

import (
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

var DB *gorm.DB

func InitMySQL(dsn string, maxIdle, maxOpen, maxLifetime int, zapLogger *zap.Logger) error {
	gormLogger := NewZapGormLogger(zapLogger, logger.Info)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         gormLogger,
		NamingStrategy: CamelNamingStrategy{},
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

	DB = db
	return nil
}
