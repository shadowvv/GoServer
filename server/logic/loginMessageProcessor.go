package logic

import (
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type LoginMessageProcessor struct {
	processMap map[int32]MessageHandleFunc
}

var _ serviceInterface.MessageProcessorInterface = (*LoginMessageProcessor)(nil)

func (l *LoginMessageProcessor) RegisterProcess(id int32, msg proto.Message, h MessageHandleFunc) {

}

func (l *LoginMessageProcessor) Put(connectionId int64, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (l *LoginMessageProcessor) Process(connectionId int64, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

type LoginTask struct {
	connectionID int64
	msgID        uint32
	msg          proto.Message
	handler      MessageHandleFunc
}

func (l *LoginTask) GetConnectionID() int64 {
	return l.connectionID
}

func (l *LoginTask) GetMsgID() uint32 {
	return l.msgID
}

func (l *LoginTask) GetMsg() proto.Message {
	return l.msg
}

type Processor struct {
	tasks chan *LoginTask
}

func NewProcessor() *Processor {
	return &Processor{
		tasks: make(chan *LoginTask, 1000),
	}
}

func (p *Processor) Put(connectionId int64, msgID uint32, msg proto.Message, handler MessageHandleFunc) {
	task := &LoginTask{
		connectionID: connectionId,
		msgID:        msgID,
		msg:          msg,
		handler:      handler,
	}
	p.tasks <- task
}

func (p *Processor) start() {
	for task := range p.tasks {
		user := platform.GetUserByConnectionID(task.connectionID)
		if user == nil {
			platform.Error("[login] user not found", user)
			continue
		}
		task.handler(task.msg, user)
	}
}
