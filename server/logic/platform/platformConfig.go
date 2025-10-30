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
	configs map[enum.ServerType]map[enum.Environment]*PlatformConfig `yaml:"configs"`
}

type PlatformConfig struct {
	*logger.LoggerConfig `yaml:"loggerConfig"`
	*sNet.NetConfig      `yaml:"netConfig"`
	*db.MySQLConfig      `yaml:"mysqlConfig"`
	*db.RedisConfig      `yaml:"redisConfig"`
	*RunConfig           `yaml:"runConfig"`
}

type RunConfig struct {
	DBPoolInfo []*DBPoolInfo `yaml:"dbPoolInfo"`
}

type DBPoolInfo struct {
	PoolType       enum.DBPoolType `yaml:"poolType"`
	WorkerNum      int32           `yaml:"workerNum"`
	WorkerTaskSize int32           `yaml:"workerTaskSize"`
}
