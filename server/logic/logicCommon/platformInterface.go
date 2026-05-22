package logicCommon

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"google.golang.org/protobuf/proto"
)

type MessageSenderInterface interface {
	SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message)
	SendMessage(player UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message)
	Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int32)
	SendErrorMessage(player UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE)
	CloseSessionWithError(player UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE)
}

type RpcMessageSenderInterface interface {
	SendMessageToRankBoard(userId int64, rankId string, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error
}

type UnlockServiceInterface interface {
	CheckUnlock(unlockId int32, player PlayerInterface) bool
	CheckSystemUnlock(systemId int32, player PlayerInterface) bool
	CheckServerInfoUnlock(unlockId int32, server ServerInfoInterface) bool
}

type SessionCloseHooker interface {
	OnSessionClose(player PlayerInterface)
}

// RPC服务钩子
type RPCServiceHooker interface {
	// 节点断开
	OnNodeDisconnect(id int32, nodeType enum.NodeType, nodeAddress string)
	OnNodeConnect(id int32, nodeType enum.NodeType)
}

type GrpcSenderInterface[Req any, Resp any] interface {
	// Send 发送一条消息到 gRPC 流
	Send(msg *Resp) error

	// Close 关闭发送器，释放资源
	Close()
}

// TODO:尝试修改未GetSessionBySessionId和GetSessionById
type SessionManagerInterface interface {
	// 获取玩家
	GetPlayerBasicInfoBySessionId(sessionId int64) UserBaseInterface
	// 获取玩家
	GetPlayerBasicInfoByUserId(userId int64) UserBaseInterface
}

// 服务器信息接口
type ServerInfoInterface interface {
	// 获取服务器ID
	GetServerId() int32
	// 获取服务器启动时间
	GetServerOpenTime() int64
	// 获取服务器时间
	GetServerTime() int64
	// 获取服务器注册数
	GetRegisterCount() int32
	// 获取服务器活跃玩家数
	GetActivePlayerCount() int32
}

type SignalHooker interface {
	// 配置重载后
	AfterAllConfigReload()
	// 踢掉所有玩家
	KickAllPlayer()
}
