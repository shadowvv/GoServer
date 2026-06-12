package dispatcherService

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type SidewayTask struct {
	user    *logicCommon.GatewayPlayerInfo
	msg     proto.Message
	handler logicCommon.SidewayMessageHandler
}

type SidewayProcessor struct {
	id                  int32
	tickInterval        time.Duration
	messageCountPerTick int32
	stopCh              chan struct{}
	tasks               chan *SidewayTask
	monitor             *logicCommon.ThroughputMonitor
}

func NewSidewayProcessor(id int32, tickInterval time.Duration, messageCountPerTick, messageBufferSize int32) *SidewayProcessor {
	return &SidewayProcessor{
		id:                  id,
		tickInterval:        tickInterval,
		messageCountPerTick: messageCountPerTick,
		stopCh:              make(chan struct{}),
		tasks:               make(chan *SidewayTask, messageBufferSize),
		monitor:             logicCommon.NewThroughputMonitor(enum.GetSidewayProcessKey(id), 5*time.Second, 1*time.Minute),
	}
}

func (p *SidewayProcessor) PushTask(user *logicCommon.GatewayPlayerInfo, msg proto.Message, handler logicCommon.SidewayMessageHandler) {
	task := &SidewayTask{
		user:    user,
		msg:     msg,
		handler: handler,
	}
	select {
	case p.tasks <- task:
		p.monitor.AddReceived(1)
	default:
		logger.ErrorBySprintf("[sidewayProcessor] local queue full processorId:%d", p.id)
	}
}

func (p *SidewayProcessor) start() {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	go p.monitor.Start()
	var lastTick time.Time

	for now := range ticker.C {
		if !lastTick.IsZero() {
			delay := now.Sub(lastTick)
			if delay > p.tickInterval*2 {
				logger.ErrorBySprintf("[sidewayProcessor] tick delayed processorId:%d delay=%v", p.id, delay)
			}
		}
		lastTick = now

		start := time.Now()

		p.processExternalTasks()

		cost := time.Since(start)
		if cost > p.tickInterval {
			logger.ErrorBySprintf("[sidewayProcessor] tick cost exceed processorId:%d cost=%v", p.id, cost)
		}
	}
}

func (p *SidewayProcessor) processExternalTasks() {
	var handled int32

	for handled < p.messageCountPerTick {
		select {
		case task := <-p.tasks:
			if task.handler == nil || task.msg == nil || task.user == nil {
				logger.InfoWithSprintf("[sidewayProcessor] invalid task processorId:%d", p.id)
				continue
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[sidewayProcessor] handle message processorId:%d panic:%+v", p.id, r)
					}
				}()
				task.handler(task.msg, task.user)
			}()
			handled++
		default:
			p.monitor.AddHandled(handled)
			return
		}
	}
	p.monitor.AddHandled(handled)
}

type SidewayMessageProcessor struct {
	sessionManager    logicCommon.SessionManagerInterface
	messageHandlerMap map[int32]logicCommon.SidewayMessageHandler
	processors        []*SidewayProcessor
	processorNum      int
}

var _ serviceInterface.MessageProcessorInterface = (*SidewayMessageProcessor)(nil)

func NewSidewayMessageProcessor(sessionManager logicCommon.SessionManagerInterface, config *nodeConfig.MessageProcessConfig) *SidewayMessageProcessor {
	logger.InfoWithSprintf("[sidewayProcessor] init sideway message processor config:%+v", config)
	processor := &SidewayMessageProcessor{
		sessionManager:    sessionManager,
		messageHandlerMap: make(map[int32]logicCommon.SidewayMessageHandler),
		processors:        make([]*SidewayProcessor, config.RoutineNum),
		processorNum:      int(config.RoutineNum),
	}
	for i := 0; i < processor.processorNum; i++ {
		processor.processors[i] = NewSidewayProcessor(int32(i), config.TickInterval, config.MessageCountPerTick, config.MessageBufferSize)
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

func (s *SidewayMessageProcessor) RegisterSidewayMessageHandler(msgID int32, h logicCommon.SidewayMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[sidewayProcessor] RegisterSidewayMessageHandler handler is nil msgId:%d", msgID)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := s.messageHandlerMap[msgID]; ok {
		errorMessage := fmt.Sprintf("[sidewayProcessor] RegisterSidewayMessageHandler msgId:%d already registered", msgID)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	s.messageHandlerMap[msgID] = h
}

func (s *SidewayMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	if session == nil {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessage nil session")
		return
	}
	baseUser := s.sessionManager.GetPlayerBasicInfoBySessionId(session.GetID())
	if baseUser == nil {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessage sessionID:%d not found", session.GetID())
		return
	}
	user, ok := baseUser.(*logicCommon.GatewayPlayerInfo)
	if !ok {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessage user type error sessionID:%d", session.GetID())
		return
	}
	s.PushMessageByUser(user, msgID, msg)
}

func (s *SidewayMessageProcessor) PushMessageByUser(user *logicCommon.GatewayPlayerInfo, msgID int32, msg proto.Message) {
	if user == nil {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessageByUser nil user")
		return
	}
	if user.GetSession() == nil {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessageByUser nil session userId:%d msgId:%d", user.UserId, msgID)
		return
	}
	if s.processorNum <= 0 {
		logger.ErrorBySprintf("[sidewayProcessor] PushMessageByUser invalid processorNum:%d", s.processorNum)
		return
	}
	sessionID := user.GetSession().GetID()
	index := tool.HashIndexByInt64(sessionID, s.processorNum)
	if index < 0 || index >= s.processorNum {
		logger.ErrorBySprintf("[sidewayProcessor] index out of range sessionId:%d,msgId:%d", sessionID, msgID)
		return
	}
	handler := s.messageHandlerMap[msgID]
	if handler == nil {
		logger.ErrorBySprintf("[sidewayProcessor] no handler for msgId:%d", msgID)
		return
	}
	s.processors[index].PushTask(user, msg, handler)
}
