package serviceInterface

import "google.golang.org/protobuf/proto"

// CodecInterface 消息编解码接口
type CodecInterface interface {
	// 编码
	Marshal(msg proto.Message) ([]byte, error)
	// 解码
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
	// 注册消息
	RegisterProcess(msgType uint32, msgID int32, msg proto.Message)
	// 注册消息处理器
	RegisterProcessor(msgType uint32, processor MessageProcessorInterface)
	// 分发消息
	Dispatch(connectionId int64, msgID int32, msg proto.Message)
	// 获取消息
	GetMessage(msgID int32) proto.Message
}

// MessageProcessorInterface 消息处理接口
type MessageProcessorInterface interface {
	// 放入消息
	Put(connectionId int64, msgID int32, msg proto.Message)
	// 处理消息
	Process(connectionId int64, msgID int32, msg proto.Message)
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
