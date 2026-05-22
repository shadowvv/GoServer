package dispatcherService

import (
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type SceneMessageProcessor struct {
	sessionManager         logicCommon.SessionManagerInterface
	messageFunctionMap     map[int32]enum.FunctionIdEnum
	messageHandlerMap      map[int32]logicCommon.PlayerMessageHandler
	innerMessageHandlerMap map[int32]logicCommon.InnerMessageHandler
	processors             map[int32]logicCommon.SingleSceneProcessor
}

var _ serviceInterface.MessageProcessorInterface = (*SceneMessageProcessor)(nil)

func NewSceneMessageProcessor(sessionManager logicCommon.SessionManagerInterface) *SceneMessageProcessor {
	processor := &SceneMessageProcessor{
		sessionManager:         sessionManager,
		messageFunctionMap:     make(map[int32]enum.FunctionIdEnum),
		messageHandlerMap:      make(map[int32]logicCommon.PlayerMessageHandler),
		innerMessageHandlerMap: make(map[int32]logicCommon.InnerMessageHandler),
		processors:             make(map[int32]logicCommon.SingleSceneProcessor),
	}
	return processor
}

func (s *SceneMessageProcessor) RegisterSceneProcessor(sceneId int32, processor logicCommon.SingleSceneProcessor) {
	if processor == nil {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterSceneProcessor processor is nil sceneId:%d", sceneId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := s.processors[sceneId]; ok {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterSceneProcessor sceneId:%d already registered", sceneId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	s.processors[sceneId] = processor
}

func (s *SceneMessageProcessor) RegisterPlayerMessageHandler(msgId int32, h logicCommon.PlayerMessageHandler, function enum.FunctionIdEnum) {
	if h == nil {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterPlayerMessageHandler handler is nil msgId:%d", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := s.messageHandlerMap[msgId]; ok {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterPlayerMessageHandler msgId:%d already registered", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	s.messageHandlerMap[msgId] = h
	s.messageFunctionMap[msgId] = function
}

func (l *SceneMessageProcessor) RegisterInnerMessageHandler(msgId int32, h logicCommon.InnerMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterInnerMessageHandler handler is nil msgId:%d", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := l.innerMessageHandlerMap[msgId]; ok {
		errorMessage := fmt.Sprintf("[sceneProcessor] RegisterInnerMessageHandler msgId:%d already registered", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	l.innerMessageHandlerMap[msgId] = h
}

func (s *SceneMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	h := s.messageHandlerMap[msgID]
	if h == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushMessage msgId:%d not registered", msgID))
		return
	}
	player := s.sessionManager.GetPlayerBasicInfoBySessionId(session.GetID())
	if player == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushMessage sessionID:%d not found", session.GetID()))
		return
	}
	isSceneTransfer := false
	if msgID == int32(pb.MESSAGE_ID_LOAD_SCENE_OVER_REQ) {
		isSceneTransfer = true
	}
	processor := s.processors[player.GetSceneId()]
	if processor == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushMessage sceneId:%d not registered", player.GetSceneId()))
		return
	}
	processor.PushPlayerMessage(player.GetUserId(), msgID, msg, h, s.messageFunctionMap[msgID], isSceneTransfer)
}

func (s *SceneMessageProcessor) PushSceneInnerTask(task *InnerTask) {
	if task == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushSceneInnerTask task is nil")
		return
	}
	task.ReqCallHandle = s.innerMessageHandlerMap[int32(task.MessageId)]
	if task.ReqCallHandle == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] no handler for msgId:%d", task.MessageId))
		return
	}
	processor := s.processors[int32(task.GetReqId())]
	if processor == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushSceneInnerTask sceneId:%d not registered", task.GetReqId()))
		return
	}
	processor.PutSceneInnerTask(task)
}

func (s *SceneMessageProcessor) PushSceneInnerResp(task *InnerTask, respHandle func()) {
	if task == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushSceneInnerResp task is nil")
		return
	}
	if respHandle == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushSceneInnerResp handle is nil")
		return
	}
	processor := s.processors[int32(task.GetReqId())]
	if processor == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushSceneInnerResp sceneId:%d not registered", task.GetReqId()))
		return
	}
	processor.PutSceneInnerResp(respHandle)
}

func (s *SceneMessageProcessor) PushPlayerInnerTask(task *InnerTask) {
	if task == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushPlayerInnerTask task is nil")
		return
	}
	task.ReqCallHandle = s.innerMessageHandlerMap[int32(task.MessageId)]
	if task.ReqCallHandle == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] no handler for msgId:%d", task.MessageId))
		return
	}
	player := s.sessionManager.GetPlayerBasicInfoByUserId(task.GetReqId())
	if player == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushPlayerInnerTask userId:%d not found", task.GetReqId()))
		return
	}
	processor := s.processors[player.GetSceneId()]
	if processor == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushPlayerInnerTask sceneId:%d not registered", player.GetSceneId()))
		return
	}
	processor.PushPlayerInnerTask(task)
}

func (s *SceneMessageProcessor) PushPlayerInnerResp(task *InnerTask, respHandle func()) {
	if task == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushPlayerInnerResp task is nil")
		return
	}
	if respHandle == nil {
		logger.ErrorWithZapFields("[sceneProcessor] PushPlayerInnerResp handle is nil")
		return
	}
	player := s.sessionManager.GetPlayerBasicInfoByUserId(task.GetReqId())
	if player == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushPlayerInnerResp userId:%d not found", task.GetReqId()))
		return
	}
	processor := s.processors[player.GetSceneId()]
	if processor == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[sceneProcessor] PushPlayerInnerResp sceneId:%d not registered", player.GetSceneId()))
		return
	}
	processor.PushPlayerInnerResp(task, respHandle)
}
