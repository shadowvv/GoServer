package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type InnerMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*InnerMessageProcessor)(nil)

func (p *InnerMessageProcessor) Put(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (p *InnerMessageProcessor) Process(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
