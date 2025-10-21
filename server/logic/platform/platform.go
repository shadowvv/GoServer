package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/pb"
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
	go InitServer(env)
	user := &logicInterface.BasicUserInfo{}
	Info("Init platform", user)
	Error("Error platform", user)
}

func InitServer(env enum.Enviroment) {
	server := sNet.NewServer(":8080", 1, nil, NewCodec(), sNet.NewRouter())
	server.Register(1, &pb.TestMessageReq{}, func(msgId uint32, message proto.Message) {

	})

	err := server.Start()
	if err != nil {
		return
	}
}
