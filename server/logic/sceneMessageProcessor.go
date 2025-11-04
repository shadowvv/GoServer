package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type SceneMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*SceneMessageProcessor)(nil)

func (p *SceneMessageProcessor) Put(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (p *SceneMessageProcessor) Process(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
