package platform

import "github.com/drop/GoServer/server/service/serviceInterface"

type UserSession struct {
	Account    string
	UserID     int64
	ServerId   int32
	Connection serviceInterface.ConnectionInterface
}

func (u *UserSession) GetSessionId() int64 {
	return u.Connection.GetID()
}

func (u *UserSession) GetUserId() int64 {
	return u.UserID
}

func (u *UserSession) GetUserAccount() string {
	return u.Account
}

func (u *UserSession) GetUserServerId() int32 {
	return u.ServerId
}
