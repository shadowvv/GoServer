package platform

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/fileLoader"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/sNet"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"log"
	"os"
)

var serverId int32             // 服务ID
var serverType enum.ServerType // 服务类型
var env enum.Environment       // 环境

func InitPlatform() error {
	InitBootingLog()
	config := &ServerConfig{}
	err := fileLoader.LoadYaml("config/serverConfig.yaml", config)
	if err != nil {
		log.Fatalf("Load server config error:%v", err)
		return err
	}

	allPlatformConfig := &AllPlatformConfig{}
	err = fileLoader.LoadYaml("config/platformConfig.yaml", allPlatformConfig)
	if err != nil {
		log.Fatalf("Load platform config error:%v", err)
		return err
	}
	env = config.Environment
	serverId = config.ServerId
	serverType = config.ServerType

	configs := allPlatformConfig.configs[serverType]
	if configs == nil {
		log.Fatalf("No config for server type %d", serverType)
		return err
	}
	cfg := configs[env]
	if cfg == nil {
		log.Fatalf("No config for server type %d and environment %d", serverType, env)
		return err
	}

	err = logger.InitLoggerByConfig(cfg.LoggerConfig)
	if err != nil {
		return err
	}
	err = InitDB(cfg.MySQLConfig, cfg.RedisConfig)
	if err != nil {
		return err
	}
	err = InitServer(cfg.NetConfig)
	if err != nil {
		return err
	}
	return nil
}

func InitBootingLog() {
	file, err := os.OpenFile("bootingLog.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("faildd to open file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

var sessionManager SessionManager
var codec = NewCodec()
var router = sNet.NewRouter()

func InitServer(config *sNet.NetConfig) error {
	server := sNet.NewServer(config, serverId, &sessionManager, codec, router)
	server.Register(1, &pb.TestMessageReq{}, func(msgId uint32, message proto.Message) {
		req := message.(*pb.TestMessageReq)
		logger.Info(fmt.Sprintf("Receive message token:%s platform:%s", req.Token, req.Platform))
		logger.Info("test Receive message")
	})

	err := server.Start()
	if err != nil {
		return err
	}
	return nil
}

var dbPool *DBPool

func InitDB(mySQLConfig *db.MySQLConfig, redisConfig *db.RedisConfig) error {

	err := db.InitAll(mySQLConfig, redisConfig)
	if err != nil {
		return err
	}
	dbPool = NewDBPool(3, db.DB)
	dbPool.Submit(1, func(gdb *gorm.DB) {
		repo := db.PlayerRepository{}
		player, err := repo.GetPlayerByUID(1)
		if err != nil {
			logger.Error("GetPlayerByUID error", zap.Error(err))
			player = &db.Player{
				Account: "test",
				UserId:  1,
			}
			err := repo.SavePlayer(player)
			if err != nil {
				logger.Error("SavePlayer error", zap.Error(err))
				return
			}
			return
		}
		logger.Info("GetPlayerByUID success", zap.Any("player", player))
	})

	// 2. 测试写入
	key := "test"
	val := 10

	if err := db.RDB.LPush(context.Background(), key, val).Err(); err != nil {
		fmt.Printf("LPush error: %v\n", err)
		return err
	}
	fmt.Printf("LPush %v -> %s\n", val, key)

	// 3. 测试读取
	result, err := db.RDB.LPop(context.Background(), key).Result()
	if err != nil {
		fmt.Printf("LPop error: %v\n", err)
		return err
	}
	fmt.Printf("LPop from %s: %v\n", key, result)
	return nil
}
