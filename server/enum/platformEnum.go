package enum

// 环境枚举
type Environment string

const (
	ENV_LOCAL   Environment = "local" // 本地环境
	ENV_DEVELOP Environment = "dev"   // 开发环境
	ENV_TEST    Environment = "test"  // 测试环境
	ENV_AUDIT   Environment = "audit" // 审核环境
	ENV_STAGE   Environment = "stage" // 预发布环境
	ENV_PRODUCT Environment = "prod"  // 生产环境
)

// 服务器类型枚举
type NodeType string

// 服务器类型枚举
const (
	NODE_TYPE_HTTP      NodeType = "http"      // 逻辑服务器
	NODE_TYPE_GATEWAY   NodeType = "gateway"   // 网关服务器
	NODE_TYPE_GAME      NodeType = "game"      // 游戏服务器
	NODE_TYPE_SOCIAL    NodeType = "social"    // 社交服务器
	NODE_TYPE_RANKBOARD NodeType = "rankBoard" // 排行榜服务器
)

// 服务器类型枚举索引
var nodeTypeIndex = map[NodeType]int32{
	NODE_TYPE_HTTP:      1,
	NODE_TYPE_GATEWAY:   2,
	NODE_TYPE_GAME:      3,
	NODE_TYPE_SOCIAL:    4,
	NODE_TYPE_RANKBOARD: 5,
}

// 获取服务器类型枚举索引
func GetNodeTypeIndex(nodeType NodeType) int32 {
	return nodeTypeIndex[nodeType]
}

const (
	MIN_NODE_ID = 1 // 最小服务器ID
	MAX_NODE_ID = 9 // 最大服务器ID
)

// 广播类型枚举
type BroadcastType int32

const (
	BROADCAST_TYPE_ALL       BroadcastType = iota + 1 // 广播给所有玩家
	BROADCAST_TYPE_SERVER_ID                          // 广播给指定服务器
	BROADCAST_TYPE_ALLIANCE                           // 广播给联盟
)

// 节点状态枚举
type GameServerStatus int32

const (
	GAME_SERVER_STATUS_ONLINE   GameServerStatus = iota // 在线
	GAME_SERVER_STATUS_MAINTAIN                         // 维护
)

// 数据库连接池枚举
type DBPoolType string

// 数据库连接池枚举
const (
	DB_POOL_TYPE_PLAYER DBPoolType = "player" // 玩家数据库连接池
)

// 公告信息类型枚举
type AnnounceInfoType int32

const (
	ANNOUNCE_INFO_TYPE_UPDATE_ANNOUNCE    int32 = iota + 1 // 更新公告
	ANNOUNCE_INFO_TYPE_MERGE_ANNOUNCE                      // 合服公告
	ANNOUNCE_INFO_TYPE_INTERCEPT_ANNOUNCE                  // 拦截公告
	ANNOUNCE_INFO_TYPE_TEMP_ANNOUNCE                       // 临时公告
	ANNOUNCE_INFO_TYPE_PREHEAT_ANNOUNCE                    // 预热
	ANNOUNCE_INFO_TYPE_REGISTER_ANNOUNCE                   // 注册欢迎
	ANNOUNCE_INFO_TYPE_RULE_ANNOUNCE                       // 规则说明
)

type RechargeOrderStatus int32

const (
	RECHARGE_ORDER_STATUS_CREATED   RechargeOrderStatus = iota + 1 // 订单创建
	RECHARGE_ORDER_STATUS_PAYED                                    // 支付成功
	RECHARGE_ORDER_STATUS_DELIVERED                                // 发货成功
	RECHARGE_ORDER_STATUS_FAILED                                   // 发货失败
)

type RechargeType int32

const (
	RECHARGE_TYPE_NORMAL RechargeType = iota + 1 // 正常充值
	RECHARGE_TYPE_TOKEN                          // 充值代金券
	RECHARGE_TYPE_FREE                           // 免费
	RECHARGE_TYPE_NONE                           // 非充值
)

const (
	PLAYER_SCENE_STATUS_TRANSFERING int32 = iota + 1 // 玩家正在切换场景
	PLAYER_SCENE_STATUS_RUNNING                      // 玩家正在正常游戏
)

// ID 生成枚举
type IDGeneratorType int64

const (
	ID_GENERATOR_INNER_MESSAGE    IDGeneratorType = iota // 内部消息
	ID_GENERATOR_NET                                     // 网络
	ID_GENERATOR_USER                                    // 玩家
	ID_GENERATOR_ITEM                                    // 物品
	ID_GENERATOR_HERO                                    // 英雄
	ID_GENERATOR_GM                                      // GM
	ID_GENERATOR_EQUIPMENT                               // 装备
	ID_GENERATOR_BATTLE_ID                               // 战斗
	ID_GENERATOR_MAIL                                    // 邮件
	ID_GENERATOR_MSG                                     // 消息
	ID_GENERATOR_RECHARGE_ORDER                          // 充值订单
	ID_GENERATOR_GM_HTTP                                 // GM后台
	ID_GENERATOR_RANK_BOARD_MAIL                         // 排行榜邮件
	ID_GENERATOR_RANK_SOCIAL_MAIL                        // 设计邮件
	ID_GENERATOR_ALLIANCE                                // 联盟
	ID_GENERATOR_MAX              = 63                   // 最大ID
)
