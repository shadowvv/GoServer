package db

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
)

type DBConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
	Redis RedisConfig `yaml:"redis"`
}

type MySQLConfig struct {
	DSN                string `yaml:"dsn"`
	MaxIdleConnections int    `yaml:"maxIdleConnections"`
	MaxOpenConnections int    `yaml:"maxOpenConnections"`
	MaxLifetime        int    `yaml:"maxLifetime"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"poolSize"`
}

func InitAll(cfg *DBConfig) {
	if err := InitMySQL(
		cfg.MySQL.DSN,
		cfg.MySQL.MaxIdleConnections,
		cfg.MySQL.MaxOpenConnections,
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
