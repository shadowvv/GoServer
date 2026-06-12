package logicSessionManager

import (
	"context"
	"errors"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/dbService"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"go.uber.org/zap"
)

type GatewaySessionManager struct {
	mu sync.RWMutex
	// sessionId -> session
	sessionMap map[int64]serviceInterface.SessionInterface
	// sessionId -> playerInfo
	sessionPlayerMap map[int64]*logicCommon.GatewayPlayerInfo
	// account -> playerInfo
	UserIdPlayerMap map[int64]*logicCommon.GatewayPlayerInfo
	// serverId ->sessionId-> playerInfo
	serverIdSessionMap map[int32]map[int64]*logicCommon.GatewayPlayerInfo
}

func NewGatewaySessionManager() *GatewaySessionManager {
	return &GatewaySessionManager{
		mu:                 sync.RWMutex{},
		sessionMap:         make(map[int64]serviceInterface.SessionInterface),
		sessionPlayerMap:   make(map[int64]*logicCommon.GatewayPlayerInfo),
		UserIdPlayerMap:    make(map[int64]*logicCommon.GatewayPlayerInfo),
		serverIdSessionMap: make(map[int32]map[int64]*logicCommon.GatewayPlayerInfo),
	}
}

var _ serviceInterface.AcceptorInterface = (*GatewaySessionManager)(nil)
var _ logicCommon.SessionManagerInterface = (*GatewaySessionManager)(nil)

func (sm *GatewaySessionManager) Accept(session serviceInterface.SessionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessionMap[session.GetID()]; ok {
		logger.ErrorWithZapFields("[net] session already exist", zap.Int64("sessionId", session.GetID()))
		session.Close()
		return
	}
	sm.sessionMap[session.GetID()] = session
	sm.sessionPlayerMap[session.GetID()] = &logicCommon.GatewayPlayerInfo{
		Account:  "_login",
		ServerId: 0,
		NodeId:   0,
		Session:  session,
	}
	logger.InfoWithZapFields("[net] accept new session", zap.Int64("sessionId", session.GetID()))
}

func (sm *GatewaySessionManager) BindPlayerWithNode(user *logicCommon.GatewayPlayerInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检测已accept
	if _, ok := sm.sessionMap[user.Session.GetID()]; !ok {
		logger.ErrorWithZapFields("[net] session not exist", zap.Int64("sessionId", user.Session.GetID()))
		return errors.New("session not exist")
	}
	if _, ok := sm.sessionPlayerMap[user.Session.GetID()]; !ok {
		logger.ErrorWithZapFields("[net] session already bind account", zap.String("account", user.Account))
		return errors.New("session already bind account")
	}

	// 检测未绑定
	if _, ok := sm.UserIdPlayerMap[user.UserId]; ok {
		logger.ErrorWithZapFields("[net] userId already exist", zap.Int64("userId", user.UserId))
		return errors.New("[net] userId already exist")
	}

	sm.addPlayer(user)
	return nil
}

func (sm *GatewaySessionManager) addPlayer(user *logicCommon.GatewayPlayerInfo) {
	sm.sessionPlayerMap[user.Session.GetID()] = user
	if sm.serverIdSessionMap[user.ServerId] == nil {
		sm.serverIdSessionMap[user.ServerId] = make(map[int64]*logicCommon.GatewayPlayerInfo)
	}
	sm.UserIdPlayerMap[user.UserId] = user
	sm.serverIdSessionMap[user.ServerId][user.Session.GetID()] = user
	logger.InfoWithZapFields("[net] bind with account", zap.Int64("sessionId", user.Session.GetID()), zap.String("account", user.Account), zap.Int32("serverId", user.ServerId), zap.Int32("nodeId", user.NodeId))

	_ = dbService.RDB.SetEX(context.Background(), enum.GetOnlinePlayerKey(user.GetUserServerId()), len(sm.serverIdSessionMap[user.ServerId]), 0)
}

func (sm *GatewaySessionManager) ReplacePlayerWithNewInfo(newUser *logicCommon.GatewayPlayerInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检测已accept
	if _, ok := sm.sessionMap[newUser.Session.GetID()]; !ok {
		logger.ErrorWithZapFields("[net] session not exist", zap.Int64("sessionId", newUser.Session.GetID()))
		return errors.New("session not exist")
	}
	if _, ok := sm.sessionPlayerMap[newUser.Session.GetID()]; !ok {
		logger.ErrorWithZapFields("[net] session already bind account", zap.String("account", newUser.Account))
		return errors.New("session already bind account")
	}

	sm.addPlayer(newUser)
	return nil
}

