package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type ServiceMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*SceneMessageProcessor)(nil)

func (s *ServiceMessageProcessor) Put(connectionId int64, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (s *ServiceMessageProcessor) Process(connectionId int64, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
