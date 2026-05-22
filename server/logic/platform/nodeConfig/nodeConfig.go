package nodeConfig

import (
	"time"

	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/etcd"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/netService"
	"github.com/drop/GoServer/server/service/payService"
	"github.com/drop/GoServer/server/service/webService"
)

type NodeConfig struct {
	NodeId      int32  `yaml:"nodeId"`
	NodeType    string `yaml:"nodeType"`
	Environment string `yaml:"environment"`
	ConfigName  string `yaml:"configName"`
	ChannelId   int32  `yaml:"channelId"`
}

type AllPlatformConfig struct {
	Configs map[string]map[string]*PlatformConfig `yaml:"configs"`
}

type PlatformConfig struct {
	LoggerConfig      *logger.LoggerConfig         `yaml:"loggerConfig"`
	NetConfig         *netService.NetConfig        `yaml:"netConfig"`
	HttpConfig        *webService.HttpConfig       `yaml:"httpConfig"`
	ServerDBConfig    *dbService.MySQLConfig       `yaml:"serverDBConfig"`
	BackEndDBConfig   *dbService.MySQLConfig       `yaml:"backendDBConfig"`
	LogDBConfig       *dbService.MySQLConfig       `yaml:"logDBConfig"`
	GameDBConfig      *dbService.MySQLConfig       `yaml:"gameDBConfig"`
	RankDBConfig      *dbService.MySQLConfig       `yaml:"rankDBConfig"`
	RedisConfig       *dbService.RedisConfig       `yaml:"redisConfig"`
	EtcdConfig        *etcd.Config                 `yaml:"etcdConfig"`
	RpcConfig         *ServerNodeService.RpcConfig `yaml:"rpcConfig"`
	RunConfig         *RunConfig                   `yaml:"runConfig"`
	PayConfig         []*payService.PayConfig      `yaml:"payConfig"`
	AuditServerConfig *AuditServerConfig           `yaml:"auditServerConfig"`
}

type AuditServerConfig struct {
	Addr string `yaml:"addr"`
}

type RunConfig struct {
	DBPoolInfo                 []*DBPoolInfo                    `yaml:"dbPoolInfo"`
	SceneGoroutineMaxPlayerNum int32                            `yaml:"sceneGoroutineMaxPlayerNum"`
	PlayerOfflineTimeout       time.Duration                    `yaml:"playerOfflineTimeout"`
	MessageProcessConfig       map[string]*MessageProcessConfig `yaml:"messageProcessConfig"`
}

type DBPoolInfo struct {
	PoolType       string `yaml:"poolType"`
	WorkerNum      int32  `yaml:"workerNum"`
	WorkerTaskSize int32  `yaml:"workerTaskSize"`
}

type MessageProcessConfig struct {
	RoutineNum               int32         `yaml:"routineNum"`
	TickInterval             time.Duration `yaml:"tickInterval"`
	MessageCountPerTick      int32         `yaml:"messageCountPerTick"`
	MessageBufferSize        int32         `yaml:"messageBufferSize"`
	InnerMessageCountPerTick int32         `yaml:"innerMessageCountPerTick"`
	InnerMessageBufferSize   int32         `yaml:"innerMessageBufferSize"`
}