func (sm *GatewaySessionManager) OnSessionClose(session serviceInterface.SessionInterface, isForce bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.removePlayer(session)
}

func (sm *GatewaySessionManager) removePlayer(session serviceInterface.SessionInterface) {
	sm.sessionMap[session.GetID()] = nil
	delete(sm.sessionMap, session.GetID())

	if player, ok := sm.sessionPlayerMap[session.GetID()]; ok {
		sm.sessionPlayerMap[session.GetID()] = nil
		delete(sm.sessionPlayerMap, session.GetID())

		if sm.serverIdSessionMap[player.ServerId] != nil && sm.serverIdSessionMap[player.ServerId][player.Session.GetID()] != nil {
			sm.serverIdSessionMap[player.ServerId][player.Session.GetID()] = nil
			delete(sm.serverIdSessionMap[player.ServerId], session.GetID())
		}
		sm.UserIdPlayerMap[player.UserId] = nil
		delete(sm.UserIdPlayerMap, player.UserId)

		_ = dbService.RDB.SetEX(context.Background(), enum.GetOnlinePlayerKey(player.ServerId), len(sm.serverIdSessionMap[player.ServerId]), 0)
	}

	logger.InfoWithZapFields("[net] session timeout", zap.Int64("sessionId", session.GetID()))
}

func (sm *GatewaySessionManager) GetPlayerBasicInfoBySessionId(sessionId int64) logicCommon.UserBaseInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if _, ok := sm.sessionPlayerMap[sessionId]; !ok {
		return nil
	}
	player := sm.sessionPlayerMap[sessionId]
	return player
}

func (sm *GatewaySessionManager) GetPlayerBasicInfoByUserId(userId int64) logicCommon.UserBaseInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if _, ok := sm.UserIdPlayerMap[userId]; !ok {
		return nil
	}
	player := sm.UserIdPlayerMap[userId]
	return player
}

func (sm *GatewaySessionManager) GetSessionById(sessionId int64) serviceInterface.SessionInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.sessionMap[sessionId]
}

func (sm *GatewaySessionManager) GetPlayerSessionsByServerId(serverId int32) map[int64]*logicCommon.GatewayPlayerInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.serverIdSessionMap[serverId]
}

func (sm *GatewaySessionManager) GetPlayerSessionsByUserIds(userId []int64) map[int64]*logicCommon.GatewayPlayerInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	players := make(map[int64]*logicCommon.GatewayPlayerInfo)
	for _, playerId := range userId {
		if _, ok := sm.UserIdPlayerMap[playerId]; !ok {
			continue
		}
		player := sm.UserIdPlayerMap[playerId]
		players[player.UserId] = player
	}

	return players
}

func (sm *GatewaySessionManager) GetPlayerByUserId(useId int64) *logicCommon.GatewayPlayerInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if _, ok := sm.UserIdPlayerMap[useId]; !ok {
		return nil
	}

	return sm.UserIdPlayerMap[useId]
}

func (sm *GatewaySessionManager) KickOutNodePlayer(nodeId int32, hooker func(player logicCommon.UserBaseInterface)) {
	sm.mu.RLock()
	players := make([]*logicCommon.GatewayPlayerInfo, 0)
	for _, player := range sm.sessionPlayerMap {
		if player.NodeId == nodeId {
			players = append(players, player)
		}
	}
	sm.mu.RUnlock()

	for _, player := range players {
		if player.NodeId == nodeId {
			hooker(player)
		}
	}
}

func (sm *GatewaySessionManager) KickOutPlayer(playerId int64, hooker func(player logicCommon.UserBaseInterface)) {
	sm.mu.RLock()
	player := sm.UserIdPlayerMap[playerId]
	sm.mu.RUnlock()
	if player == nil {
		logger.ErrorBySprintf("player %d is not online", playerId)
		return
	}
	hooker(player)
}

func (sm *GatewaySessionManager) KickOutServerPlayer(serverId int32, hooker func(player logicCommon.UserBaseInterface)) {
	sm.mu.RLock()
	players := make([]*logicCommon.GatewayPlayerInfo, 0)
	for _, player := range sm.serverIdSessionMap[serverId] {
		players = append(players, player)
	}
	sm.mu.RUnlock()

	for _, player := range players {
		hooker(player)
	}
}

func (sm *GatewaySessionManager) KickOutAllPlayer(hooker func(player logicCommon.UserBaseInterface)) {
	sm.mu.RLock()
	players := make([]*logicCommon.GatewayPlayerInfo, 0)
	for _, player := range sm.sessionPlayerMap {
		players = append(players, player)
	}
	sm.mu.RUnlock()

	for _, player := range players {
		hooker(player)
	}
}
