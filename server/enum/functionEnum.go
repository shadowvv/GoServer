package enum

// 功能ID枚举
type FunctionIdEnum int32

const (
	FUNCTION_ID_NONE                           FunctionIdEnum = 0    // 无
	FUNCTION_ID_HERO                                          = 1    // 英雄
	FUNCTION_ID_PACK                                          = 2    // 背包
	FUNCTION_ID_ADVANTURE                                     = 3    // 冒险
	FUNCTION_ID_DRAW_HERO_CARD                                = 4    // 抽英雄卡
	FUNCTION_ID_BOUNTY                                        = 5    // 触发任务
	FUNCTION_ID_MAIN_QUEST                                    = 6    // 主任务
	FUNCTION_ID_CAPITAL                                       = 7    // 主城
	FUNCTION_ID_SUMMONING                                     = 8    // 召唤
	FUNCTION_ID_IDLE                                          = 9    // 挂机奖励
	FUNCTION_ID_MAIL                                          = 10   // 邮件
	FUNCTION_ID_ANNOUNCE                                      = 11   // 公告
	FUNCTION_ID_CHAT                                          = 12   // 聊天
	FUNCTION_ID_SHOP                                          = 13   // 商城
	FUNCTION_ID_VIP_CARD                                      = 14   // 通行证
	FUNCTION_ID_RANKBOARD                                     = 15   // 排行榜
	FUNCTION_ID_ALLIANCE                                      = 16   // 联盟
	FUNCTION_ID_MYSTICREALM                                   = 17   // 奇遇秘境
	FUNCTION_ID_CITYAGE                                       = 18   // 主城时代
	FUNCTION_ID_TRIAL                                         = 19   // 七日试炼
	FUNCTION_ID_BATTLE_SPEED                                  = 20   // 加速符
	FUNCTION_ID_HERO_LEVEL_UP                                 = 101  // 英雄升级
	FUNCTION_ID_HERO_FORMATION                                = 102  // 英雄编队
	FUNCTION_ID_HERO_PACK                                     = 103  // 英雄背包
	FUNCTION_ID_HERO_DETAIL                                   = 104  // 英雄详情
	FUNCTION_ID_HERO_STAR                                     = 105  // 英雄升星
	FUNCTION_ID_HERO_EQUIP                                    = 106  // 英雄装备
	FUNCTION_ID_HERO_TRANSFORM                                = 107  // 英雄转职
	FUNCTION_ID_HERO_ACCESSORY                                = 108  // 英雄配饰
	FUNCTION_ID_HERO_FORMATION_FIRST_LOCATION                 = 109  // 英雄阵容第一位
	FUNCTION_ID_HERO_FORMATION_SECOND_LOCATION                = 110  // 英雄阵容第二位
	FUNCTION_ID_HERO_FORMATION_THIRD_LOCATION                 = 111  // 英雄阵容第三位
	FUNCTION_ID_HERO_FORMATION_FOURTH_LOCATION                = 112  // 英雄阵容第四位
	FUNCTION_ID_HERO_FORMATION_FIFTH_LOCATION                 = 113  // 英雄阵容第五位
	FUNCTION_ID_TOWER                                         = 301  // 爬塔
	FUNCTION_ID_ARENA                                         = 302  // 竞技场
	FUNCTION_ID_CAPSULE_INSTANCE                              = 303  // 胶囊副本
	FUNCTION_ID_HERO_INSTANCE                                 = 304  // 英雄副本
	FUNCTION_ID_COIN_INSTANCE                                 = 305  // 金币副本
	FUNCTION_ID_PET_INSTANCE                                  = 306  // 宠物副本
	FUNCTION_ID_GLORY_ARENA                                   = 307  // 荣耀擂台
	FUNCTION_ID_SUB_QUEST                                     = 601  // 支线任务
	FUNCTION_ID_DAILY_QUEST                                   = 602  // 日常任务
	FUNCTION_ID_WEEKLY_QUEST                                  = 603  // 周常任务
	FUNCTION_ID_STONE                                         = 701  // 传承石像
	FUNCTION_ID_ACCESSORY_LUCKY                               = 801  // 饰品召唤
	FUNCTION_ID_CALL_BOX                                      = 802  // 召
	FUNCTION_ID_AD_CHEST                                      = 803  // 广告宝箱
	FUNCTION_ID_SHOP_PRIVILIGE_PAGE                           = 1301 // 商城特权页
	FUNCTION_ID_PET                                           = 114  // 宠物
	FUNCTION_ID_PET_SANCTUARY                                 = 702  // 宠物招募
	FUNCTION_ID_COLLECTION                                    = 703  // 藏品楼
	FUNCTION_ID_EQUIPMENT                                     = 704  // 装备
	FUNCTION_ID_EXPEDITION                                    = 705  // 派遣
	FUNCTION_ID_LUMBERYARD                                    = 706  // 伐木场

	FUNCTION_ID_LOCK_SYSTEM = 9999999 // 永久锁定的系统
)

