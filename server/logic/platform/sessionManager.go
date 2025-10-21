package platform

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"sync"
)

type UserSession struct {
	Account    string
	UserID     int64
	Connection serviceInterface.ConnectionInterface
}

type SessionManager struct {
	sessions sync.Map // map[int64]*UserSession
}

func (sm *SessionManager) Accept(connection serviceInterface.ConnectionInterface) {

}

func (sm *SessionManager) Bind(userID int64, conn serviceInterface.ConnectionInterface) {
	if old, ok := sm.sessions.Load(userID); ok {
		// 断开旧连接
		old.(*UserSession).Connection.Close()
	}
	session := &UserSession{UserID: userID, Connection: conn}
	sm.sessions.Store(userID, session)
}
