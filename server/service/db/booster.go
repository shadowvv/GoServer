package db

import "log"

func InitAll(cfg *DBConfig) {
	if err := InitMySQL(
		cfg.MySQL.DSN,
		cfg.MySQL.MaxIdle,
		cfg.MySQL.MaxOpen,
		cfg.MySQL.MaxLifetime,
	); err != nil {
		log.Fatalf("Failed to init MySQL: %v", err)
	}

	if err := InitRedis(
		cfg.Redis.Addr,
		cfg.Redis.DB,
		cfg.Redis.PoolSize,
	); err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}

	log.Println("✅ Database initialized successfully")
}
