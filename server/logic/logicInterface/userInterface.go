package logicInterface

type UserBaseInterface interface {
	GetUserId() int64
	GetUserAccount() string
	GetUserServerId() int32
	GetSessionId() int64
}

type UserQueueInterface interface {
	UserBaseInterface
	GetQueuePosition() int32
}

type UserSceneInterface interface {
	UserBaseInterface
	GetSceneId() int32
}