var AllFunctionId = map[int32]FunctionIdEnum{
	int32(FUNCTION_ID_NONE):                           FUNCTION_ID_NONE,
	int32(FUNCTION_ID_HERO):                           FUNCTION_ID_HERO,
	int32(FUNCTION_ID_PACK):                           FUNCTION_ID_PACK,
	int32(FUNCTION_ID_ADVANTURE):                      FUNCTION_ID_ADVANTURE,
	int32(FUNCTION_ID_DRAW_HERO_CARD):                 FUNCTION_ID_DRAW_HERO_CARD,
	int32(FUNCTION_ID_BOUNTY):                         FUNCTION_ID_BOUNTY,
	int32(FUNCTION_ID_MAIN_QUEST):                     FUNCTION_ID_MAIN_QUEST,
	int32(FUNCTION_ID_CAPITAL):                        FUNCTION_ID_CAPITAL,
	int32(FUNCTION_ID_IDLE):                           FUNCTION_ID_IDLE,
	int32(FUNCTION_ID_SUMMONING):                      FUNCTION_ID_SUMMONING,
	int32(FUNCTION_ID_MAIL):                           FUNCTION_ID_MAIL,
	int32(FUNCTION_ID_CHAT):                           FUNCTION_ID_CHAT,
	int32(FUNCTION_ID_ALLIANCE):                       FUNCTION_ID_ALLIANCE,
	int32(FUNCTION_ID_MYSTICREALM):                    FUNCTION_ID_MYSTICREALM,
	int32(FUNCTION_ID_CITYAGE):                        FUNCTION_ID_CITYAGE,
	int32(FUNCTION_ID_TRIAL):                          FUNCTION_ID_TRIAL,
	int32(FUNCTION_ID_BATTLE_SPEED):                   FUNCTION_ID_BATTLE_SPEED,
	int32(FUNCTION_ID_HERO_LEVEL_UP):                  FUNCTION_ID_HERO_LEVEL_UP,
	int32(FUNCTION_ID_HERO_FORMATION):                 FUNCTION_ID_HERO_FORMATION,
	int32(FUNCTION_ID_HERO_PACK):                      FUNCTION_ID_HERO_PACK,
	int32(FUNCTION_ID_HERO_DETAIL):                    FUNCTION_ID_HERO_DETAIL,
	int32(FUNCTION_ID_HERO_STAR):                      FUNCTION_ID_HERO_STAR,
	int32(FUNCTION_ID_HERO_EQUIP):                     FUNCTION_ID_HERO_EQUIP,
	int32(FUNCTION_ID_HERO_TRANSFORM):                 FUNCTION_ID_HERO_TRANSFORM,
	int32(FUNCTION_ID_HERO_ACCESSORY):                 FUNCTION_ID_HERO_ACCESSORY,
	int32(FUNCTION_ID_HERO_FORMATION_FIRST_LOCATION):  FUNCTION_ID_HERO_FORMATION_FIRST_LOCATION,
	int32(FUNCTION_ID_HERO_FORMATION_SECOND_LOCATION): FUNCTION_ID_HERO_FORMATION_SECOND_LOCATION,
	int32(FUNCTION_ID_HERO_FORMATION_THIRD_LOCATION):  FUNCTION_ID_HERO_FORMATION_THIRD_LOCATION,
	int32(FUNCTION_ID_HERO_FORMATION_FOURTH_LOCATION): FUNCTION_ID_HERO_FORMATION_FOURTH_LOCATION,
	int32(FUNCTION_ID_HERO_FORMATION_FIFTH_LOCATION):  FUNCTION_ID_HERO_FORMATION_FIFTH_LOCATION,
	int32(FUNCTION_ID_TOWER):                          FUNCTION_ID_TOWER,
	int32(FUNCTION_ID_ARENA):                          FUNCTION_ID_ARENA,
	int32(FUNCTION_ID_CAPSULE_INSTANCE):               FUNCTION_ID_CAPSULE_INSTANCE,
	int32(FUNCTION_ID_HERO_INSTANCE):                  FUNCTION_ID_HERO_INSTANCE,
	int32(FUNCTION_ID_COIN_INSTANCE):                  FUNCTION_ID_COIN_INSTANCE,
	int32(FUNCTION_ID_PET_INSTANCE):                   FUNCTION_ID_PET_INSTANCE,
	int32(FUNCTION_ID_GLORY_ARENA):                    FUNCTION_ID_GLORY_ARENA,
	int32(FUNCTION_ID_SUB_QUEST):                      FUNCTION_ID_SUB_QUEST,
	int32(FUNCTION_ID_DAILY_QUEST):                    FUNCTION_ID_DAILY_QUEST,
	int32(FUNCTION_ID_WEEKLY_QUEST):                   FUNCTION_ID_WEEKLY_QUEST,
	int32(FUNCTION_ID_STONE):                          FUNCTION_ID_STONE,
	int32(FUNCTION_ID_ACCESSORY_LUCKY):                FUNCTION_ID_ACCESSORY_LUCKY,
	int32(FUNCTION_ID_CALL_BOX):                       FUNCTION_ID_CALL_BOX,
	int32(FUNCTION_ID_AD_CHEST):                       FUNCTION_ID_AD_CHEST,
	int32(FUNCTION_ID_SHOP_PRIVILIGE_PAGE):            FUNCTION_ID_SHOP_PRIVILIGE_PAGE,
	int32(FUNCTION_ID_LOCK_SYSTEM):                    FUNCTION_ID_LOCK_SYSTEM,
	int32(FUNCTION_ID_ANNOUNCE):                       FUNCTION_ID_ANNOUNCE,
	int32(FUNCTION_ID_SHOP):                           FUNCTION_ID_SHOP,
	int32(FUNCTION_ID_VIP_CARD):                       FUNCTION_ID_VIP_CARD,
	int32(FUNCTION_ID_RANKBOARD):                      FUNCTION_ID_RANKBOARD,
	int32(FUNCTION_ID_PET):                            FUNCTION_ID_PET,
	int32(FUNCTION_ID_PET_SANCTUARY):                  FUNCTION_ID_PET_SANCTUARY,
	int32(FUNCTION_ID_COLLECTION):                     FUNCTION_ID_COLLECTION,
	int32(FUNCTION_ID_EQUIPMENT):                      FUNCTION_ID_EQUIPMENT,
	int32(FUNCTION_ID_EXPEDITION):                     FUNCTION_ID_EXPEDITION,
	int32(FUNCTION_ID_LUMBERYARD):                     FUNCTION_ID_LUMBERYARD,
}

