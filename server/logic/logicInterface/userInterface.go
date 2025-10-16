package logicInterface

type UserBaseInterface interface {
	GetUserId() int64
	GetUserAccount() string
	GetUserServerId() int32
}

type BasicUserInfo struct {
	UserId       int64
	UserAccount  string
	UserServerId int32
}

func (b *BasicUserInfo) GetUserId() int64 {
	return 0
}

func (b *BasicUserInfo) GetUserAccount() string {
	return "test"
}

func (b *BasicUserInfo) GetUserServerId() int32 {
	return 0
}

type UserQueueInterface interface {
	UserBaseInterface
	GetQueuePosition() int32
}

type UserSceneInterface interface {
	UserBaseInterface
	GetSceneId() int32
}
