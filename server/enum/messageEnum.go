package enum

// 消息类型枚举
type MessageType int32

// 消息类型
const (
	MSG_TYPE_LOGIN     MessageType = iota + 1 // 登录
	MSG_TYPE_PLAYER                           // 玩家
	MSG_TYPE_Alliance                         // 社交
	MSG_TYPE_RANKBOARD                        // 排行榜
)

// 内部消息类型枚举
type InnerMessageType uint32

const (
	INNER_MSG_TYPE_PLAYER InnerMessageType = 101 // 玩家
	INNER_MSG_TYPE_SCENE  InnerMessageType = 102 // 场景
)

// 内部消息
type InnerMessageId int32

const (
	INNER_MEG_AFTER_LOGIN            InnerMessageId = 101 // 登录后
	INNER_MSG_PLAYER_LOGOUT          InnerMessageId = 102 // 玩家下线
	INNER_MSG_DELIVER_RECHARGE_ITEM  InnerMessageId = 103 // 派发充值道具
	INNER_MSG_EVENT_TASK_PLAYER      InnerMessageId = 300 // 发送事件到场景player
	INNER_MSG_EVENT_UPDATE_RANKBOARD InnerMessageId = 301 // 更新排行榜
)
