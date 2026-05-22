package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/service/serviceInterface"
)

type LoginUser struct {
	Account  string
	ServerId int32
	UserId   int64
	NodeId   int32
	session  serviceInterface.SessionInterface
}

var _ logicCommon.UserBaseInterface = (*LoginUser)(nil)

func NewLoginUser(session serviceInterface.SessionInterface) *LoginUser {
	return &LoginUser{
		session: session,
		Account: "_login",
	}
}

func (l *LoginUser) GetUserId() int64 {
	return l.UserId
}

func (l *LoginUser) GetNodeId() int32 {
	return l.NodeId
}

func (l *LoginUser) GetUserAccount() string {
	return l.Account
}

func (l *LoginUser) GetUserServerId() int32 {
	return l.ServerId
}
func (l *LoginUser) GetSession() serviceInterface.SessionInterface {
	return l.session
}

// 登录时还没有分配场景
func (l *LoginUser) GetSceneId() int32 {
	return 0
}
