package logic

import "google.golang.org/protobuf/proto"

type ProcessorInterface interface {
	RegisterProcess(msgID int32, msg proto.Message, h MessageHandleFunc)
	Put(connectionId int64, msgID uint32, msg proto.Message)
	Process(connectionId int64, msgID uint32, msg proto.Message)
}

type TaskInterface interface {
	GetTargetID() int64
	GetMsgID() uint32
	GetMsg() proto.Message
}
