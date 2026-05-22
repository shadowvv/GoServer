package logicScene

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
	"sync/atomic"
)

// 玩家场景结构
type ScenePlayer struct {
	UserID                   int64                                    // 用户ID
	SceneId                  int32                                    // 场景ID
	Status                   atomic.Int32                             // 玩家状态
	TaskChan                 chan *PlayerTask                         // 玩家任务队列
	InnerMessageTask         chan serviceInterface.InnerTaskInterface // 内部任务队列
	taskResp                 chan func()                              // 任务响应队列
	messageCountPerTick      int32                                    // 消息队列数量
	innerMessageCountPerTick int32                                    // 内部任务队列数量
	OfflineTime              atomic.Int64                             // 玩家离线时间
}

// 创建玩家场景结构
func NewScenePlayer(userID int64) *ScenePlayer {
	return &ScenePlayer{
		UserID:                   userID,
		Status:                   atomic.Int32{},
		TaskChan:                 make(chan *PlayerTask, playerProcessorConfig.MessageBufferSize),
		taskResp:                 make(chan func(), playerProcessorConfig.InnerMessageBufferSize),
		InnerMessageTask:         make(chan serviceInterface.InnerTaskInterface, playerProcessorConfig.InnerMessageBufferSize),
		messageCountPerTick:      playerProcessorConfig.MessageCountPerTick,
		innerMessageCountPerTick: playerProcessorConfig.InnerMessageCountPerTick,
	}
}

func (s *ScenePlayer) PushTask(task *PlayerTask) {
	select {
	case s.TaskChan <- task:
	default:
		logger.ErrorBySprintf("[scenePlayer] local queue full playerId:%d", s.UserID)
	}
}

func (s *ScenePlayer) PushInnerTask(task serviceInterface.InnerTaskInterface) {
	select {
	case s.InnerMessageTask <- task:
	default:
		logger.ErrorBySprintf("[scenePlayer] inner task queue full playerId:%d", s.UserID)
	}
}

func (s *ScenePlayer) PushInnerTaskResp(handler func()) {
	select {
	case s.taskResp <- handler:
	default:
		logger.ErrorBySprintf("[scenePlayer] inner task resp queue full playerId:%d", s.UserID)
	}
}

func (s *ScenePlayer) ProcessAllTask() int32 {
	var handled int32 = 0
	handled += s.processExternalTasks()
	handled += s.processInnerTasks()
	handled += s.processInnerRespTasks()
	if s.Status.Load() != enum.PLAYER_SCENE_STATUS_RUNNING {
		return handled
	}
	s.processHeartbeat()
	return handled
}

func (s *ScenePlayer) processHeartbeat() {
	func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorBySprintf("[scenePlayer] processHeartbeat playerId:%d panic:%+v", s.UserID, r)
			}
		}()
		player := sessionMgr.GetPlayerBasicInfoByUserId(s.UserID)
		if player == nil {
			return
		}
		playerModel := player.(*model.PlayerModel)
		if playerModel == nil {
			return
		}
		err := playerModel.Heartbeat(tool.UnixNowMilli())
		if err != nil {
			logger.ErrorBySprintf("[scenePlayer] heartbeat error playerId:%d", s.UserID)
			return
		}
		playerModel.CheckAndPushHeroChange()
		playerModel.SavePlayerToDB()
	}()
}

func (s *ScenePlayer) processExternalTasks() int32 {
	var handled int32

	player := sessionMgr.GetPlayerBasicInfoByUserId(s.UserID)
	if player == nil {
		return handled
	}
	playerModel := player.(*model.PlayerModel)
	if playerModel == nil {
		return handled
	}

	for handled < s.messageCountPerTick {
		select {
		case task := <-s.TaskChan:
			if task.Handler == nil || task.Msg == nil {
				logger.InfoWithSprintf("[scenePlayer] invalid task playerId:%d", s.UserID)
				continue
			}
			// only process scene message when player status is running
			if s.Status.Load() != enum.PLAYER_SCENE_STATUS_RUNNING {
				if !task.IsSceneTransferMessage && task.MsgId != int32(pb.MESSAGE_ID_HEART_REQ) && task.MsgId < int32(rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ID_BEGIN) {
					logger.ErrorBySprintf("[scenePlayer] task is Illegal playerId:%d,msgId:%d", s.UserID, task.MsgId)
					continue
				}
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[scenePlayer] handle message playerId:%d panic:%+v", s.UserID, r)
					}
				}()
				task.Handler(task.Msg, playerModel, task.FunctionId)
			}()

			handled++
		default:
			return handled
		}
	}
	return handled
}

func (s *ScenePlayer) processInnerTasks() int32 {
	var handled int32

	for handled < s.innerMessageCountPerTick {
		select {
		case task := <-s.InnerMessageTask:
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[scenePlayer] handle inner message playerId:%d panic:%+v", s.UserID, r)
					}
				}()
				task.Resolve(task.ReqCall(task))
			}()

			handled++
		default:
			return handled
		}
	}
	return handled
}

func (s *ScenePlayer) processInnerRespTasks() int32 {
	var handled int32

	for handled < s.innerMessageCountPerTick {
		select {
		case resp := <-s.taskResp:
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[scenePlayer] handle inner task response playerId:%d panic:%+v", s.UserID, r)
					}
				}()
				resp()
			}()

			handled++
		default:
			return handled
		}
	}
	return handled
}

// 玩家场景任务结构
type PlayerTask struct {
	MsgId                  int32                            // 消息ID
	Msg                    proto.Message                    // 消息
	Handler                logicCommon.PlayerMessageHandler // 消息处理接口
	FunctionId             enum.FunctionIdEnum              // 功能ID
	IsSceneTransferMessage bool                             // 是否是场景转移消息
}

// 创建玩家场景任务结构
func NewPlayerTask(msgId int32, msg proto.Message, handler logicCommon.PlayerMessageHandler, function enum.FunctionIdEnum, sceneTransferMessage bool) *PlayerTask {
	return &PlayerTask{
		MsgId:                  msgId,
		Msg:                    msg,
		Handler:                handler,
		FunctionId:             function,
		IsSceneTransferMessage: sceneTransferMessage,
	}
}
