package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/sNet"
)

type ServerConfig struct {
	ServerId    int32            `yaml:"serverId"`
	ServerType  enum.ServerType  `yaml:"serverType"`
	Environment enum.Environment `yaml:"environment"`
}

type AllPlatformConfig struct {
	configs map[enum.ServerType]map[enum.Environment]*PlatformConfig
}

type PlatformConfig struct {
	*sNet.NetConfig
	*db.MySQLConfig
	*db.RedisConfig
	*logger.LoggerConfig
	*RunConfig
}

type RunConfig struct {
}
