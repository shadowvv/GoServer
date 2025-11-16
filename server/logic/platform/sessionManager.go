package platform

import (
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"go.uber.org/zap"
	"sync"
)

type SessionManager struct {
	mu                 sync.RWMutex
	sessionCount       int32
	sessionMap         map[int64]serviceInterface.SessionInterface
	accountSessionMap  map[string]*UserSession
	playerIdSessionMap map[int64]*UserSession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessionMap:         make(map[int64]serviceInterface.SessionInterface),
		accountSessionMap:  make(map[string]*UserSession),
		playerIdSessionMap: make(map[int64]*UserSession),
	}
}

func (sm *SessionManager) OnConnectionTimeout(connectionInterface serviceInterface.SessionInterface) {
	// TODO: 断线处理
}

func (sm *SessionManager) Accept(connection serviceInterface.SessionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessionMap[connection.GetID()]; ok {
		logger.Error("[net] session already exist", zap.Int64("connectionId", connection.GetID()))
		connection.Close()
		return
	}
	sm.sessionMap[connection.GetID()] = connection
	sm.sessionCount++
	logger.Info("[net] new connection", zap.Int64("connectionId", connection.GetID()))
}

func (sm *SessionManager) Bind(userID int64, conn serviceInterface.SessionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if old, ok := sm.playerIdSessionMap[userID]; ok {
		old.Connection.Close()
		sm.sessionCount--
	}
	session := &UserSession{UserID: userID, Connection: conn}
	sm.playerIdSessionMap[userID] = session
	sm.sessionCount++
	logger.Info("[net] bind user", zap.Int64("userId", userID), zap.Int64("connectionId", conn.GetID()))
}

type UserSession struct {
	Account    string
	UserID     int64
	ServerId   int32
	Connection serviceInterface.SessionInterface
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
