package db

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

var RDB *redis.Client

func InitRedis(addr string, dbIndex, poolSize int) error {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		DB:          dbIndex,
		PoolSize:    poolSize,
		ReadTimeout: 10 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	RDB = client
	return nil
}
