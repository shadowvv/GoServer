package serviceInterface

import "google.golang.org/protobuf/proto"

type HandlerFunc func(msgID uint32, msg *proto.Message)

type CodecInterface interface {
	Marshal(msg *proto.Message) ([]byte, error)
	Unmarshal(data []byte, msg *proto.Message) error
}

type AcceptorInterface interface {
	Accept(connection ConnectionInterface)
}

type RouterInterface interface {
	Register(msgID uint32, msg *proto.Message, h HandlerFunc)
	Dispatch(msgID uint32, msg *proto.Message)
	GetMessage(msgID uint32) *proto.Message
}

type ConnectionInterface interface {
	Send(msg *proto.Message) error
	Close()
	OnDisconnect()
	GetID() int64
}
