package logic

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform"
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

func (l *LoginMessageProcessor) Put(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	index := session.GetID() % l.processorNum
	l.processors[index].Put(session, msgID, msg, l.processMap[msgID])
}

func (l *LoginMessageProcessor) Process(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

type LoginTask struct {
	user    *model.LoginUser
	msg     proto.Message
	handler MessageHandleFunc
}

func (l *LoginTask) GetUser() logicInterface.UserBaseInterface {
	return l.user
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

func (p *LoginProcessor) Put(session serviceInterface.SessionInterface, msgID int32, msg proto.Message, handler MessageHandleFunc) {
	task := &LoginTask{
		user:    model.NewLoginUser(session),
		msg:     msg,
		handler: handler,
	}
	p.tasks <- task
}

func (p *LoginProcessor) start() {
	for task := range p.tasks {
		if task.handler == nil {
			platform.ErrorWithFunction(enum.FUNC_LOGIN, "handler is nil", task.user, nil)
			continue
		}
		if task.msg == nil {
			platform.ErrorWithFunction(enum.FUNC_LOGIN, "msg is nil", task.user, nil)
			continue
		}
		if task.user == nil {
			platform.ErrorWithFunction(enum.FUNC_LOGIN, "user is nil", task.user, nil)
			continue
		}
		task.handler(task.msg, task.user)
	}
}
