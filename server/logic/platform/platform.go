package platform

import (
	"GoServer/server/logic/logicInterface"
)

type NetRoutine func(message interface{}, user logicInterface.UserBaseInterface)

type Platform interface {
	Init()
	Run()
	Stop()
	AddNetRoutine(routine NetRoutine, targetId int32, functionId int32)
}
