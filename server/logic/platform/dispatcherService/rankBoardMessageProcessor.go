package dispatcherService

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type RankBoardTask struct {
	session serviceInterface.SessionInterface
	msg     proto.Message
	rankId  string
	handler logicCommon.RankBoardMessageHandler
}
type RankBoardProcessor struct {
	id                  int32
	tickInterval        time.Duration
	messageCountPerTick int32
	stopCh              chan struct{}
	tasks               chan *RankBoardTask
	monitor             *logicCommon.ThroughputMonitor
}

func NewRankBoardProcessor(id int32, tickInterval time.Duration, messageCountPerTick, messageBufferSize int32) *RankBoardProcessor {
	return &RankBoardProcessor{
		id:                  id,
		tickInterval:        tickInterval,
		messageCountPerTick: messageCountPerTick,
		stopCh:              make(chan struct{}),
		tasks:               make(chan *RankBoardTask, messageBufferSize),
		monitor:             logicCommon.NewThroughputMonitor(enum.GetRankBoardProcessKey(id), 5*time.Second, 1*time.Minute),
	}
}

func (p *RankBoardProcessor) PutTask(session serviceInterface.SessionInterface, msgID int32, rankId string, msg proto.Message, handler logicCommon.RankBoardMessageHandler) {
	task := &RankBoardTask{
		session: session,
		msg:     msg,
		rankId:  rankId,
		handler: handler,
	}
	select {
	case p.tasks <- task:
		p.monitor.AddReceived(1)
	default:
		logger.ErrorWithZapFields(fmt.Sprintf("[rankBoardProcessor] local queue full for session:%d msg:%d", session.GetID(), msgID))
	}
}

func (p *RankBoardProcessor) start() {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	go p.monitor.Start()
	var lastTick time.Time

	for now := range ticker.C {
		if !lastTick.IsZero() {
			delay := now.Sub(lastTick)
			if delay > p.tickInterval*2 {
				logger.ErrorBySprintf("[rankBoardProcessor] tick delayed processorId:%d delay=%v", p.id, delay)
			}
		}
		lastTick = now

		start := time.Now()
		handled := int32(0)

		for handled < p.messageCountPerTick {
			select {
			case task := <-p.tasks:
				if task.handler == nil {
					logger.ErrorBySprintf("[rankBoardProcessor] PushMessage handler is nil processorId:%d", p.id)
					continue
				}
				if task.msg == nil {
					logger.ErrorBySprintf("[rankBoardProcessor] PushMessage user or msg is nil processorId:%d", p.id)
					continue
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.ErrorBySprintf("[rankBoardProcessor] handle message processorId:%d panic:%+v", p.id, r)
						}
					}()
					task.handler(task.msg, task.rankId, task.session)
				}()
				handled++
			case <-p.stopCh:
				p.monitor.Stop()
				logger.InfoWithSprintf("[rankBoardProcessor] stop processorId:%d", p.id)
				return
			default:
				goto END
			}
		}

	END:
		cost := time.Since(start)
		p.monitor.AddHandled(handled)
		if cost > p.tickInterval {
			logger.ErrorBySprintf("[rankBoardProcessor] tick cost exceed processorId:%d cost=%v handled=%d", p.id, cost, handled)
		}
	}
}

type RankBoardMessageProcessor struct {
	messageHandlerMap map[int32]logicCommon.RankBoardMessageHandler
	processors        []*RankBoardProcessor
	processorNum      int
}

var _ serviceInterface.MessageProcessorInterface = (*RankBoardMessageProcessor)(nil)

func NewRankBoardMessageProcessor(config *nodeConfig.MessageProcessConfig) *RankBoardMessageProcessor {
	logger.InfoWithSprintf("[rankBoardProcessor] init rankBoard message processor config:%+v", config)
	processor := &RankBoardMessageProcessor{
		messageHandlerMap: make(map[int32]logicCommon.RankBoardMessageHandler),
		processors:        make([]*RankBoardProcessor, config.RoutineNum),
		processorNum:      int(config.RoutineNum),
	}
	for i := 0; i < processor.processorNum; i++ {
		processor.processors[i] = NewRankBoardProcessor(int32(i), config.TickInterval, config.MessageCountPerTick, config.MessageBufferSize)
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

func (l *RankBoardMessageProcessor) RegisterRankBoardMessageHandler(msgId int32, h logicCommon.RankBoardMessageHandler) {
	if h == nil {
		errorMessage := fmt.Sprintf("[rankBoardProcessor] RegisterRankBoardMessageHandler handler is nil msgId:%d", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	if _, ok := l.messageHandlerMap[msgId]; ok {
		errorMessage := fmt.Sprintf("[rankBoardProcessor] RegisterRankBoardMessageHandler msgId:%d already registered", msgId)
		logger.ErrorWithZapFields(errorMessage)
		panic(errorMessage)
	}
	l.messageHandlerMap[msgId] = h
}

func (l *RankBoardMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	if session == nil {
		logger.ErrorBySprintf("[rankBoardProcessor] PushMessage nil session")
		return
	}
	rankBoardSession := session.(*logicSessionManager.RankBoardSession)
	if rankBoardSession == nil {
		logger.ErrorBySprintf("[rankBoardProcessor] PushMessage nil rankBoardSession")
		return
	}
	index := tool.RandInt32(0, int32(l.processorNum)-1)
	if index < 0 || int(index) >= l.processorNum {
		logger.ErrorBySprintf(fmt.Sprintf("[rankBoardProcessor] RankBoard index out of range sessionId:%d,msgId:%d", session.GetID(), msgID))
		return
	}
	handler := l.messageHandlerMap[msgID]
	if handler == nil {
		logger.ErrorBySprintf(fmt.Sprintf("[rankBoardProcessor] no handler for msgId:%d", msgID))
		return
	}
	l.processors[index].PutTask(session, msgID, rankBoardSession.RankBoardId, msg, handler)
}
