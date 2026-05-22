package dbService

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"poolSize"`
	Password string `yaml:"password"`
}

var REDIS_READ_TIME_OUT = 10 * time.Second

var RDB *redis.Client

func InitRedis(config *RedisConfig) error {
	if config == nil {
		return errors.New("redis config is nil")
	}

	client := redis.NewClient(&redis.Options{
		Addr:        config.Addr,
		DB:          config.DB,
		PoolSize:    config.PoolSize,
		ReadTimeout: REDIS_READ_TIME_OUT,
		Password:    config.Password,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	RDB = client
	return nil
}
