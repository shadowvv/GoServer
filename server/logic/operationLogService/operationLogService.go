package operationLogService

import (
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type UseOperationType int32

const (
	OP_TYPE_MAIN_INSTANCE                  UseOperationType = iota // 主线关卡
	OP_TYPE_TOWER                                                  // 试炼之塔
	OP_TYPE_ARENA                                                  // 竞技场
	OP_TYPE_BEGIN_PAY                                              // 开始支付
	OP_TYPE_PAY_SUCCESS                                            // 支付成功
	OP_TYPE_EQUIPMENT_OPERATION                                    // 装备操作
	OP_TYPE_IDLE_REWARD_CLAIM                                      // 挂机奖励领取
	OP_TYPE_FINISH_TASK                                            // 任务完成
	OP_TYPE_ARCHITECTURE                                           // 建筑升级
	OP_TYPE_HERO_LEVEL                                             // 英雄升级
	OP_TYPE_HERO_STAR                                              // 英雄升星
	OP_TYPE_HERO_EVOLUTION                                         // 英雄专职
	OP_TYPE_HERO_FORMATION                                         // 英雄上阵 formations
	OP_TYPE_ACCESSORY_LUCKY                                        // 收拾召唤
	OP_TYPE_OPEN_LOOP_BOX                                          // 开秘罐
	OP_TYPE_HERO_LOTTERY                                           // 英雄招募
	OP_TYPE_EQUIPMENT_OPERATION_2_0_forge                          // 装备操作2.0 打造装备
	OP_TYPE_EQUIPMENT_OPERATION_2_0_refine                         // 装备操作2.0 强化装备
	OP_TYPE_COLLECTION_SYSTEM_STARDUST                             // 藏品升星
	OP_TYPE_COLLECTION_SYSTEM_ACTIVATE                             // 词条激活
	OP_TYPE_COLLECTION_SYSTEM_UPGRADE                              // 藏品升级
	OP_TYPE_TRIAL_FINISH_TASK                                      // 试炼任务完成
	OP_TYPE_PET_RECRUIT                                            // 宠物招募
	OP_TYPE_PET_LEVEL_UP                                           // 宠物升级
	OP_TYPE_PET_STAR_UP                                            // 宠物升星
	OP_TYPE_ADVENTURE_INSTANCE_ENTER                               // 奇遇副本进入
	OP_TYPE_ADVENTURE_INSTANCE_LEAVE                               // 奇遇副本离开
	OP_TYPE_EXPEDITION_START                                       // 开始派遣
	OP_TYPE_EXPEDITION_CANCEL                                      // 撤回派遣
)

const (
	ARENA_OPER_CHALLENGE     = 1 // 挑战
	ARENA_OPER_BE_CHALLENGED = 2 // 被挑战
	ARENA_OPER_INIT          = 3 // 初始化
)

func userOperationLog(uid int64, operationType UseOperationType, param1, param2, param3, param4 int64) {
	logData := map[string]interface{}{
		"user_id":        uid,
		"add_time":       tool.UnixNowMilli(),
		"operation_type": int32(operationType),
		"param1":         param1,
		"param2":         param2,
		"param3":         param3,
		"param4":         param4,
	}
	err := easyDB.LogCreatEntity(logData, easyDB.OPER_LOG)
	if err != nil {
		return
	}
}

func OnUserMainInstanceChange(uid int64, isWin, stageId int32) {
	userOperationLog(uid, OP_TYPE_MAIN_INSTANCE, int64(isWin), int64(stageId), 0, 0)
}

func OnUserTowerChange(uid int64, isWin, stageId int32) {
	userOperationLog(uid, OP_TYPE_TOWER, int64(isWin), int64(stageId), 0, 0)
}

func OnUserArenaChange(uid int64, operType, isWin, beforeScore, changeScore int32) {
	userOperationLog(uid, OP_TYPE_ARENA, int64(operType), int64(isWin), int64(beforeScore), int64(changeScore))
}

func OnUserBeginPay(uid int64, payId int32, dollar int32) {
	userOperationLog(uid, OP_TYPE_BEGIN_PAY, int64(payId), int64(dollar), 0, 0)
}

func OnUserPaySuccess(uid int64, payId, dollar int32) {
	userOperationLog(uid, OP_TYPE_PAY_SUCCESS, int64(payId), int64(dollar), 0, 0)
}

func OnUserEquipmentOperation(uid int64, heroId, slotIndex, beforeEquipmentId, afterEquipmentId int32) {
	userOperationLog(uid, OP_TYPE_EQUIPMENT_OPERATION, int64(heroId), int64(slotIndex), int64(beforeEquipmentId), int64(afterEquipmentId))
}

func OnUserIdleRewardClaim(uid int64, claimType int32) {
	userOperationLog(uid, OP_TYPE_IDLE_REWARD_CLAIM, int64(claimType), 0, 0, 0)
}

func OnUserFinishTask(uid int64, attribution, taskId int32) {
	userOperationLog(uid, OP_TYPE_FINISH_TASK, int64(attribution), int64(taskId), 0, 0)
}

func OnUserArchitecture(uid int64, archId, beforeLevel, afterLevel int32) {
	userOperationLog(uid, OP_TYPE_ARCHITECTURE, int64(archId), int64(beforeLevel), int64(afterLevel), 0)
}

func OnUserHeroLevel(uid int64, heroId int32, heroOwnId int64, beforeHeroLevel, afterHeroLevel int32) {
	userOperationLog(uid, OP_TYPE_HERO_LEVEL, int64(heroId), heroOwnId, int64(beforeHeroLevel), int64(afterHeroLevel))
}

func OnUserHeroStar(uid int64, heroId int32, heroOwnId int64, beforeHeroStar, afterHeroStar int32) {
	userOperationLog(uid, OP_TYPE_HERO_STAR, int64(heroId), heroOwnId, int64(beforeHeroStar), int64(afterHeroStar))
}

func OnUserHeroEvolutation(uid int64, heroId int32, heroOwnId int64, beforeHeroEvo, afterHeroEvo int32) {
	userOperationLog(uid, OP_TYPE_HERO_EVOLUTION, int64(heroId), heroOwnId, int64(beforeHeroEvo), int64(afterHeroEvo))
}

func OnUserHeroFormation(uid int64, formationType int32, operationType int32, heroId int32, heroOwnId int64) {
	userOperationLog(uid, OP_TYPE_HERO_FORMATION, int64(formationType), int64(operationType), int64(heroId), heroOwnId)
}

func OnUserAccessoryLucky(uid int64, luckyNum int32) {
	userOperationLog(uid, OP_TYPE_ACCESSORY_LUCKY, int64(luckyNum), 0, 0, 0)
}

func OnUserOpenLoopBox(uid int64, boxType int32, num int32) {
	userOperationLog(uid, OP_TYPE_OPEN_LOOP_BOX, int64(boxType), int64(num), 0, 0)
}

func OnUserHeroLottery(uid int64, lotteryNum int32) {
	userOperationLog(uid, OP_TYPE_HERO_LOTTERY, int64(lotteryNum), 0, 0, 0)
}

func OnUserEquipmentOperation2_0Forge(uid int64, equipmentId int32) {
	userOperationLog(uid, OP_TYPE_EQUIPMENT_OPERATION_2_0_forge, int64(equipmentId), 0, 0, 0)
}

func OnUserEquipmentOperation2_0Refine(uid int64, isSuccess int32, equipmentId int32, beforeLevel int32, afterLevel int32) {
	userOperationLog(uid, OP_TYPE_EQUIPMENT_OPERATION_2_0_refine, int64(isSuccess), int64(equipmentId), int64(beforeLevel), int64(afterLevel))
}

func OnUserCollectionSystemStarDust(uid int64, level int32, beforeLevel int32, afterLevel int32) {
	userOperationLog(uid, OP_TYPE_COLLECTION_SYSTEM_STARDUST, int64(level), int64(beforeLevel), int64(afterLevel), 0)
}

func OnUserCollectionEntryActive(uid int64, entryId int32) {
	userOperationLog(uid, OP_TYPE_COLLECTION_SYSTEM_ACTIVATE, int64(entryId), 0, 0, 0)
}

func OnUserCollectionEntryUpLevel(uid int64, entryId int32, beforeLevel int32, afterLevel int32) {
	userOperationLog(uid, OP_TYPE_COLLECTION_SYSTEM_UPGRADE, int64(entryId), int64(beforeLevel), int64(afterLevel), 0)
}

func OnUserTrialFinishTask(uid int64, taskId int32) {
	userOperationLog(uid, OP_TYPE_TRIAL_FINISH_TASK, int64(taskId), 0, 0, 0)
}

func OnUserPetRecruit(uid int64, petId int32, petOwnId int64) {
	userOperationLog(uid, OP_TYPE_PET_RECRUIT, int64(petId), petOwnId, 0, 0)
}

func OnUserPetLevelUp(uid int64, petId int32, petOwnId int64, beforeLevel int32, afterLevel int32) {
	userOperationLog(uid, OP_TYPE_PET_LEVEL_UP, int64(petId), petOwnId, int64(beforeLevel), int64(afterLevel))
}

func OnUserPetStarUp(uid int64, petId int32, petOwnId int64, beforeStar int32, afterStar int32) {
	userOperationLog(uid, OP_TYPE_PET_STAR_UP, int64(petId), petOwnId, int64(beforeStar), int64(afterStar))
}

func OnUserAdventureInstanceEnter(uid int64, stageId int32) {
	userOperationLog(uid, OP_TYPE_ADVENTURE_INSTANCE_ENTER, int64(stageId), 0, 0, 0)
}

func OnUserAdventureInstanceLeave(uid int64, stageId int32) {
	userOperationLog(uid, OP_TYPE_ADVENTURE_INSTANCE_LEAVE, int64(stageId), 0, 0, 0)
}

func OnUserExpeditionStart(uid int64, monsterId int32) {
	userOperationLog(uid, OP_TYPE_EXPEDITION_START, int64(monsterId), 0, 0, 0)
}

func OnUserExpeditionCancel(uid int64, monsterId int32) {
	userOperationLog(uid, OP_TYPE_EXPEDITION_CANCEL, int64(monsterId), 0, 0, 0)
}
