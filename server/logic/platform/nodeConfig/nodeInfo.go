package nodeConfig

import "github.com/drop/GoServer/server/enum"

var NodeId int32         // 服务节点ID
var NodeType string      // 服务类型
var Env enum.Environment // 环境
var ConfigName string    // 配置名
var ChannelId int32      // 渠道ID

func InitNodeInfo(nodeId int32, nodeType string, env enum.Environment, configName string, channelId int32) {
	NodeId = nodeId
	NodeType = nodeType
	Env = env
	ConfigName = configName
	ChannelId = channelId
}
