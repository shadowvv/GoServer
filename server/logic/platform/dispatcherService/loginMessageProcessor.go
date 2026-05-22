package dispatcherService

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type LoginTask struct {
	user    *model.LoginUser
	msg     proto.Message
	handler logicCommon.LoginMessageHandler
}

type LoginProcessor struct {
	id                  int32
	tickInterval        time.Duration
	messageCountPerTick int32
	stopCh              chan struct{}
	tasks               chan *LoginTask
	monitor             *logicCommon.ThroughputMonitor
}

func NewLoginProcessor(id int32, tickInterval time.Duration, messageCountPerTick, messageBufferSize int32) *LoginProcessor {
	return &LoginProcessor{
		id:                  id,
		tickInterval:        tickInterval,
		messageCountPerTick: messageCountPerTick,
		stopCh:              make(chan struct{}),
		tasks:               make(chan *LoginTask, messageBufferSize),
		monitor:             logicCommon.NewThroughputMonitor(enum.GetLoginProcessKey(id), 5*time.Second, 1*time.Minute),
	}
}

func (p *LoginProcessor) PushTask(session serviceInterface.SessionInterface, msgID int32, msg proto.Message, handler logicCommon.LoginMessageHandler) {
	gameSession, ok := session.(*logicSessionManager.GameSession)
	if !ok {
		logger.ErrorBySprintf("[loginProcessor] login session error")
		return
	}
	task := &LoginTask{
		user:    model.NewLoginUser(session),
		msg:     msg,
		handler: handler,
	}
	task.user.UserId = gameSession.UserId
	select {
	case p.tasks <- task:
		p.monitor.AddReceived(1)
	default:
		logger.ErrorBySprintf("[loginProcessor] local queue full processorId:%d", p.id)
	}
}

func (p *LoginProcessor) start() {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	go p.monitor.Start()
	var lastTick time.Time

	for now := range ticker.C {
		if !lastTick.IsZero() {
			delay := now.Sub(lastTick)
			if delay > p.tickInterval*2 {
				logger.ErrorBySprintf("[loginProcessor] tick delayed processorId:%d delay=%v", p.id, delay)
			}
		}
		lastTick = now

		start := time.Now()

		p.processExternalTasks()

		cost := time.Since(start)
		if cost > p.tickInterval {
			logger.ErrorBySprintf("[loginProcessor] tick cost exceed processorId:%d cost=%v", p.id, cost)
		}
	}
}
func (p *LoginProcessor) processExternalTasks() {
	var handled int32

	for handled < p.messageCountPerTick {
		select {
		case task := <-p.tasks:
			if task.handler == nil || task.msg == nil || task.user == nil {
				logger.InfoWithSprintf("[loginProcessor] invalid task processorId:%d", p.id)
				continue
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorBySprintf("[loginProcessor] handle message processorId:%d panic:%+v", p.id, r)
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

type LoginMessageProcessor struct {
	messageHandlerMap      map[int32]logicCommon.LoginMessageHandler
	innerMessageHandlerMap map[int32]logicCommon.InnerMessageHandler
	processors             []*LoginProcessor
	processorNum           int
}

var _ serviceInterface.MessageProcessorInterface = (*LoginMessageProcessor)(nil)

func NewLoginMessageProcessor(config *nodeConfig.MessageProcessConfig) *LoginMessageProcessor {
	logger.InfoWithSprintf("[loginProcessor] init login message processor config:%+v", config)
	processor := &LoginMessageProcessor{
		messageHandlerMap:      make(map[int32]logicCommon.LoginMessageHandler),
		innerMessageHandlerMap: make(map[int32]logicCommon.InnerMessageHandler),
		processors:             make([]*LoginProcessor, config.RoutineNum),
		processorNum:           int(config.RoutineNum),
	}
	for i := 0; i < processor.processorNum; i++ {
		processor.processors[i] = NewLoginProcessor(int32(i), config.TickInterval, config.MessageCountPerTick, config.MessageBufferSize)
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

func (l *LoginMessageProcessor) RegisterLoginMessageHandler(msgId int32, h logicCommon.LoginMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[loginProcessor] RegisterLoginMessageHandler handler is nil msgId:%d", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := l.messageHandlerMap[msgId]; ok {
		errorMessage := fmt.Sprintf("[loginProcessor] RegisterLoginMessageHandler msgId:%d already registered", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	l.messageHandlerMap[msgId] = h
}

func (l *LoginMessageProcessor) RegisterInnerMessageHandler(msgId int32, h logicCommon.InnerMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[loginProcessor] RegisterInnerMessageHandler handler is nil msgId:%d", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := l.innerMessageHandlerMap[msgId]; ok {
		errorMessage := fmt.Sprintf("[loginProcessor] RegisterInnerMessageHandler msgId:%d already registered", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	l.innerMessageHandlerMap[msgId] = h
}

func (l *LoginMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	if session == nil {
		logger.ErrorBySprintf("[loginProcessor] PushMessage nil session")
		return
	}
	index := int(session.GetID() % int64(l.processorNum))
	if index < 0 || index >= l.processorNum {
		logger.ErrorBySprintf("[loginProcessor] login index out of range sessionId:%d,msgId:%d", session.GetID(), msgID)
		return
	}
	handler := l.messageHandlerMap[msgID]
	if handler == nil {
		logger.ErrorBySprintf("[loginProcessor] no handler for msgId:%d", msgID)
		return
	}
	l.processors[index].PushTask(session, msgID, msg, handler)
}
