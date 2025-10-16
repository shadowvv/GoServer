package serviceInterface

import "google.golang.org/protobuf/proto"

type NetInterface interface {
	GetName() string
	GetIp() string
	GetMac() string
}

type CodecInterface interface {
	Marshal(pb proto.Message) ([]byte, error)
	Unmarshal(data []byte, msg proto.Message) error
}

type AcceptorInterface interface {
	Accept(data []byte)
}

type RouterInterface interface {
	Dispatch(msgID uint32, c SocketInterface, payload []byte)
}

type SocketInterface interface {
	Send(data []byte)
	Close()
	OnDisconnect()
}
