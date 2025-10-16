package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
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

func Init(env enum.Enviroment) {
	InitLogger(env)
	user := &logicInterface.BasicUserInfo{}
	Info("Init platform", user)
	Error("Error platform", user)
}
