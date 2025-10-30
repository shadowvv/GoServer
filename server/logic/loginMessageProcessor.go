package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type LoginMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*LoginMessageProcessor)(nil)

func (l *LoginMessageProcessor) RegisterProcess(id uint32, msg proto.Message, h MessageHandleFunc) {

}

func (l *LoginMessageProcessor) Put(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (l *LoginMessageProcessor) Process(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
