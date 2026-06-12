package enum

// 解锁类型枚举
type UnlockType int32

const (
	UNLOCK_TYPE_PLAYER_IN_MAIN_INSTANCE        UnlockType = iota + 1 // 玩家在主线副本中
	UNLOCK_TYPE_PLAYER_FINISH_MAIN_INSTANCE                          // 玩家完成主线副本
	UNLOCK_TYPE_PLAYER_IN_INSTANCE                                   // 玩家在特定副本中,unlockParam：具体副本类型
	UNLOCK_TYPE_PLAYER_FINISH_INSTANCE                               // 玩家完成特定副本,unlockParam：具体副本类型
	UNLOCK_TYPE_PLAYER_LEVEL                                         // 玩家等级
	UNLOCK_TYPE_PLAYER_VIP_LEVEL                                     // 玩家VIP等级
	UNLOCK_TYPE_PLAYER_IN_MAIN_TASK                                  // 玩家在主线任务中
	UNLOCK_TYPE_PLAYER_FINISH_MAIN_TASK                              // 玩家完成主线任务
	UNLOCK_TYPE_PLAYER_IN_SUB_TASK                                   // 玩家在支线任务中
	UNLOCK_TYPE_PLAYER_FINISH_SUB_TASK                               // 玩家完成支线任务
	UNLOCK_TYPE_SERVER_OPEN_TIME                                     // 服务器开放时间 unlockParam 0=天数,1=小时
	UNLOCK_TYPE_PLAYER_REGISTER_TIME                                 // 玩家注册时间 unlockParam 0=天数,1=小时
	UNLOCK_TYPE_SERVER_TIME                                          // 服务器时间 unlockParam 0=天数,1=小时
	UNLOCK_TYPE_SERVER_CURRENT_TIME                                  // 服务器当前时间 unlockParam 0=具体时间(2006-01-02 15:04:05),1=cron表达式（日 月 周）（不需要配置分钟和小时）
	UNLOCK_TYPE_SERVER_REGISTER_COUNT                                // 服务器注册人数 unlockParam 0: >=value 1: <=value
	UNLOCK_TYPE_SERVER_ACTIVE_PLAYER_COUNT                           // 玩家活跃人数 unlockParam 0: >=value 1: <=value
	UNLOCK_TYPE_ALLIANCE_MEMBER_COUNT                                // 联盟成员数 unlockParam 0: >=value 1: <=value
	UNLOCK_TYPE_PLAYER_CHARGE_COUNT                                  // 玩家充值数量 unlockParam 0: >=value 1: <=value
	UNLOCK_TYPE_PLAYER_CHARGE_TIMES                                  // 玩家充值次数 unlockParam 0: >=value 1: <=value
	UNLOCK_TYPE_PLAYER_CHARGE_DAY                                    // 玩家充值时间（天数）
	UNLOCK_TYPE_PLAYER_BUY_TARGET_SHOP_ITEM                          // 玩家购买礼包
	UNLOCK_TYPE_PLAYER_BUY_PRIVILEGE                                 // 玩家购买特权
	UNLOCK_TYPE_PLAYER_LOGIN_DAYS                                    // 玩家登录天数
	UNLOCK_TYPE_PLAYER_REGISTER_DAYS                                 // 玩家注册天数
	UNLOCK_TYPE_PLAYER_HERO_MIN_LEVEL                                // 玩家英雄等级
	UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_LEVEL                        // 过去玩家英雄等级
	UNLOCK_TYPE_PLAYER_HERO_MIN_STAR                                 // 玩家英雄星级
	UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_STAR                         // 过去玩家英雄星级
	UNLOCK_TYPE_PLAYER_HAVE_HERO                                     // 玩家有英雄
	UNLOCK_TYPE_PLAYER_DRAW_CARD_TIMES                               // 玩家抽卡次数
	UNLOCK_TYPE_PLAYER_FAILED_IN_MAIN_INSTANCE                       // 玩家在主线副本中失败 unlockParam -1: <= value 0: ==value 1: >=value value= stageId
	UNLOCK_TYPE_PLAYER_IN_PRIVILEGE                                  // 玩家在特权中

	UNLOCK_TYPE_PLAYER_HERO_LEVEL_UP_SUM_TODAY             // 今天英雄升级总次数
	UNLOCK_TYPE_PLAYER_WATCH_THE_ADVERTISEMENT             // 观看广告
	UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO                     // 是否抽到过英雄A
	UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO_TODAY               // 是否今天抽到英雄A
	UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY       // 是否首次抽到x品质英雄
	UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY_TODAY // 是否今天首次抽到x品质英雄
	UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_STAR                   // 某英雄当前最高星级
	UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_LEVEL                  // 某英雄当前最高等级
	UNLOCK_TYPE_PLAYER_ARCHITECTURE_LEVEL                  // 某建筑达到某级
	UNLOCK_TYPE_PLAYER_COLLECTION_NUM                      // 激活藏品数量
	UNLOCK_TYPE_ACTIVITY_OPEN_DAY                          // 活动开启天数
	UNLOCK_TYPE_PLAYER_CITY_AGE                            // 玩家到达主城时代
	UNLOCK_TYPE_PLAYER_PET_INSTANCE_FINISH                 // 玩家通关宠物副本
	UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_LOST             // 玩家擂台报名3负
	UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_WIN_COUNT        // 玩家擂台报名n胜
	UNLOCK_TYPE_PLAYER_GLORY_ARENA_FIRST_ENTER             // 玩家每日首次进入擂台
	UNLOCK_TYPE_PLAYER_PET_LOTTERY_DRAW_COUNT              // 玩家招募宠物次数
	UNLOCK_TYPE_PLAYER_EXPEDITION_COUNT                    // 玩家出征次数 param = 0:今天 param = 1:累计
	UNLOCK_TYPE_PLAYER_COLLECTION_LOTTERY_DRAW_COUNT       // 玩家藏品抽取次数 param = 0:今天 param = 1:累计

)

var serverUnlockType = map[UnlockType]bool{
	UNLOCK_TYPE_SERVER_OPEN_TIME:           true,
	UNLOCK_TYPE_SERVER_TIME:                true,
	UNLOCK_TYPE_SERVER_CURRENT_TIME:        true,
	UNLOCK_TYPE_SERVER_REGISTER_COUNT:      true,
	UNLOCK_TYPE_SERVER_ACTIVE_PLAYER_COUNT: true,
}

func IsValidUnlockType(v int32) bool {
	if v >= int32(UNLOCK_TYPE_PLAYER_IN_MAIN_INSTANCE) && v <= int32(UNLOCK_TYPE_PLAYER_COLLECTION_LOTTERY_DRAW_COUNT) {
		return true
	}
	return false
}

func IsServerUnlock(unlockType UnlockType) bool {
	if serverUnlockType[unlockType] {
		return true
	}
	return false
}
