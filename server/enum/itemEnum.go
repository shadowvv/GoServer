package enum

const (
	DIAMOND_ITEM_ID = 1       // 钻石
	STAMINA_ITEM_ID = 1110052 // 体力
)

// 物品类型枚举
type ItemType uint32

const (
	ITEM_TYPE_CURRENCY              ItemType = 10 // 货币
	ITEM_TYPE_RESOURCE              ItemType = 11 // 资源
	ITEM_TYPE_NORMAL_CHEST          ItemType = 12 // 普通宝箱
	ITEM_TYPE_HERO                  ItemType = 13 // 英雄
	ITEM_TYPE_SCRAP                 ItemType = 14 // 碎片
	ITEM_TYPE_PICK_CHEST            ItemType = 15 // 选择宝箱
	ITEM_TYPE_EQUIP                 ItemType = 16 // 装备
	ITEM_TYPE_LOOP_BOX              ItemType = 17 // 循环盒
	ITEM_TYPE_ACCESSORY             ItemType = 18 // 饰品
	ITEM_TYPE_PLAYER_EXP            ItemType = 19 // 经验
	ITEM_TYPE_VIP_CARD              ItemType = 20 // 特权卡
	ITEM_TYPE_PASS                  ItemType = 21 // 通行证
	ITEM_TYPE_PASS_VIP              ItemType = 22 // 通行证VIP
	ITEM_TYPE_ARCHITECTURE_SPEED_UP ItemType = 23 // 建筑加速道具
	ITEM_TYPE_CREATE_SPEED_UP       ItemType = 24 // 生成加速道具
	ITEM_TYPE_AD_CHEST              ItemType = 25 // 广告宝箱
	ITEM_TYPE_PET                   ItemType = 26 // 宠物
	ITEM_TYPE_COLLECTION            ItemType = 27 // 藏品
	ITEM_TYPE_COLLECTION_FRAGMENT   ItemType = 28 // 藏品碎片
	ITEM_TYPE_EXPEDITION_SPEED_UP   ItemType = 29 // 派遣加速道具
	ITEM_TYPE_EQUIPMENT_PAPER       ItemType = 30 // 装备图纸
	ITEM_TYPE_EQUIPMENT_ENHANCEMENT ItemType = 31 // 装备强化道具
	ITEM_TYPE_TRIAL                 ItemType = 32 // 试炼
	ITEM_TYPE_APPEARANCE            ItemType = 33 // 外观
	ITEM_TYPE_ACTIVITY_ITEM         ItemType = 34 // 活动道具
	ITEM_TYPE_ACTIVITY_POINT        ItemType = 35 // 活动积分（兑换商店）
	ITEM_TYPE_ACTIVITY_PASS_POINT   ItemType = 36 // 活动通行证积分
)

func IsValidItemType(v int32) bool {
	return v >= int32(ITEM_TYPE_CURRENCY) && v <= int32(ITEM_TYPE_ACTIVITY_PASS_POINT)
}

// 物品品质枚举
type ItemQuality uint32

const (
	ITEM_QUALITY_BAD    ItemQuality = 1 // 次品
	ITEM_QUALITY_NORMAL ItemQuality = 2 // 普通
	ITEM_QUALITY_RARE   ItemQuality = 3 // 稀有
	ITEM_QUALITY_EPIC   ItemQuality = 4 // 史诗
	ITEM_QUALITY_LEGEND ItemQuality = 5 // 传说
	ITEM_QUALITY_MYTHIC ItemQuality = 6 // 神话
	ITEM_QUALITY_UNIQUE ItemQuality = 7 // 远古
	ITEM_QUALITY_EIGHT  ItemQuality = 8
	ITEM_QUALITY_NINE   ItemQuality = 9
)

func IsValidItemQuality(v int32) bool {
	return v >= int32(ITEM_QUALITY_BAD) && v <= int32(ITEM_QUALITY_NINE)
}

type InventoryType uint32

const (
	INVENTORY_TYPE_MAIN  InventoryType = 1 // 主背包
	INVENTORY_TYPE_EQUIP InventoryType = 2 // 装备栏
	INVENTORY_TYPE_VIP   InventoryType = 5 // 特权背包
)

// 背包操作结果枚举
type InventoryResult uint32

