package platform

import (
	"fmt"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/pb"
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

func InitPlatform(env enum.Enviroment) {
	InitLogger(env)
	user := &logicInterface.BasicUserInfo{}
	Info("Init platform", user)
	Error("Error platform", user)

	InitServer(env)
}

var sessionManager SessionManager

func InitServer(env enum.Enviroment) {
	server := sNet.NewServer(":8080", 1, &sessionManager, NewCodec(), sNet.NewRouter())
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
