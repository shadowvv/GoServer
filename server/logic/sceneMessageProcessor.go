package logic

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type SceneMessageProcessor struct {
}

var _ serviceInterface.MessageProcessorInterface = (*SceneMessageProcessor)(nil)

func (p *SceneMessageProcessor) Put(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}

func (p *SceneMessageProcessor) Process(connectionId int64, msgID uint32, msg proto.Message) {
	//TODO implement me
	panic("implement me")
}
