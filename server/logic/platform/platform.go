package platform

import (
	"fmt"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/db"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/sNet"
	"google.golang.org/protobuf/proto"
)

type NetRoutine func(message interface{}, user logicInterface.UserBaseInterface)

type PlatformInterface interface {
	Init()
	Run()
	Stop()
	AddNetRoutine(routine NetRoutine, targetId int32, functionId int32)
}

type Platform struct {
	NetRoutines map[int32]map[int32]NetRoutine
}

func InitPlatform(env enum.Environment) {
	InitLogger(env)
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

var dbPool = NewDBPool(1, db.DB)

func InitDB() {
	dbConfig := db.DBConfig{}
	db.InitAll(&dbConfig)
}
