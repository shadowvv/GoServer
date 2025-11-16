package db

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
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

func InitDBService(mysqlCfg *MySQLConfig, redisCfg *RedisConfig, zapLogger *zap.Logger) error {

	logger.Info("[db] Init database", zap.String("dsn", mysqlCfg.DSN), zap.Int("maxIdleConnections", mysqlCfg.MaxIdleConnections), zap.Int("maxOpenConnections", mysqlCfg.MaxOpenConnections), zap.Int("maxLifetime", mysqlCfg.MaxLifetime))

	if err := InitMySQL(
		mysqlCfg.DSN,
		mysqlCfg.MaxIdleConnections,
		mysqlCfg.MaxOpenConnections,
		mysqlCfg.MaxLifetime,
		zapLogger,
	); err != nil {
		logger.Error(fmt.Sprintf("[db] Failed to init MySQL: %v", err))
		return err
	}

	logger.Info("[db] MySQL initialized successfully")

	logger.Info("[db] Init redis", zap.String("addr", redisCfg.Addr), zap.Int("db", redisCfg.DB), zap.Int("poolSize", redisCfg.PoolSize))

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
