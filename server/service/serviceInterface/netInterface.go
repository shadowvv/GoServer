package serviceInterface

import "google.golang.org/protobuf/proto"

// HandlerFunc 消息处理函数
type HandlerFunc func(msgID uint32, msg proto.Message)

// CodecInterface 消息编解码接口
type CodecInterface interface {
	// Marshal 编码
	Marshal(msg proto.Message) ([]byte, error)
	// Unmarshal 解码
	Unmarshal(data []byte, msg proto.Message) error
}

// AcceptorInterface 接收器器接口
type AcceptorInterface interface {
	// 接收连接
	Accept(connection ConnectionInterface)
	// 连接超时处理
	OnConnectionTimeout(connectionInterface ConnectionInterface)
}

// RouterInterface 路由器接口
type RouterInterface interface {
	// 注册消息处理函数
	Register(msgID uint32, msg proto.Message, h HandlerFunc)
	// 分发消息
	Dispatch(msgID uint32, msg proto.Message)
	// 获取消息
	GetMessage(msgID uint32) proto.Message
}

// ConnectionInterface 连接接口
type ConnectionInterface interface {
	// 发送消息
	Send(msg proto.Message) error
	// 主动关闭连接
	Close()
	// 获取连接ID
	GetID() int64
}
