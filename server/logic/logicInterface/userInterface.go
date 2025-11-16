package logicInterface

import "github.com/drop/GoServer/server/service/serviceInterface"

type UserBaseInterface interface {
	GetUserId() int64
	GetUserAccount() string
	GetUserServerId() int32
	GetSession() serviceInterface.SessionInterface
}

type UserQueueInterface interface {
	UserBaseInterface
	GetQueuePosition() int32
}

type UserSceneInterface interface {
	UserBaseInterface
	GetSceneId() int32
}
