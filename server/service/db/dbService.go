package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

var DB *gorm.DB

type DBConfig struct {
	MySQL struct {
		DSN         string `yaml:"dsn"`
		MaxIdle     int    `yaml:"maxIdle"`
		MaxOpen     int    `yaml:"maxOpen"`
		MaxLifetime int    `yaml:"maxLifetime"`
	} `yaml:"mysql"`
	Redis struct {
		Addr     string `yaml:"addr"`
		DB       int    `yaml:"db"`
		PoolSize int    `yaml:"poolSize"`
	} `yaml:"redis"`
}

func InitMySQL(dsn string, maxIdle, maxOpen, maxLifetime int) error {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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
