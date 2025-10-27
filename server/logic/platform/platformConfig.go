package platform

import "github.com/drop/GoServer/server/service/logger"

type platformConfigList struct {
	PlatformConfigList []*platformConfig
}

type platformConfig struct {
	*NetConfig
	*MySQLConfig
	*RedisConfig
	*logger.LoggerConfig
	*ServerConfig
}

type NetConfig struct {
	TargetId   int32 `yaml:"targetId"`
	FunctionId int32 `yaml:"functionId"`
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

type ServerConfig struct {
	Port int `yaml:"port"`
}