// 功能ID枚举校验
func IsValidFunctionId(v int32) bool {
	_, ok := AllFunctionId[v]
	return ok
}

// 抽奖类型枚举
type LotteryType string

const (
	LOTTERY_TYPE_HERO    LotteryType = "hero"    // 英雄抽奖
	LOTTERY_TYPE_COLLECT LotteryType = "collect" // 藏品楼抽奖
)

var AllLotteryType = map[int32]string{
	1: string(LOTTERY_TYPE_HERO),
	2: string(LOTTERY_TYPE_COLLECT),
}

func GetSystemIdByInstanceId(instanceTyp InstanceTypeEnum) FunctionIdEnum {
	switch instanceTyp {
	case InstanceType_MAIN:
		return FUNCTION_ID_NONE
	case InstanceType_TOWER:
		return FUNCTION_ID_TOWER
	case InstanceType_CAPSULE:
		return FUNCTION_ID_CAPSULE_INSTANCE
	case InstanceType_HERO:
		return FUNCTION_ID_HERO_INSTANCE
	case InstanceType_COIN:
		return FUNCTION_ID_COIN_INSTANCE
	case InstanceType_PET:
		return FUNCTION_ID_PET_INSTANCE
	default:
		return FUNCTION_ID_LOCK_SYSTEM
	}
}

type FunctionStatusEnum int32

const (
	FUNCTION_STATUS_LOCK   FunctionStatusEnum = 0 // 锁定
	FUNCTION_STATUS_SHOW   FunctionStatusEnum = 1 // 可见
	FUNCTION_STATUS_UNLOCK FunctionStatusEnum = 2 // 解锁
)
