package dispatcherService

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/logic/socialService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type AllianceTask struct {
	session  serviceInterface.SessionInterface
	msg      proto.Message
	alliance *socialService.AllianceModel
	handler  logicCommon.AllianceMessageHandler
}

type AllianceProcessor struct {
	id                  int32
	processorNum        int
	tickInterval        time.Duration
	messageCountPerTick int32
	stopCh              chan struct{}
	tasks               chan *AllianceTask
	monitor             *logicCommon.ThroughputMonitor
}

func NewAllianceProcessor(id int32, processorNum int, tickInterval time.Duration, messageCountPerTick, messageBufferSize int32) *AllianceProcessor {
	return &AllianceProcessor{
		id:                  id,
		processorNum:        processorNum,
		tickInterval:        tickInterval,
		messageCountPerTick: messageCountPerTick,
		stopCh:              make(chan struct{}),
		tasks:               make(chan *AllianceTask, messageBufferSize),
		monitor:             logicCommon.NewThroughputMonitor(enum.GetAllianceProcessKey(id), 5*time.Second, 1*time.Minute),
	}
}

func (p *AllianceProcessor) PutTask(session serviceInterface.SessionInterface, msgID int32, msg proto.Message, handler logicCommon.AllianceMessageHandler, alliance *socialService.AllianceModel) {
	task := &AllianceTask{
		session:  session,
		msg:      msg,
		handler:  handler,
		alliance: alliance,
	}
	select {
	case p.tasks <- task:
		p.monitor.AddReceived(1)
	default:
		logger.ErrorBySprintf("[allianceProcessor] local queue full for session:%d msg:%d", session.GetID(), msgID)
	}
}

func (p *AllianceProcessor) start() {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	go p.monitor.Start()
	var lastTick time.Time

	for now := range ticker.C {
		if !lastTick.IsZero() {
			delay := now.Sub(lastTick)
			if delay > p.tickInterval*2 {
				logger.ErrorBySprintf("[allianceProcessor] tick delayed processorId:%d delay=%v", p.id, delay)
			}
		}
		lastTick = now

		start := time.Now()
		handled := int32(0)

		for handled < p.messageCountPerTick {
			select {
			case task := <-p.tasks:
				if task.handler == nil {
					logger.ErrorBySprintf("[allianceProcessor] PushMessage handler is nil processorId:%d", p.id)
					continue
				}
				if task.msg == nil || task.session == nil {
					logger.ErrorBySprintf("[allianceProcessor] PushMessage session or msg is nil processorId:%d", p.id)
					continue
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.ErrorBySprintf("[allianceProcessor] handle message processorId:%d panic:%+v", p.id, r)
						}
					}()
					socialSession, ok := task.session.(*logicSessionManager.AllianceSession)
					if !ok || socialSession == nil {
						logger.ErrorBySprintf("[allianceProcessor] invalid social session processorId:%d", p.id)
						return
					}
					task.handler(task.msg, socialSession, task.alliance)
				}()
				handled++
			case <-p.stopCh:
				p.monitor.Stop()
				logger.InfoWithSprintf("[allianceProcessor] stop processorId:%d", p.id)
				return
			default:
				goto END
			}
		}
	END:
		socialService.GetService().HeartbeatByProcessor(p.id, p.processorNum)
		socialService.GetService().FlushDirtyByProcessor(p.id, p.processorNum)
		cost := time.Since(start)
		p.monitor.AddHandled(handled)
		if cost > p.tickInterval {
			logger.ErrorBySprintf("[allianceProcessor] tick cost exceed processorId:%d cost=%v handled=%d", p.id, cost, handled)
		}
	}
}

type AllianceMessageProcessor struct {
	messageHandlerMap map[int32]logicCommon.AllianceMessageHandler
	processors        []*AllianceProcessor
	processorNum      int
}

var _ serviceInterface.MessageProcessorInterface = (*AllianceMessageProcessor)(nil)

func NewAllianceMessageProcessor(config *nodeConfig.MessageProcessConfig) *AllianceMessageProcessor {
	logger.InfoWithSprintf("[allianceProcessor] init alliance message processor config:%+v", config)
	processor := &AllianceMessageProcessor{
		messageHandlerMap: make(map[int32]logicCommon.AllianceMessageHandler),
		processors:        make([]*AllianceProcessor, config.RoutineNum),
		processorNum:      int(config.RoutineNum),
	}
	for i := 0; i < processor.processorNum; i++ {
		processor.processors[i] = NewAllianceProcessor(int32(i), processor.processorNum, config.TickInterval, config.MessageCountPerTick, config.MessageBufferSize)
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

func (s *AllianceMessageProcessor) RegisterAllianceMessageHandler(msgID int32, h logicCommon.AllianceMessageHandler) {
	if h == nil {
		errMessage := fmt.Sprintf("[allianceProcessor] RegisterAllianceMessageHandler handler is nil msgId:%d", msgID)
		logger.ErrorWithZapFields(errMessage)
		panic(errMessage)
	}
	if _, ok := s.messageHandlerMap[msgID]; ok {
		errMessage := fmt.Sprintf("[allianceProcessor] RegisterAllianceMessageHandler msgId:%d already registered", msgID)
		logger.ErrorWithZapFields(errMessage)
		panic(errMessage)
	}
	s.messageHandlerMap[msgID] = h
}

func (s *AllianceMessageProcessor) PushMessage(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	if session == nil {
		logger.ErrorBySprintf("[allianceProcessor] PushMessage nil session")
		return
	}
	socialSession, ok := session.(*logicSessionManager.AllianceSession)
	if !ok || socialSession == nil {
		logger.ErrorBySprintf("[allianceProcessor] PushMessage nil socialSession")
		return
	}
	handler := s.messageHandlerMap[msgID]
	if handler == nil {
		logger.ErrorBySprintf("[allianceProcessor] no handler for msgId:%d", msgID)
		return
	}

	allianceId := socialSession.AllianceId
	msgAllianceId := extractAllianceID(msg)
	if msgAllianceId > 0 {
		if allianceId > 0 && allianceId != msgAllianceId {
			logger.ErrorBySprintf("[allianceProcessor] session alliance mismatch userId:%d msgId:%d sessionAllianceId:%d msgAllianceId:%d", socialSession.UserId, msgID, allianceId, msgAllianceId)
		}
		allianceId = msgAllianceId
	}

	if allianceId <= 0 {
		s.processors[0].PutTask(session, msgID, msg, handler, nil)
		return
	}

	alliance := socialService.GetService().GetAllianceById(allianceId)
	index := int(allianceId % int64(s.processorNum))
	if index < 0 || index >= s.processorNum {
		logger.ErrorBySprintf("[allianceProcessor] index out of range userId:%d,msgId:%d allianceId:%d", socialSession.UserId, msgID, socialSession.AllianceId)
		return
	}
	s.processors[index].PutTask(session, msgID, msg, handler, alliance)
}

func extractAllianceID(msg proto.Message) int64 {
	switch req := msg.(type) {
	case *rpcPb.ChangeAllianceBasicInfoReq:
		return req.GetAllianceId()
	case *rpcPb.GetAllianceInfoReq:
		return req.GetAllianceId()
	case *rpcPb.ApplyAllianceReq:
		return req.GetAllianceId()
	case *rpcPb.KickAllianceMemberReq:
		return req.GetAllianceId()
	case *rpcPb.QuitAllianceReq:
		return req.GetAllianceId()
	case *rpcPb.ChangeMemberPositionReq:
		return req.GetAllianceId()
	default:
		return 0
	}
}
