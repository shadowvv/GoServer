package platform

import (
	"GoServer/server/logic/enum"
	"GoServer/server/logic/logicInterface"
	"GoServer/server/service/log"
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

func (p *Platform) Init(env enum.Enviroment) {
	log.InitLogger("config/log.yaml")
}
