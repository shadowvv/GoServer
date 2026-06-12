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

//TODO: 后续优化
//session → user 查询在 PushMessage 阶段做（可优化）
//user := g.sessionManager.GetPlayerBasicInfoBySessionId(...)
//问题不是“错”，而是：
//PushMessage 阶段做 IO / map / lock
//processor 设计初衷是 把所有逻辑推到 processor 内
//PushMessage 只负责路由
//查 user / handler 在 processor 内完成
//这样可以把 sessionManager 的锁争用集中到 processor。

type GatewayTask struct {
	user    *logicCommon.GatewayPlayerInfo
	msg     proto.Message
	handler logicCommon.GatewayMessageHandler
}

type GatewayProcessor struct {
	id                  int32
	tickInterval        time.Duration
	messageCountPerTick int32
	stopCh              chan struct{}
	tasks               chan *GatewayTask
	monitor             *logicCommon.ThroughputMonitor
}

func NewGatewayProcessor(id int32, tickInterval time.Duration, messageCountPerTick, messageBufferSize int32) *GatewayProcessor {
	processor := &GatewayProcessor{
		id:                  id,
		tickInterval:        tickInterval,
		messageCountPerTick: messageCountPerTick,
		stopCh:              make(chan struct{}),
		tasks:               make(chan *GatewayTask, messageBufferSize),
	}
	processor.monitor = logicCommon.NewThroughputMonitor(enum.GetGatewayProcessKey(id), 5*time.Second, 1*time.Minute)
	return processor
}

func (p *GatewayProcessor) PushTask(user logicCommon.UserBaseInterface, msg proto.Message, handler logicCommon.GatewayMessageHandler) {
	gatewayUser, ok := user.(*logicCommon.GatewayPlayerInfo)
	if !ok {
		logger.ErrorBySprintf("[gatewayProcessor] PushMessage user is not gatewayPlayerInfo processorId:%d", p.id)
		return
	}
	task := &GatewayTask{
		user:    gatewayUser,
		msg:     msg,
		handler: handler,
	}
	select {
	case p.tasks <- task:
		p.monitor.AddReceived(1)
	default:
		logger.ErrorBySprintf("[gatewayProcessor] local queue full processorId:%d", p.id)
	}
}

func (p *GatewayProcessor) start() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	go p.monitor.Start()
	var lastTick time.Time

	for now := range ticker.C {
		if !lastTick.IsZero() {
			delay := now.Sub(lastTick)
			if delay > p.tickInterval*2 {
				logger.ErrorBySprintf("[gatewayProcessor] tick delayed processorId:%d delay=%v", p.id, delay)
			}
		}
		lastTick = now

		start := time.Now()
		handled := int32(0)

		for handled < p.messageCountPerTick {
			select {
			case task := <-p.tasks:
				if task.handler == nil {
					logger.ErrorBySprintf("[gatewayProcessor] PushMessage handler is nil processorId:%d", p.id)
					continue
				}
				if task.msg == nil || task.user == nil {
					logger.ErrorBySprintf("[gatewayProcessor] PushMessage user or msg is nil processorId:%d", p.id)
					continue
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.ErrorBySprintf("[gatewayProcessor] handle message processorId:%d panic:%+v", p.id, r)
						}
					}()
					task.handler(task.msg, task.user)
				}()
				handled++
			case <-p.stopCh:
				p.monitor.Stop()
				logger.InfoWithSprintf("[gatewayProcessor] stop processorId:%d", p.id)
				return
			default:
				goto END
			}
		}

	END:
		p.monitor.AddHandled(handled)
		cost := time.Since(start)
		if cost > p.tickInterval {
			logger.ErrorBySprintf("[gatewayProcessor] tick cost exceed processorId:%d cost=%v handled=%d", p.id, cost, handled)
		}
	}
}

type GatewayMessageProcessor struct {
	sessionManager    logicCommon.SessionManagerInterface
	messageHandlerMap map[int32]logicCommon.GatewayMessageHandler
	processors        []*GatewayProcessor
	processorNum      int
}

var _ serviceInterface.MessageProcessorInterface = (*GatewayMessageProcessor)(nil)

func NewGatewayMessageProcess(sessionManager logicCommon.SessionManagerInterface, config *nodeConfig.MessageProcessConfig) *GatewayMessageProcessor {
	logger.InfoWithSprintf("[gatewayProcessor] init gateway message processor config:%+v", config)
	processor := &GatewayMessageProcessor{
		sessionManager:    sessionManager,
		messageHandlerMap: make(map[int32]logicCommon.GatewayMessageHandler),
		processors:        make([]*GatewayProcessor, config.RoutineNum),
		processorNum:      int(config.RoutineNum),
	}
	for i := 0; i < processor.processorNum; i++ {
		processor.processors[i] = NewGatewayProcessor(int32(i), config.TickInterval, config.MessageCountPerTick, config.MessageBufferSize)
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

func (g *GatewayMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgId int32, msg proto.Message) {
	if session == nil {
		logger.ErrorBySprintf("[gatewayProcessor] PushMessage nil session")
		return
	}
	index := tool.HashIndexByInt64(session.GetID(), g.processorNum)
	if index < 0 || index >= g.processorNum {
		logger.ErrorBySprintf("[gatewayProcessor] index out of range sessionId:%d,msgId:%d", session.GetID(), msgId)
		return
	}
	handler := g.messageHandlerMap[msgId]
	if handler == nil {
		logger.ErrorBySprintf("[gatewayProcessor] no handler for msgId:%d", msgId)
		return
	}
	user := g.sessionManager.GetPlayerBasicInfoBySessionId(session.GetID())
	if user == nil {
		logger.ErrorBySprintf("[gatewayProcessor] PushMessage sessionID:%d not found", session.GetID())
		return
	}
	g.processors[index].PushTask(user, msg, handler)
}

func (g *GatewayMessageProcessor) RegisterGatewayMessageHandler(msgType int32, h logicCommon.GatewayMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[processor] RegisterGatewayMessageHandler handler is nil msgType:%d", msgType)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := g.messageHandlerMap[msgType]; ok {
		errorMessage := fmt.Sprintf("[processor] RegisterGatewayMessageHandler msgType:%d already registered", msgType)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	g.messageHandlerMap[msgType] = h
	logger.InfoWithSprintf("[processor] RegisterGatewayMessageHandler msgType:%d", msgType)
}
