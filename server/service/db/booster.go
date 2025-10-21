package db

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
)

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

func InitAll(cfg *DBConfig) {
	if err := InitMySQL(
		cfg.MySQL.DSN,
		cfg.MySQL.MaxIdle,
		cfg.MySQL.MaxOpen,
		cfg.MySQL.MaxLifetime,
	); err != nil {
		logger.Error(fmt.Sprintf("[db] Failed to init MySQL: %v", err))
	}

	logger.Info("[db] MySQL initialized successfully")

	if err := InitRedis(
		cfg.Redis.Addr,
		cfg.Redis.DB,
		cfg.Redis.PoolSize,
	); err != nil {
		logger.Error(fmt.Sprintf("[db] Failed to init Redis: %v", err))
	}

	logger.Info("[db] Database initialized successfully")
}
