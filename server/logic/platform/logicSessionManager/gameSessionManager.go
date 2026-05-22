package logicSessionManager

import (
	"sync"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"go.uber.org/zap"
)

type GameSessionManager struct {
	mu sync.RWMutex
	// session关闭钩子
	hooker logicCommon.SessionCloseHooker
	// sessionId -> session
	sessionMap map[int64]serviceInterface.SessionInterface
	// sessionId -> player
	sessionPlayerMap map[int64]*model.PlayerModel
	// playerId -> player
	playerIdPlayerMap map[int64]*model.PlayerModel
}

func NewGameSessionManager(hooker logicCommon.SessionCloseHooker) *GameSessionManager {
	return &GameSessionManager{
		mu:                sync.RWMutex{},
		hooker:            hooker,
		sessionMap:        make(map[int64]serviceInterface.SessionInterface),
		sessionPlayerMap:  make(map[int64]*model.PlayerModel),
		playerIdPlayerMap: make(map[int64]*model.PlayerModel),
	}
}

var _ serviceInterface.AcceptorInterface = (*GameSessionManager)(nil)

func (sm *GameSessionManager) Accept(session serviceInterface.SessionInterface) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessionMap[session.GetID()]; ok {
		logger.ErrorWithZapFields("[net] session already exist", zap.Int64("sessionId", session.GetID()))
		return
	}
	sm.sessionMap[session.GetID()] = session
	logger.InfoWithZapFields("[net] new session", zap.Int64("sessionId", session.GetID()))
}

func (sm *GameSessionManager) BindWithSession(user logicCommon.UserBaseInterface, player *model.PlayerModel) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	player.Session = user.GetSession()
	sm.sessionPlayerMap[player.Session.GetID()] = player
	sm.playerIdPlayerMap[player.GetUserId()] = player
	platformLogger.InfoWithUser("[net] bind user with session", player)
}

func (sm *GameSessionManager) ReplaceSession(session serviceInterface.SessionInterface, player *model.PlayerModel) {
	sm.mu.Lock()
	delete(sm.sessionPlayerMap, player.GetSession().GetID())
	delete(sm.sessionMap, player.GetSession().GetID())
	sm.sessionPlayerMap[session.GetID()] = player
	sm.sessionMap[session.GetID()] = session
	player.Session = session
	platformLogger.InfoWithUser("[net] replace session", player)
	sm.mu.Unlock()
}

func (sm *GameSessionManager) OnConnectionTimeout(session serviceInterface.SessionInterface) {
	sm.mu.Lock()
	if _, ok := sm.sessionMap[session.GetID()]; ok {
		delete(sm.sessionMap, session.GetID())
	}
	sm.mu.Unlock()
	if player, ok := sm.sessionPlayerMap[session.GetID()]; ok {
		sm.hooker.OnSessionClose(player)
	}
	logger.InfoWithZapFields("[net] session timeout", zap.Int64("sessionId", session.GetID()))
}

// RangeOnlinePlayers 遍历当前节点在线玩家
func (sm *GameSessionManager) RangeOnlinePlayers(f func(*model.PlayerModel)) {
	sm.mu.RLock()
	snapshot := make([]*model.PlayerModel, 0, len(sm.playerIdPlayerMap))
	for _, p := range sm.playerIdPlayerMap {
		snapshot = append(snapshot, p)
	}
	sm.mu.RUnlock()
	for _, p := range snapshot {
		if p != nil {
			f(p)
		}
	}
}

func (sm *GameSessionManager) GetSessionById(id int64) serviceInterface.SessionInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if _, ok := sm.sessionMap[id]; !ok {
		return nil
	}

	return sm.sessionMap[id]
}

func (sm *GameSessionManager) GetPlayerBasicInfoByAccount(account string) logicCommon.UserBaseInterface {
	return nil
}

func (sm *GameSessionManager) GetPlayerBasicInfoBySessionId(sessionId int64) logicCommon.UserBaseInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if player, ok := sm.sessionPlayerMap[sessionId]; ok {
		return player
	}
	return nil
}

func (sm *GameSessionManager) GetPlayerBasicInfoByUserId(userId int64) logicCommon.UserBaseInterface {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if player, ok := sm.playerIdPlayerMap[userId]; ok {
		return player
	}
	return nil
}

func (sm *GameSessionManager) RemovePlayer(player *model.PlayerModel) {
	sm.mu.Lock()
	if player.GetSession() != nil {
		delete(sm.sessionMap, player.GetSession().GetID())
		delete(sm.sessionPlayerMap, player.GetSession().GetID())
	}
	delete(sm.playerIdPlayerMap, player.GetUserId())
	platformLogger.InfoWithUser("[net] remove user", player)
	sm.mu.Unlock()
}
