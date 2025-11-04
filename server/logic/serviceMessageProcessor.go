package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type ServiceMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*ServiceMessageProcessor)(nil)

func (s *ServiceMessageProcessor) Put(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (s *ServiceMessageProcessor) Process(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
