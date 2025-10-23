package platform

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/sNet"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	"log"
	"os"
)

func InitPlatform(env enum.Environment) {
	InitLogger(env)
	InitDB()
	InitServer(env)
}

var sessionManager SessionManager
var codec = NewCodec()
var router = sNet.NewRouter()

func InitServer(env enum.Environment) {
	server := sNet.NewServer(":8080", 1, &sessionManager, codec, router)
	server.Register(1, &pb.TestMessageReq{}, func(msgId uint32, message proto.Message) {
		req := message.(*pb.TestMessageReq)
		logger.Info(fmt.Sprintf("Receive message token:%s platform:%s", req.Token, req.Platform))
		logger.Info("test Receive message")
	})

	err := server.Start()
	if err != nil {
		return
	}
}

var dbPool *DBPool

func InitDB() {
	dbConfig := db.DBConfig{}
	data, err := os.ReadFile("config/serverConfig.yaml")
	if err != nil {
		log.Fatalf("Failed to read database config: %v", err)
	}

	if err := yaml.Unmarshal(data, &dbConfig); err != nil {
		log.Fatalf("Failed to parse database config: %v", err)
	}
	db.InitAll(&dbConfig)
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
		return
	}
	fmt.Printf("LPush %v -> %s\n", val, key)

	// 3. 测试读取
	result, err := db.RDB.LPop(context.Background(), key).Result()
	if err != nil {
		fmt.Printf("LPop error: %v\n", err)
		return
	}
	fmt.Printf("LPop from %s: %v\n", key, result)
}
