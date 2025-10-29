package platform

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"sync"
)

type SessionManager struct {
	mu                 sync.Mutex
	sessionCount       int32
	sessionMap         map[int64]serviceInterface.ConnectionInterface
	accountSessionMap  map[string]*UserSession
	playerIdSessionMap map[int64]*UserSession
}

func (sm *SessionManager) OnConnectionTimeout(connectionInterface serviceInterface.ConnectionInterface) {
	// TODO: 断线处理
}

func (sm *SessionManager) Accept(connection serviceInterface.ConnectionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessionMap[connection.GetID()] = connection
	sm.sessionCount++
}

func (sm *SessionManager) Bind(userID int64, conn serviceInterface.ConnectionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if old, ok := sm.playerIdSessionMap[userID]; ok {
		old.Connection.Close()
		sm.sessionCount--
		Info("unbind session", old)
	}
	session := &UserSession{UserID: userID, Connection: conn}
	sm.playerIdSessionMap[userID] = session
	sm.sessionCount++
	Info("bind session", session)
}