const (
	INVENTORY_RESULT_SUCCESS              InventoryResult = 0  // 成功
	INVENTORY_RESULT_FULL                 InventoryResult = 1  // 背包已满
	INVENTORY_RESULT_INVALID_ITEM         InventoryResult = 2  // 无效物品
	INVENTORY_RESULT_NOT_ENOUGH_SPACE     InventoryResult = 3  // 空间不足
	INVENTORY_RESULT_ITEM_NOT_FOUND       InventoryResult = 4  // 物品不存在
	INVENTORY_RESULT_INVALID_POSITION     InventoryResult = 5  // 无效位置（已废弃）
	INVENTORY_RESULT_CANNOT_STACK         InventoryResult = 6  // 无法堆叠（已废弃）
	INVENTORY_RESULT_NOT_ENOUGH_ITEMS     InventoryResult = 7  // 物品数量不足
	INVENTORY_RESULT_INSUFFICIENT_ITEMNUM InventoryResult = 8  // 数量不足
	INVENTORY_RESULT_CANNOT_USE           InventoryResult = 9  // 无法使用
	INVENTORY_RESULT_INVALID_INVENTORY    InventoryResult = 10 // 无效背包类型
)

type ItemChangeReason uint32

const (
	ITEM_CHANGE_REASON_LOTTERY                      ItemChangeReason = iota // 抽卡
	ITEM_CHANGE_REASON_MONSTER_DROP                                         // 怪物掉落
	ITEM_CHANGE_REASON_HERO_ALBUM_REWARD                                    // 英雄专辑奖励
	ITEM_CHANGE_REASON_HERO_REBIRTH                                         // 英雄重生
	ITEM_CHANGE_REASON_GM                                                   // GM
	ITEM_CHANGE_REASON_HERO_BREAK                                           // 英雄突破
	ITEM_CHANGE_REASON_HERO_LEVEL_UP                                        // 英雄升级
	ITEM_CHANGE_REASON_HERO_STAR_UP                                         // 英雄技能升级
	ITEM_CHANGE_REASON_DISCARD_ITEM                                         // 丢弃物品
	ITEM_CHANGE_REASON_DRAW_LOTTERY                                         // 抽奖
	ITEM_CHANGE_REASON_CHANGE_NICKNAME                                      // 修改昵称
	ITEM_CHANGE_REASON_DECOMPOSE_EQUIP                                      // 装备分解
	ITEM_CHANGE_REASON_DECOMPOSE_PET                                        // 宠物分解
	ITEM_CHANGE_REASON_EXCHANGE_ITEM                                        // 兑换
	ITEM_CHANGE_REASON_CASH_SHOP_BUY                                        // 商城购买
	ITEM_CHANGE_REASON_WEEK_PASS                                            // 周卡
	ITEM_CHANGE_REASON_ACCESSORY_LUCKY                                      // 幸运配件
	ITEM_CHANGE_REASON_TASK                                                 // 任务
	ITEM_CHANGE_REASON_DAILY_TASK                                           // 日常任务
	ITEM_CHANGE_REASON_CREATE_PLAYER                                        // 创建角色
	ITEM_CHANGE_REASON_USE_ITEM                                             // 使用物品
	ITEM_CHANGE_REASON_MAIL_ATTACHMENT                                      // 邮件附件
	ITEM_CHANGE_REASON_RANK_LIKE                                            // 评分奖励
	ITEM_CHANGE_REASON_RANK_CLAIM_BOX                                       // 排名
	ITEM_CHANGE_REASON_IDLE_REWARD                                          // 挂机奖励
	ITEM_CHANGE_REASON_IDLE_QUICK_CLAIM                                     // 挂机快速领取
	ITEM_CHANGE_REASON_SWEEP_INSTANCE                                       // 扫荡
	ITEM_CHANGE_REASON_COMMIT_INSTANCE_LEVEL_REWARD                         // 领取副本奖励
	ITEM_CHANGE_REASON_CHALLENGE_INSTANCE                                   // 挑战
	ITEM_CHANGE_REASON_INSTANCE_RESET_TICKET                                // 副本重置券
	ITEM_CHANGE_REASON_VIP_PRIVILEGE_REWARD                                 // 特权奖励
	ITEM_CHANGE_REASON_PASS_CARD_TASK                                       // 循环卡任务
	ITEM_CHANGE_REASON_SYSTEM_UNLOCK_REWARD                                 // 系统解锁奖励
	ITEM_CHANGE_REASON_TOWER_STAGE_REWARD                                   // 塔
	ITEM_CHANGE_REASON_ARCHITECTURE                                         // 建筑
	ITEM_CHANGE_REASON_ARENA_REFRESH                                        // 刷新竞技场列表
	ITEM_CHANGE_REASON_ARENA_CHALLENGE                                      // 挑战
	ITEM_CHANGE_REASON_BINDING_BONUS                                        // 绑定账号奖励
	ITEM_CHANGE_REASON_ARENA_WIN                                            // 竞技场胜利
	ITEM_CHANGE_REASON_ARENA_LOSE                                           // 竞技场失败
	ITEM_CHANGE_REASON_LOOP_BOX_POINT                                       // 秘罐积分兑换
	ITEM_CHANGE_REASON_LOOP_BOX_OPEN                                        // 秘罐开启消耗
	ITEM_CHANGE_REASON_DECOMPOSE_EQUIP_CONSUME                              // 装备分解消耗
	ITEM_CHANGE_REASON_HERO_STAR_UP_MATERIAL                                // 英雄升星材料消耗
	ITEM_CHANGE_REASON_SIGN                                                 // 七日签到奖励
	ITEM_CHANGE_REASON_PASS_CARD                                            // 通行证奖励
	ITEM_CHANGE_REASON_BOUNTY_TASK                                          // 悬赏令任务奖励
	ITEM_CHANGE_REASON_COLLECTION_LEVEL_UP                                  // 藏品升级
	ITEM_CHANGE_REASON_ENTRY_LEVEL_UP                                       // 藏品词条升级
	ITEM_CHANGE_RESON_START_EXPEDITION                                      // 开始派遣
	ITEM_CHANGE_REASON_FINISH_EXPEDITION                                    // 派遣完成
	ITEM_CHANGE_REASON_FREE_STAMINA                                         // 免费领取体力
	ITEM_CHANGE_REASON_RECOVERY_STAMINA                                     // 恢复体力
	ITEM_CHANGE_REASON_CLIAM_EXPEDITION_REWARD                              // 领取派遣奖励
	ITEM_CHANGE_REASON_FORGE_EQUIPMENT                                      // 装备锻造
	ITEM_CHANGE_REASON_CITY_LUMBER_COLLECT                                  // 领取伐木场产物
	ITEM_CHANGE_REASON_CITY_FURNITURE_LEVEL_UP                              // 家具升级消耗
	ITEM_CHANGE_REASON_STRONG_EQUIPMENT                                     // 装备强化
	ITEM_CHANGE_REASON_REBRITH_EQUIPMENT                                    // 装备重生
	ITEM_CHANGE_REASON_TRIAL_TASK_REWARD                                    // 试炼任务奖励
	ITEM_CHANGE_REASON_TRIAL_PROGRESS_REWARD                                // 试炼进度奖励
	ITEM_CHANGE_REASON_CREATE_ALLIANCE                                      // 创建联盟
	ITEM_CHANGE_REASON_CREATE_ALLIANCE_FILED                                // 创建联盟失败
	ITEM_CHANGE_REASON_CREATE_ALLIANCE_NAME                                 // 修改联盟名字
	ITEM_CHANGE_REASON_CREATE_ALLIANCE_NAME_FILED                           // 修改联盟名字失败
	ITEM_CHANGE_REASON_BUY_DISPATCH_FORMATION                               // 购买派遣formation
	ITEM_CHANGE_REASON_CITY_AGE                                             // 主城时代
	ITEM_CHANGE_REASON_APPEARANCE_UNLOCK                                    // 解锁外观
	ITEM_CHANGE_REASON_ADVENTURE_REWARD                                     // adventure reward
	ITEM_CHANGE_REASON_PET_RECRUIT                                          // 宠物招募发放（不弹窗）
	ITEM_CHANGE_REASON_GLORY_ARENA_OPEN_AWARD                               // 荣耀擂台宝箱
	ITEM_CHANGE_REASON_GLORY_ARENA_WIN                                      // 荣耀擂台胜利
	ITEM_CHANGE_REASON_GLORY_ARENA_LOSE                                     // 荣耀擂台胜利
	ITEM_CHANGE_REASON_TURN_TABLE_REWARD                                    // 转盘奖励
	ITEM_CHANGE_REASON_GLORY_ARENA_REFRESH_LIST                             // 荣耀擂台刷新挑战列表
)

