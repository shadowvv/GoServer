package serviceInterface

import (
	"github.com/drop/GoServer/server/enum"
	"google.golang.org/protobuf/proto"
)

// CodecInterface 消息编解码接口
type CodecInterface interface {
	// 编码
	Marshal(msgID int32, msg proto.Message) ([]byte, error)
	// 解码
	Unmarshal(data []byte, msg proto.Message) error
}

// AcceptorInterface 接收器器接口
type AcceptorInterface interface {
	// 接收连接
	Accept(connection SessionInterface)
	// 获取连接
	GetSessionById(id int64) SessionInterface
	// 连接超时处理
	OnConnectionTimeout(connectionInterface SessionInterface)
}

// RouterInterface 路由器接口
type RouterInterface interface {
	// 注册消息
	RegisterMessage(msgType int32, msgID int32, msg proto.Message)
	// 分发消息
	Dispatch(session SessionInterface, msgID int32, msg proto.Message)
	// 获取消息
	GetMessage(msgID int32) proto.Message
	// 判断消息是否没有任何字段
	IsEmpty(msgID int32) bool
}

// 内部消息处理接口
type InnerTaskInterface interface {
	// 请求处理
	ReqCall(task InnerTaskInterface) (any, error)
	// 处理完成
	Resolve(res any, err error)
	// 获取请求ID
	GetReqId() int64
	// 获取响应ID
	GetRespId() int64
	// 设置错误
	SetError(err error)
}

type DispatchInterface interface {
	// 分发消息
	DispatchGameMessage(session SessionInterface, msgID, msgType int32, msg proto.Message)
	// 分发内部任务
	DispatchInnerMessageTask(reqType enum.InnerMessageType, msgId enum.InnerMessageId, reqId int64, parameter any, respType enum.InnerMessageType, respId int64, respCallback InnerTaskResult)
	// 分发内部消息
	DispatchInnerTask(task InnerTaskInterface)
	// 分发内部消息响应
	DispatchInnerTaskResp(task InnerTaskInterface, respHandler func())
}

// 内部消息结果处理接口
type InnerTaskResult func(result any, err error)

// MessageProcessorInterface 消息处理接口
type MessageProcessorInterface interface {
	// 放入消息
	PushMessage(session SessionInterface, msgID int32, msg proto.Message)
}

// SessionInterface 连接接口
type SessionInterface interface {
	// 发送消息
	Send(msgId int32, msg proto.Message)
	// 关闭连接并发送消息
	SendAndClose(msgId int32, msg proto.Message)
	// 主动关闭连接
	Close()
	// 获取连接ID
	GetID() int64
	// 获取连接状态
	IsActive() bool
}
