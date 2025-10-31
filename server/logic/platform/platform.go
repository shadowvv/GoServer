package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/fileLoader"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/netService"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
)

var serverId int32    // 服务ID
var serverType string // 服务类型
var env string        // 环境

func BootPlatform() {
	InitBootingLog()
	config := &ServerConfig{}
	err := fileLoader.LoadYaml("config/serverConfig.yaml", config)
	if err != nil {
		log.Fatalf("[platform] Load server config error:%v", err)
	}

	allPlatformConfig := &AllPlatformConfig{}
	err = fileLoader.LoadYaml("config/platformConfig.yaml", allPlatformConfig)
	if err != nil {
		log.Fatalf("[platform] Load platform config error:%v", err)
	}
	env = config.Environment
	serverId = config.ServerId
	serverType = config.ServerType

	configs, ok := allPlatformConfig.Configs[serverType]
	if !ok {
		log.Fatalf("[platform] No config for server type %d", serverType)
	}
	cfg, ok := configs[env]
	if !ok {
		log.Fatalf("[platform] No config for server type %d and environment %d", serverType, env)
	}

	err = logger.InitLoggerByConfig(cfg.LoggerConfig)
	if err != nil {
		logger.Error("[platform] Init logger error", zap.Error(err))
	}

	logger.Info("[platform] boot platform", zap.Int32("serverId", serverId), zap.String("serverType", serverType), zap.String("env", env))

	err = InitDBService(cfg.MySQLConfig, cfg.RedisConfig, cfg.RunConfig)
	if err != nil {
		logger.Error("[platform] Init db error", zap.Error(err))
	}
	err = InitNetService(cfg.NetConfig)
	if err != nil {
		logger.Error("[platform] Init server error", zap.Error(err))
	}

	logger.Info("[platform] Boot platform success")
}

func InitBootingLog() {
	file, err := os.OpenFile("bootingErrorLog.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("[platform] faildd to open file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

var sessionManager = NewSessionManager()
var messageCodec = NewCodec()
var router = netService.NewRouter()

func InitNetService(config *netService.NetConfig) error {
	server := netService.NewNetService(config, serverId, sessionManager, messageCodec, router)
	go func() {
		err := server.Start()
		if err != nil {
			logger.Error("[platform] Start server error", zap.Error(err))
		}
	}()
	return nil
}

// RegisterProcess 注册消息处理
func RegisterProcess(msgType uint32, msgID int32, msg proto.Message) {
	router.RegisterProcess(msgType, msgID, msg)
}

// RegisterProcessor 注册消息处理
func RegisterProcessor(msgType uint32, processor serviceInterface.MessageProcessorInterface) {
	router.RegisterProcessor(msgType, processor)
}

var dbPoolManager *DBPoolManager

func InitDBService(mySQLConfig *db.MySQLConfig, redisConfig *db.RedisConfig, runConfig *RunConfig) error {
	err := db.InitDBService(mySQLConfig, redisConfig)
	if err != nil {
		return err
	}
	dbPoolManager = NewDBPoolManager(db.DB)
	for _, poolInfo := range runConfig.DBPoolInfo {
		err := AddDBPool(enum.DBPoolType(poolInfo.PoolType), poolInfo.WorkerNum, poolInfo.WorkerTaskSize)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddDBPool(poolType enum.DBPoolType, workerNum, workerTaskSize int32) error {
	return dbPoolManager.AddDBPool(poolType, workerNum, workerTaskSize)
}

func AddDBTask(poolType enum.DBPoolType, playerID int64, task DBTask) {

}