type ItemRefreshType int32

const (
	ITEM_REFRESH_TYPE_NONE             ItemRefreshType = 0 // 无
	ITEM_REFRESH_TYPE_DAY              ItemRefreshType = 1 // 天
	ITEM_REFRESH_TYPE_WEEK             ItemRefreshType = 2 // 周
	ITEM_REFRESH_TYPE_MONTH            ItemRefreshType = 3 // 月
	ITEM_REFRESH_TYPE_NOT_IN_PRIVILEGE ItemRefreshType = 4 // 特权
)

func IsValidItemRefreshType(v int32) bool {
	return v >= int32(ITEM_REFRESH_TYPE_NONE) && v <= int32(ITEM_REFRESH_TYPE_NOT_IN_PRIVILEGE)
}

// 特权卡相关常量
const (
	// VIP_CARD_PERMANENT_HOURS 永久特权卡小时数（约11.4年，业务上可视为"永久"阈值）
	// 规则：当发放的 hours >= VIP_CARD_PERMANENT_HOURS 时，ExpireTime 置 -1 表示永久
	VIP_CARD_PERMANENT_HOURS int64 = 99999

	// AD_CHEST_DAILY_OPEN_LIMIT 广告宝箱每日开启上限（基础值，特权卡可增加）
	AD_CHEST_DAILY_OPEN_LIMIT int32 = 20

	// AD_CHEST_OPEN_TOLERANCE_MS 开启时兼容网络波动的容差(毫秒)
	AD_CHEST_OPEN_TOLERANCE_MS int64 = 5000
)

