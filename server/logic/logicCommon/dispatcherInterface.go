package logicCommon

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type SingleSceneProcessor interface {
	PushPlayerMessage(playerId int64, msgId int32, msg proto.Message, handler PlayerMessageHandler, function enum.FunctionIdEnum, isSceneTransferMessage bool)
	PushPlayerInnerTask(task serviceInterface.InnerTaskInterface)
	PushPlayerInnerResp(task serviceInterface.InnerTaskInterface, respHandle func())
	PutSceneInnerTask(task serviceInterface.InnerTaskInterface)
	PutSceneInnerResp(respHandle func())
}

// 战斗消息处理接口
type BattleMessageHandler func(message proto.Message)

// 网关消息处理接口
type GatewayMessageHandler func(message proto.Message, user *GatewayPlayerInfo)

// 登录消息处理接口
type LoginMessageHandler func(message proto.Message, user UserBaseInterface)

// 排行榜消息处理接口
type RankBoardMessageHandler func(message proto.Message, rankId string, session serviceInterface.SessionInterface)

// 社交消息处理接口
type AllianceMessageHandler func(message proto.Message, session serviceInterface.SessionInterface, alliance AllianceInterface)

// 场景消息处理接口
type PlayerMessageHandler func(message proto.Message, player PlayerInterface, function enum.FunctionIdEnum)

// 内部消息处理接口
type InnerMessageHandler func(innerMessageTask serviceInterface.InnerTaskInterface) (any, error)
