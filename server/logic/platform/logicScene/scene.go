package logicScene

import (
	"fmt"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

type Scene struct {
	mu                       sync.RWMutex
	SceneID                  int32
	TemplateID               int32
	players                  map[int64]*ScenePlayer
	playerNum                atomic.Int32
	InnerMessageTask         chan serviceInterface.InnerTaskInterface
	taskResp                 chan func()
	tickInterval             time.Duration
	innerMessageCountPerTick int32
	monitor                  *logicCommon.ThroughputMonitor
}

var _ logicCommon.SingleSceneProcessor = (*Scene)(nil)

func NewScene(id int32, templateID int32, sceneProcessorConfig *nodeConfig.MessageProcessConfig) *Scene {
	s := &Scene{
		SceneID:                  id,
		TemplateID:               templateID,
		players:                  make(map[int64]*ScenePlayer),
		taskResp:                 make(chan func(), sceneProcessorConfig.InnerMessageBufferSize),
		InnerMessageTask:         make(chan serviceInterface.InnerTaskInterface, sceneProcessorConfig.InnerMessageBufferSize),
		tickInterval:             sceneProcessorConfig.TickInterval,
		innerMessageCountPerTick: sceneProcessorConfig.InnerMessageCountPerTick,
		monitor:                  logicCommon.NewThroughputMonitor(enum.GetSceneProcessKey(templateID, id), 5*time.Second, 1*time.Minute),
	}
	s.playerNum.Store(0)
	return s
}

func (s *Scene) PushPlayerMessage(playerId int64, msgId int32, msg proto.Message, handler logicCommon.PlayerMessageHandler, function enum.FunctionIdEnum, isSceneTransferMessage bool) {
	sceneTask := NewPlayerTask(msgId, msg, handler, function, isSceneTransferMessage)

	s.mu.RLock()
	scenePlayer := s.players[playerId]
	s.mu.RUnlock()
	if scenePlayer == nil {
		platformLogger.ErrorWithUser(fmt.Sprintf("[scene] player not found playerId:%d,sceneId:%d,sceneTemplateId:%d", playerId, s.SceneID, s.TemplateID), nil, nil)
		return
	}

	select {
	case scenePlayer.TaskChan <- sceneTask:
		s.monitor.AddReceived(1)
	default:
		platformLogger.ErrorWithUser(fmt.Sprintf("[scene]player task chan full playerId:%d,sceneId:%d,sceneTemplateId:%d", playerId, s.SceneID, s.TemplateID), nil, nil)
	}
}

func (s *Scene) PushPlayerInnerTask(task serviceInterface.InnerTaskInterface) {
	s.mu.RLock()
	scenePlayer := s.players[task.GetReqId()]
	s.mu.RUnlock()
	if scenePlayer == nil {
		platformLogger.ErrorWithUser(fmt.Sprintf("[scene] player not found playerId:%d,sceneId:%d,sceneTemplateId:%d", task.GetReqId(), s.SceneID, s.TemplateID), nil, nil)
		return
	}

	select {
	case scenePlayer.InnerMessageTask <- task:
		s.monitor.AddReceived(1)
	default:
		platformLogger.ErrorWithUser(fmt.Sprintf("[scene] player inner message task chan full playerId:%d sceneId:%d,sceneTemplateId:%d", task.GetReqId(), s.SceneID, s.TemplateID), nil, nil)
	}
}

func (s *Scene) PushPlayerInnerResp(task serviceInterface.InnerTaskInterface, respHandle func()) {
	s.mu.RLock()
	scenePlayer := s.players[task.GetReqId()]
	s.mu.RUnlock()

	select {
	case scenePlayer.taskResp <- respHandle:
		s.monitor.AddReceived(1)
	default:
		platformLogger.ErrorWithUser(fmt.Sprintf("[scene] player inner message task resp chan full playerId:%d,sceneId:%d,sceneTemplateId:%d", task.GetReqId(), s.SceneID, s.TemplateID), nil, nil)
	}
}

func (s *Scene) PutSceneInnerTask(task serviceInterface.InnerTaskInterface) {
	select {
	case s.InnerMessageTask <- task:
		s.monitor.AddReceived(1)
	default:
		platformLogger.ErrorWithUser(fmt.Sprintf("inner message task chan full sceneId:%d,sceneTemplateId:%d", s.SceneID, s.TemplateID), nil, nil)
	}
}

func (s *Scene) PutSceneInnerResp(respHandle func()) {
	select {
	case s.taskResp <- respHandle:
		s.monitor.AddReceived(1)
	default:
		platformLogger.ErrorWithUser(fmt.Sprintf("inner message task resp chan full sceneId:%d,sceneTemplateId:%d", s.SceneID, s.TemplateID), nil, nil)
	}
}

func (s *Scene) Start() {
	go s.loop()
}

func (s *Scene) Stop() {

}

func (s *Scene) loop() {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	go s.monitor.Start()
	for {
		select {
		case <-ticker.C:
			s.processPlayerTasks()
			s.processInnerTask()
			s.processInnerTaskResp()
		}
	}
}

func (s *Scene) processPlayerTasks() {
	s.mu.RLock()
	tickPlayers := make([]*ScenePlayer, s.playerNum.Load())
	num := 0
	for _, player := range s.players {
		tickPlayers[num] = player
		num++
	}
	s.mu.RUnlock()

	var handled int32
	for _, player := range tickPlayers {
		handled += player.ProcessAllTask()
	}
	s.monitor.AddHandled(handled)
}

func (s *Scene) processInnerTask() {
	var handled int32

	for handled < s.innerMessageCountPerTick {
		select {
		case task := <-s.InnerMessageTask:
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[scene] handle inner message scene:%d templateId:%d panic:%+v", s.SceneID, s.TemplateID, r)
					}
				}()
				task.Resolve(task.ReqCall(task))
			}()

			handled++
		default:
			s.monitor.AddHandled(handled)
			return
		}
	}
	s.monitor.AddHandled(handled)
}

