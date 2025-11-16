package model

import (
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/service/serviceInterface"
)

type UserModel struct {
	Account        string `gorm:"primaryKey;size:64"`
	UserId         int64
	LastLoginTime  int64
	LastLogoutTime int64
}

func (u *UserModel) TableName() string {
	return "account"
}

type LoginUser struct {
	session serviceInterface.SessionInterface
}

var _ logicInterface.UserBaseInterface = (*LoginUser)(nil)

func NewLoginUser(session serviceInterface.SessionInterface) *LoginUser {
	return &LoginUser{
		session: session,
	}
}

func (l *LoginUser) GetUserId() int64 {
	return 0
}

func (l *LoginUser) GetUserAccount() string {
	return "_login"
}

func (l *LoginUser) GetUserServerId() int32 {
	return 0
}
func (l *LoginUser) GetSession() serviceInterface.SessionInterface {
	return l.session
}

type QueueUser struct {
	userId   int64
	account  string
	queuePos int32
	serverId int32
	session  serviceInterface.SessionInterface
}

var _ logicInterface.UserQueueInterface = (*QueueUser)(nil)

func (q *QueueUser) GetUserId() int64 {
	return q.userId
}

func (q *QueueUser) GetUserAccount() string {
	return q.account
}

func (q *QueueUser) GetUserServerId() int32 {
	return q.serverId
}

func (q *QueueUser) GetQueuePosition() int32 {
	return q.queuePos
}

func (l *QueueUser) GetSession() serviceInterface.SessionInterface {
	return l.session
}

type SceneUser struct {
	userId   int64
	account  string
	serverId int32
	sceneId  int32
	session  serviceInterface.SessionInterface
}

var _ logicInterface.UserSceneInterface = (*SceneUser)(nil)

func (s *SceneUser) GetUserId() int64 {
	return s.userId
}

func (s *SceneUser) GetUserAccount() string {
	return s.account
}

func (s *SceneUser) GetUserServerId() int32 {
	return s.serverId
}

func (s *SceneUser) GetSceneId() int32 {
	return s.sceneId
}

func (l *SceneUser) GetSession() serviceInterface.SessionInterface {
	return l.session
}
