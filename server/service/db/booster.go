package db

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
)

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

func InitAll(mysqlCfg *MySQLConfig, redisCfg *RedisConfig) error {
	if err := InitMySQL(
		mysqlCfg.DSN,
		mysqlCfg.MaxIdleConnections,
		mysqlCfg.MaxOpenConnections,
		mysqlCfg.MaxLifetime,
	); err != nil {
		logger.Error(fmt.Sprintf("[db] Failed to init MySQL: %v", err))
		return err
	}

	logger.Info("[db] MySQL initialized successfully")

	if err := InitRedis(
		redisCfg.Addr,
		redisCfg.DB,
		redisCfg.PoolSize,
	); err != nil {
		logger.Error(fmt.Sprintf("[db] Failed to init Redis: %v", err))
		return err
	}

	logger.Info("[db] Database initialized successfully")
	return nil
}