// 特权功能类型枚举
type VipPrivilegeType uint32

const (
	VIP_PRIVILEGE_NO_AD                   VipPrivilegeType = 1  // 免广告 【数据=时间 单位：秒】
	VIP_PRIVILEGE_BATTLE_SPEED            VipPrivilegeType = 2  // 战斗加速 【数据=加速百分比 格式：万分比】
	VIP_PRIVILEGE_IDLE_TIME               VipPrivilegeType = 3  // 挂机时长 【挂机上限存储时间，单位：秒】
	VIP_PRIVILEGE_IDLE_REWARD             VipPrivilegeType = 4  // 挂机奖励 【快速领取，钻石购买次数，单位次】
	VIP_PRIVILEGE_MAIN_REWARD             VipPrivilegeType = 5  // 主线奖励 【主线杀怪掉落 触发特权掉落，配置=空】
	VIP_PRIVILEGE_LOTTERY_GUAR            VipPrivilegeType = 6  // 抽卡保底 【所有抽卡的保底所需抽取的次数 配置 -次数】
	VIP_PRIVILEGE_FATIGUE_LIMIT           VipPrivilegeType = 7  // 疲劳值上限 【疲劳值上限值，单位：整数】
	VIP_PRIVILEGE_BUILD_QUEUE             VipPrivilegeType = 8  // 建筑队列 【同时使用的建筑队列上限，单位：个，整数】
	VIP_PRIVILEGE_INSTANCE_TICKET         VipPrivilegeType = 9  // 副本门票 【每日免费门票数量，单位：个，整数】
	VIP_PRIVILEGE_RECRUITMENT             VipPrivilegeType = 10 // 招募权益 【每日可领取招募奖励】
	VIP_PRIVILEGE_AD_CHEST_OPEN           VipPrivilegeType = 11 // 广告宝箱每日开启次数 【data=额外次数，单位：次】
	VIP_PRIVILEGE_RECRUITMENT_2           VipPrivilegeType = 12 // 招募2（data=天）
	VIP_PRIVILEGE_RECRUITMENT_3           VipPrivilegeType = 13 // 招募3（data=天）
	VIP_PRIVILEGE_EXPEDITION_QUEUE_FIRST  VipPrivilegeType = 14 // 派遣队列+1
	VIP_PRIVILEGE_EXPEDITION_QUEUE_SECOND VipPrivilegeType = 15 // 派遣队列+1
)

func IsValidVipPrivilegeType(v int32) bool {
	return v >= int32(VIP_PRIVILEGE_NO_AD) && v <= int32(VIP_PRIVILEGE_EXPEDITION_QUEUE_SECOND)
}
