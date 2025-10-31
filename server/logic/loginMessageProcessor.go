package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type LoginMessageProcessor struct {
	processMap   map[int32]MessageHandleFunc
	processors   []*LoginProcessor
	processorNum int64
}

func NewLoginMessageProcessor(processorNum int32) *LoginMessageProcessor {
	processor := &LoginMessageProcessor{
		processMap:   make(map[int32]MessageHandleFunc),
		processors:   make([]*LoginProcessor, processorNum),
		processorNum: int64(processorNum),
	}
	for i := 0; i < int(processorNum); i++ {
		processor.processors[i] = NewProcessor()
	}
	for _, p := range processor.processors {
		go p.start()
	}
	return processor
}

var _ serviceInterface.MessageProcessorInterface = (*LoginMessageProcessor)(nil)

func (l *LoginMessageProcessor) RegisterProcess(msgId int32, h MessageHandleFunc) {
	l.processMap[msgId] = h
}

func (l *LoginMessageProcessor) Put(connectionId int64, msgID int32, msg proto.Message) {
	index := connectionId % l.processorNum
	l.processors[index].Put(connectionId, msgID, msg, l.processMap[msgID])
}

func (l *LoginMessageProcessor) Process(connectionId int64, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

type LoginTask struct {
	connectionID int64
	msgID        int32
	msg          proto.Message
	handler      MessageHandleFunc
}

func (l *LoginTask) GetConnectionID() int64 {
	return l.connectionID
}

func (l *LoginTask) GetMsgID() int32 {
	return l.msgID
}

func (l *LoginTask) GetMsg() proto.Message {
	return l.msg
}

type LoginProcessor struct {
	tasks chan *LoginTask
}

func NewProcessor() *LoginProcessor {
	return &LoginProcessor{
		tasks: make(chan *LoginTask, 1000),
	}
}

func (p *LoginProcessor) Put(connectionId int64, msgID int32, msg proto.Message, handler MessageHandleFunc) {
	task := &LoginTask{
		connectionID: connectionId,
		msgID:        msgID,
		msg:          msg,
		handler:      handler,
	}
	p.tasks <- task
}

func (p *LoginProcessor) start() {
	for task := range p.tasks {
		task.handler(task.msg, nil)
	}
}
