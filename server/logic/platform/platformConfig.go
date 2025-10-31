package platform

import (
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/sNet"
)

type ServerConfig struct {
	ServerId    int32  `yaml:"serverId"`
	ServerType  string `yaml:"serverType"`
	Environment string `yaml:"environment"`
}

type AllPlatformConfig struct {
	Configs map[string]map[string]*PlatformConfig `yaml:"configs"`
}

type PlatformConfig struct {
	LoggerConfig *logger.LoggerConfig `yaml:"loggerConfig"`
	NetConfig    *sNet.NetConfig      `yaml:"netConfig"`
	MySQLConfig  *db.MySQLConfig      `yaml:"mysqlConfig"`
	RedisConfig  *db.RedisConfig      `yaml:"redisConfig"`
	RunConfig    *RunConfig           `yaml:"runConfig"`
}

type RunConfig struct {
	DBPoolInfo []*DBPoolInfo `yaml:"dbPoolInfo"`
}

type DBPoolInfo struct {
	PoolType       string `yaml:"poolType"`
	WorkerNum      int32  `yaml:"workerNum"`
	WorkerTaskSize int32  `yaml:"workerTaskSize"`
}