func (s *Scene) processInnerTaskResp() {
	var handled int32

	for handled < s.innerMessageCountPerTick {
		select {
		case resp := <-s.taskResp:
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[scene] handle inner task resp scene:%d templateId:%d panic:%+v", s.SceneID, s.TemplateID, r)
					}
				}()
				resp()
			}()

			handled++
			return
		default:
			s.monitor.AddHandled(handled)
			return
		}
	}
	s.monitor.AddHandled(handled)
}

func (s *Scene) EnterScene(scenePlayer *ScenePlayer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.InfoWithSprintf("[scene] player enter scene sceneId:%d,sceneTemplateId:%d,playerId:%d", s.SceneID, s.TemplateID, scenePlayer.UserID)
	scenePlayer.SceneId = s.SceneID
	scenePlayer.Status.Store(enum.PLAYER_SCENE_STATUS_TRANSFERING)
	s.players[scenePlayer.UserID] = scenePlayer
	s.playerNum.Add(1)
}

func (s *Scene) PlayerLoadOver(userID int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scenePlayer := s.players[userID]
	if scenePlayer == nil {
		return
	}
	scenePlayer.Status.Store(enum.PLAYER_SCENE_STATUS_RUNNING)
}

func (s *Scene) RemovePlayer(userID int64) *ScenePlayer {
	s.mu.Lock()
	defer s.mu.Unlock()

	player := s.players[userID]
	if player == nil {
		return nil
	}

	logger.InfoWithSprintf("[scene] player remove sceneId:%d,sceneTemplateId:%d,playerId:%d", s.SceneID, s.TemplateID, userID)
	player.SceneId = 0
	s.playerNum.Add(-1)
	player.Status.Store(enum.PLAYER_SCENE_STATUS_TRANSFERING)
	delete(s.players, userID)
	return player
}
