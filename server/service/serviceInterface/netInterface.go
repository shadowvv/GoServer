package serviceInterface

import "google.golang.org/protobuf/proto"

// HandlerFunc 是消息处理函数签名（用户实现）
type HandlerFunc func(c ConnectionInterface, payload []byte)

type CodecInterface interface {
	Marshal(pb proto.Message) ([]byte, error)
	Unmarshal(data []byte, msg proto.Message) error
}

type AcceptorInterface interface {
	Accept(data []byte)
}

type RouterInterface interface {
	Register(msgID uint32, h HandlerFunc)
	Dispatch(msgID uint32, c ConnectionInterface, payload []byte)
}

type ConnectionInterface interface {
	Send(data []byte) error
	Close()
	OnDisconnect()
}
