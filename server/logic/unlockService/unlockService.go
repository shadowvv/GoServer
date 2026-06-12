package unlockService

import (
	"context"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/tool"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
)

type UnlockService struct {
	serverInfoService *gameServerInfoService.GameServerInfoService
	activityService   logicCommon.GameActivityServiceInterface
}

func (u *UnlockService) RecordExpedition(ctx context.Context, playerId int64) error {
	return DailyCache.RecordExpedition(ctx, playerId)
}

var _ logicCommon.UnlockServiceInterface = (*UnlockService)(nil)

func NewUnlockService(serverInfoService *gameServerInfoService.GameServerInfoService) *UnlockService {
	return &UnlockService{
		serverInfoService: serverInfoService,
	}
}

func (u *UnlockService) SetActivityService(activityService logicCommon.GameActivityServiceInterface) {
	u.activityService = activityService
}

func (u *UnlockService) CheckSystemUnlock(systemId int32, player logicCommon.PlayerInterface) bool {
	systemCfg := gameConfig.GetSystemUnlockCfg(systemId)
	if systemCfg == nil {
		return false
	}
	for _, unlockId := range systemCfg.UnlockId {
		if !u.CheckUnlock(unlockId, player) {
			return false
		}
	}
	return true
}

func (u *UnlockService) CheckServerInfoUnlock(unlockId int32, server logicCommon.ServerInfoInterface) bool {
	cfg := gameConfig.GetUnlockCfg(unlockId)
	if cfg == nil {
		return false
	}

	switch cfg.GetUnlockType() {
	case enum.UNLOCK_TYPE_SERVER_OPEN_TIME:
		return u.checkUnlockTypeServerOpenTime(cfg, server.GetServerId())
	case enum.UNLOCK_TYPE_SERVER_TIME:
		return u.checkUnlockTypeServerTime(cfg, server.GetServerId())
	case enum.UNLOCK_TYPE_SERVER_CURRENT_TIME:
		return u.checkUnlockTypeServerCurrentTime(cfg, server.GetServerId())
	case enum.UNLOCK_TYPE_SERVER_REGISTER_COUNT:
		return u.checkUnlockTypeServerRegisterCount(cfg, server.GetServerId())
	case enum.UNLOCK_TYPE_SERVER_ACTIVE_PLAYER_COUNT:
		return u.checkUnlockTypeServerActivePlayerCount(cfg, server.GetServerId())
	default:
		return false
	}
}

func (u *UnlockService) CheckUnlock(unlockId int32, p logicCommon.PlayerInterface) bool {
	//return true
	player, ok := p.(*model.PlayerModel)
	if !ok || player == nil {
		return false
	}

	cfg := gameConfig.GetUnlockCfg(unlockId)
	if cfg == nil {
		return false
	}

	switch cfg.GetUnlockType() {
	case enum.UNLOCK_TYPE_PLAYER_IN_MAIN_INSTANCE:
		return u.checkUnlockTypePlayerInMainInstance(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_FINISH_MAIN_INSTANCE:
		return u.checkUnlockTypePlayerFinishMainInstance(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IN_INSTANCE:
		return u.checkUnlockTypePlayerInInstance(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_FINISH_INSTANCE:
		return u.checkUnlockTypePlayerFinishInstance(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_LEVEL:
		return u.checkUnlockTypePlayerLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_VIP_LEVEL:
		return u.checkUnlockTypePlayerVipLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IN_MAIN_TASK:
		return u.checkUnlockTypePlayerInMainTask(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_FINISH_MAIN_TASK:
		return u.checkUnlockTypePlayerFinishMainTask(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IN_SUB_TASK:
		return u.checkUnlockTypePlayerInSubTask(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_FINISH_SUB_TASK:
		return u.checkUnlockTypePlayerFinishSubTask(cfg, player)
	case enum.UNLOCK_TYPE_SERVER_OPEN_TIME:
		return u.checkUnlockTypeServerOpenTime(cfg, player.GetUserServerId())
	case enum.UNLOCK_TYPE_PLAYER_REGISTER_TIME:
		return u.checkUnlockTypePlayerRegisterTime(cfg, player)
	case enum.UNLOCK_TYPE_SERVER_TIME:
		return u.checkUnlockTypeServerTime(cfg, player.GetUserServerId())
	case enum.UNLOCK_TYPE_SERVER_CURRENT_TIME:
		return u.checkUnlockTypeServerCurrentTime(cfg, player.GetUserServerId())
	case enum.UNLOCK_TYPE_SERVER_REGISTER_COUNT:
		return u.checkUnlockTypeServerRegisterCount(cfg, player.GetUserServerId())
	case enum.UNLOCK_TYPE_SERVER_ACTIVE_PLAYER_COUNT:
		return u.checkUnlockTypeServerActivePlayerCount(cfg, player.GetUserServerId())
	case enum.UNLOCK_TYPE_ALLIANCE_MEMBER_COUNT:
		return u.checkUnlockTypeAllianceMemberCount(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_CHARGE_COUNT:
		return u.checkUnlockTypePlayerChargeCount(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_CHARGE_TIMES:
		return u.checkUnlockTypePlayerChargeTimes(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_CHARGE_DAY:
		return u.checkUnlockTypePlayerChargeDay(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_BUY_TARGET_SHOP_ITEM:
		return u.checkUnlockTypePlayerBuyTargetShopItem(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_BUY_PRIVILEGE:
		return u.checkUnlockTypePlayerBuyPrivilege(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_LOGIN_DAYS:
		return u.checkUnlockTypePlayerLoginDays(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_REGISTER_DAYS:
		return u.checkUnlockTypePlayerRegisterDays(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_MIN_LEVEL:
		return u.checkUnlockTypePlayerHeroMinLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_LEVEL:
		return u.checkUnlockTypePlayerHeroHistoryMaxLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_MIN_STAR:
		return u.checkUnlockTypePlayerHeroMinStar(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_STAR:
		return u.checkUnlockTypePlayerHeroHistoryMaxStar(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_DRAW_CARD_TIMES:
		return u.checkUnlockTypePlayerDrawCardTimes(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_FAILED_IN_MAIN_INSTANCE:
		return u.checkUnlockTypePlayerFailedInMainInstance(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IN_PRIVILEGE:
		return u.checkUnlockTypePlayerInPrivilege(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO:
		return u.checkUnlockTypePlayerIsLotteryHero(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO_TODAY:
		return u.checkUnlockTypePlayerIsLotteryHeroToday(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY:
		return u.checkUnlockTypePlayerIsFirstLotteryHeroQuality(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY_TODAY:
		return u.checkUnlockTypePlayerIsFirstLotteryHeroQualityToday(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_LEVEL_UP_SUM_TODAY:
		return u.checkUnlockTypePlayerHeroLevelUpSumToday(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_STAR:
		return u.checkUnlockTypePlayerHeroStarUpNum(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HAVE_HERO:
		return u.checkUnlockTypePlayerHaveHero(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_LEVEL:
		return u.checkUnlockTypePlayerHeroNowMaxLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_ARCHITECTURE_LEVEL:
		return u.checkUnlockTypePlayerArchitectureLevel(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_COLLECTION_NUM:
		return u.checkUnlockTypePlayerCollectionNum(cfg, player)
	case enum.UNLOCK_TYPE_ACTIVITY_OPEN_DAY:
		return u.checkUnlockTypeActivityOpenDay(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_CITY_AGE:
		return u.checkUnlockTypePlayerCityAge(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_LOST:
		return u.checkUnlockTypeGloryArenaEnrollLost(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_WIN_COUNT:
		return u.checkUnlockTypeGloryArenaEnrollWinCount(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_FIRST_ENTER:
		return u.checkUnlockTypeGloryArenaFirstEnter(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_PET_LOTTERY_DRAW_COUNT:
		return u.checkUnlockTypePlayerPetLotteryDrawCount(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_COLLECTION_LOTTERY_DRAW_COUNT:
		return u.checkUnlockTypePlayerCollectionLotteryDrawCount(cfg, player)
	case enum.UNLOCK_TYPE_PLAYER_EXPEDITION_COUNT:
		return u.checkUnlockTypeExpeditionCount(cfg, player)
	default:
		return false
	}
}

func (u *UnlockService) checkUnlockTypePlayerInPrivilege(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return player.VipCardModel.GetFunctionValue(enum.VipPrivilegeType(unlock.UnlockParam), tool.UnixNowMilli()) > 0
}

func (u *UnlockService) checkUnlockTypePlayerFailedInMainInstance(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	stageId := player.PlayerInstanceModel.GetLastDeadMainInstanceStageId()
	if stageId == 0 {
		return false
	}
	if unlock.UnlockParam == 0 {
		return stageId == unlock.UnlockValue
	} else if unlock.UnlockParam > 0 {
		return stageId >= unlock.UnlockValue
	} else {
		return stageId <= unlock.UnlockValue
	}
}

func (u *UnlockService) checkUnlockTypePlayerInMainInstance(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	instanceData := player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instanceData == nil {
		return false
	}
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	stageId := unlock.UnlockValue
	if max(instanceData.MaxStageId, instanceData.MaxSubStageId) == stageId {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerFinishMainInstance(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	instanceData := player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instanceData == nil {
		return false
	}
	stageId := unlock.UnlockValue
	if instanceData.MaxStageId >= stageId {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerInInstance(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	instanceData := player.PlayerInstanceModel.InstanceEntities[unlock.UnlockParam]
	if instanceData == nil {
		return false
	}
	stageId := unlock.UnlockValue
	if instanceData.MaxStageId == stageId {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerFinishInstance(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	instanceData := player.PlayerInstanceModel.InstanceEntities[unlock.UnlockParam]
	if instanceData == nil {
		return false
	}
	stageId := unlock.UnlockValue
	if instanceData.MaxStageId >= stageId {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return player.ArchitectureModel.GetMainLevel() >= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerVipLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return player.User.GetVip() >= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerInMainTask(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	taskId := unlock.UnlockValue
	if taskId <= 0 {
		return false
	}
	if entity, ok := player.TaskModel.TaskEntity[enum.TaskAffiliationMain][gameConfig.GetTaskTypeByAttribution(enum.TaskAffiliationMain, taskId)][taskId]; ok {
		if entity.Status >= enum.TaskStatusFinishAndReward {
			return false
		}
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerFinishMainTask(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	taskId := unlock.UnlockValue
	for _, v := range player.TaskModel.TaskEntity[enum.TaskAffiliationMain] {
		for id, _ := range v {
			if id > taskId {
				return true
			}
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerInSubTask(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	//subId := unlock.UnlockParam
	taskId := unlock.UnlockValue
	if _, ok := player.TaskModel.TaskEntity[enum.TaskAffiliationSide][gameConfig.GetTaskTypeByAttribution(enum.TaskAffiliationSide, taskId)][taskId]; ok {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerFinishSubTask(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	subId := unlock.UnlockParam
	taskId := unlock.UnlockValue
	for _, taskIdMap := range player.TaskModel.TaskEntity[enum.TaskAffiliationSide] {
		for _, e := range taskIdMap {
			tCfg := gameConfig.GetSecondaryCfg(e.TaskID)
			if tCfg != nil && tCfg.TaskGroup == subId && e.TaskID >= taskId {
				return true
			} else {
				continue
			}
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypeServerOpenTime(cfg gameConfig.UnlockInterface, serverId int32) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	serverInfo := u.serverInfoService.GetServerInfo(serverId)
	if serverInfo == nil {
		return false
	}
	if unlock.UnlockParam == 0 {
		return tool.GetNatureDayDistance(tool.UnixNowMilli(), serverInfo.GetServerOpenTime()) >= unlock.UnlockValue
	} else {
		return tool.UnixNowMilli()-serverInfo.GetServerOpenTime() > int64(unlock.UnlockValue*3600*1000)
	}
}

func (u *UnlockService) checkUnlockTypePlayerRegisterTime(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		return tool.GetNatureDayDistance(tool.UnixNowMilli(), player.User.GetRegisterTime()) >= unlock.UnlockValue
	} else {
		return tool.UnixNowMilli()-player.User.GetRegisterTime() > int64(unlock.UnlockValue*3600*1000)
	}
}

func (u *UnlockService) checkUnlockTypeServerTime(cfg gameConfig.UnlockInterface, serverId int32) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	serverInfo := u.serverInfoService.GetServerInfo(serverId)
	if serverInfo == nil {
		return false
	}
	if unlock.UnlockParam == 0 {
		return tool.GetNatureDayDistance(tool.UnixNowMilli(), serverInfo.GetServerTime())+1 >= unlock.UnlockValue
	} else if unlock.UnlockParam == 1 {
		return tool.UnixNowMilli()-serverInfo.GetServerTime() >= int64(unlock.UnlockValue)*24*3600*1000
	}
	return false
}

func (u *UnlockService) checkUnlockTypeActivityOpenDay(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if u.activityService == nil {
		return false
	}
	act := u.activityService.IsActivityOpen(player.GetUserServerId(), unlock.UnlockParam)
	if act == nil {
		return false
	}
	return tool.GetNatureDayDistance(tool.UnixNowMilli(), act.GetOpenTime())+1 >= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypeGloryArenaEnrollLost(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	return player.PlayerGloryArenaModel.IsLose
}

func (u *UnlockService) checkUnlockTypeGloryArenaEnrollWinCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return player.PlayerGloryArenaModel.GetWinCount() == int32(unlock.UnlockValue)
}
func (u *UnlockService) checkUnlockTypeGloryArenaFirstEnter(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	return player.PlayerGloryArenaModel.EnterCount == 1
}

func (u *UnlockService) checkUnlockTypePlayerCityAge(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if player.CityAgeModel == nil || player.CityAgeModel.Entity == nil {
		return false
	}
	return gameConfig.IsCityAgeUpReached(player.CityAgeModel.Entity.AgeId, unlock.UnlockValue)
}

func (u *UnlockService) checkUnlockTypeServerCurrentTime(cfg gameConfig.UnlockInterface, serverId int32) bool {
	unlock := cfg.(*gameConfig.UnlockTimeValueBase)
	serverInfo := u.serverInfoService.GetServerInfo(serverId)
	if serverInfo == nil {
		return false
	}
	if unlock.UnlockParam == 0 {
		return tool.UnixNowMilli() > unlock.UnlockValue
	} else {
		result, err := tool.CheckCronMatch(unlock.Cron, tool.UnixNowMilli())
		if err != nil {
			return false
		}
		return result
	}
}

func (u *UnlockService) checkUnlockTypeServerRegisterCount(cfg gameConfig.UnlockInterface, serverId int32) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	serverInfo := u.serverInfoService.GetServerInfo(serverId)
	if serverInfo == nil {
		return false
	}
	if unlock.UnlockParam == 0 {
		return serverInfo.GetRegisterCount() >= unlock.UnlockValue
	}
	return serverInfo.GetRegisterCount() <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypeServerActivePlayerCount(cfg gameConfig.UnlockInterface, serverId int32) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	serverInfo := u.serverInfoService.GetServerInfo(serverId)
	if serverInfo == nil {
		return false
	}
	if unlock.UnlockParam == 0 {
		return serverInfo.GetActivePlayerCount() >= unlock.UnlockValue
	}
	return serverInfo.GetActivePlayerCount() <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypeAllianceMemberCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	// TODO: 实现联盟成员数检查逻辑
	// 使用 cfg.GetUnlockParam() 0: >=value 1: <=value
	return false
}

func (u *UnlockService) checkUnlockTypePlayerChargeCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		return player.User.GetChargeCount() >= unlock.UnlockValue
	}
	return player.User.GetChargeCount() <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerChargeTimes(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		return player.StaticData.GetChargeTimes() >= unlock.UnlockValue
	} else {
		return player.StaticData.GetChargeTimes() <= unlock.UnlockValue
	}
}

func (u *UnlockService) checkUnlockTypePlayerChargeDay(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return tool.GetNatureDayDistance(tool.UnixNowMilli(), player.User.GetLastChargeTime()) <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerBuyTargetShopItem(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	item := player.PlayerShopModel.GetShopItemInfo(unlock.UnlockValue)
	if item == nil {
		return false
	}
	if item.LastBuyTime == 0 {
		return false
	}
	return true
}

func (u *UnlockService) checkUnlockTypePlayerBuyPrivilege(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	// 实现玩家购买特权检查逻辑
	cards, err := vipCard.Service.GetActiveVipCards(player)
	if err != nil {
		return false
	}
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	for _, card := range cards {
		if card.ItemId == unlock.UnlockValue && (card.ExpireTime == -1 || tool.UnixNowMilli() <= card.ExpireTime) {
			return true
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerLoginDays(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return tool.GetNatureDayDistance(tool.UnixNowMilli(), player.User.GetLastLoginTime()) <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerRegisterDays(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	return tool.GetNatureDayDistance(tool.UnixNowMilli(), player.User.GetRegisterTime()) <= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerHeroMinLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	// TODO: 实现玩家英雄最小等级检查逻辑
	return false
}

func (u *UnlockService) checkUnlockTypePlayerHeroHistoryMaxLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	targetLevel := unlock.UnlockValue
	heroId := unlock.UnlockParam // UnlockParam 为0表示任意英雄，否则表示指定英雄ID
	// 检查历史最大等级（存储在 StaticData 中）
	if heroId == 0 {
		return player.StaticData.Entity.HeroHistoryMaxLevel >= targetLevel
	} else {
		return player.HeroAlbumModel.Entities[int64(heroId)].HistoryMaxLevel >= targetLevel
	}
}

func (u *UnlockService) checkUnlockTypePlayerHeroMinStar(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	// TODO: 实现玩家英雄最小星级检查逻辑
	return false
}

func (u *UnlockService) checkUnlockTypePlayerHeroHistoryMaxStar(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	targetStar := unlock.UnlockValue
	heroId := unlock.UnlockParam // UnlockParam 为0表示任意英雄，否则表示指定英雄ID
	// 检查历史最大星级
	if heroId == 0 {
		// 检查任意英雄的历史最大星级
		for _, album := range player.HeroAlbumModel.Entities {
			if album.HistoryMaxStar >= targetStar {
				return true
			}
		}
	} else {
		// 检查指定英雄的历史最大星级
		if album := player.HeroAlbumModel.Entities[int64(heroId)]; album != nil {
			return album.HistoryMaxStar >= targetStar
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerHaveHero(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	heroId := unlock.UnlockValue
	// 检查玩家是否拥有指定英雄（通过图鉴判断）
	if player.HeroAlbumModel.Entities[int64(heroId)] != nil {
		return true
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerDrawCardTimes(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	lotteryNum := unlock.UnlockValue // 抽卡次数
	lotteryId := unlock.UnlockParam  // 抽卡卡池ID
	// 检查指定卡池的抽卡次数
	if player.LotteryModel.LotteryEntities[lotteryId] != nil {
		if player.LotteryModel.LotteryEntities[lotteryId].AllCount >= lotteryNum {
			return true
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerIsLotteryHero(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	heroId := unlock.UnlockValue

	// 使用内存缓存快速查询 O(1)
	if player.LotteryModel.LotteryHistoryDetail == nil {
		return false
	}
	return player.LotteryModel.LotterySet[1][heroId] // 1表示指定英雄
}

func (u *UnlockService) checkUnlockTypePlayerIsLotteryHeroToday(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	heroId := unlock.UnlockValue
	// 优先从 Redis 查询今天抽到的英雄
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyLotteryHeroKey(today)
	userField := strconv.FormatInt(player.GetUserId(), 10)
	heroSetKey := redisKey + ":" + userField

	// 使用 SISMEMBER 命令检查是否存在
	ctx := context.Background()
	exists, err := dbService.RDB.SIsMember(ctx, heroSetKey, heroId).Result()
	if err == nil && exists {
		return true
	}

	// 如果 Redis 中没有，则回退到内存查询
	if player.LotteryModel.LotteryHistoryDetail == nil {
		return false
	}
	for _, logs := range player.LotteryModel.LotteryHistoryDetail {
		for _, log := range logs {
			if log.ItemId == heroId && tool.GetNatureDayDistance(tool.UnixNowMilli(), log.AddTime) == 0 {
				return true
			}
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerIsFirstLotteryHeroQuality(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	quality := unlock.UnlockValue

	// 使用内存缓存快速查询 O(1)
	if player.LotteryModel.LotteryHistoryDetail == nil {
		return false
	}
	return player.LotteryModel.LotteryQualitySet[1][quality] // 1表示英雄
}

func (u *UnlockService) checkUnlockTypePlayerIsFirstLotteryHeroQualityToday(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	quality := unlock.UnlockValue

	// 优先从 Redis 查询今天抽到的指定品质物品
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyLotteryQualityKey(today, quality)
	userField := strconv.FormatInt(player.GetUserId(), 10)
	qualitySetKey := redisKey + ":" + userField

	// 使用 SCARD 命令检查集合是否有元素
	ctx := context.Background()
	count, err := dbService.RDB.SCard(ctx, qualitySetKey).Result()
	if err == nil && count > 0 {
		return true
	}

	// 如果 Redis 中没有，则回退到内存查询
	if player.LotteryModel.LotteryHistoryDetail == nil {
		return false
	}
	for _, logs := range player.LotteryModel.LotteryHistoryDetail {
		for _, log := range logs {
			if tool.GetNatureDayDistance(tool.UnixNowMilli(), log.AddTime) == 0 {
				// 根据 itemId 获取物品品质
				itemCfg := gameConfig.GetItemCfg(log.ItemId)
				if itemCfg != nil && itemCfg.Quality == quality {
					return true
				}
			}
		}
	}
	return false
}

func (u *UnlockService) checkUnlockTypePlayerHeroLevelUpSumToday(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	targetSum := unlock.UnlockValue

	// 优先从 Redis 查询今天的英雄升级总次数
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyHeroLevelUpKey(today)
	userField := strconv.FormatInt(player.GetUserId(), 10)

	// 使用 HGET 命令获取升级次数
	ctx := context.Background()
	countStr, err := dbService.RDB.HGet(ctx, redisKey, userField).Result()
	if err == nil {
		count, _ := strconv.ParseInt(countStr, 10, 32)
		return int32(count) >= targetSum
	}

	// 如果 Redis 中没有，返回 false（需要配合事件系统实时统计）
	// 注意：这个功能需要配合任务系统或事件系统统计
	// 在英雄升级时，需要调用 DailyDataCache.RecordHeroLevelUp 方法
	return false
}

func (u *UnlockService) checkUnlockTypePlayerHeroStarUpNum(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	targetStar := unlock.UnlockValue
	heroId := unlock.UnlockParam

	if heroId == 0 {
		for _, v := range player.HeroDetailsModel.HeroMaxStarCache {
			if v >= targetStar {
				return true
			}
		}
	}
	// 使用缓存 O(1)
	return player.HeroDetailsModel.HeroMaxStarCache[int64(heroId)] >= targetStar
}

func (u *UnlockService) checkUnlockTypePlayerHeroNowMaxLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	targetLevel := unlock.UnlockValue
	heroId := unlock.UnlockParam

	if heroId == 0 {
		for _, v := range player.HeroDetailsModel.HeroMaxLevelCache {
			if v >= targetLevel {
				return true
			}
		}
	}
	// 使用缓存 O(1)
	return player.HeroDetailsModel.HeroMaxLevelCache[int64(heroId)] >= targetLevel
}

func (u *UnlockService) checkUnlockTypePlayerArchitectureLevel(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	level := unlock.UnlockValue
	arType := unlock.UnlockParam
	if arType == 0 {
		return false
	}
	if player.ArchitectureModel.Entities[arType] == nil {
		return false
	}
	return player.ArchitectureModel.Entities[arType].Level >= level
}

func (u *UnlockService) checkUnlockTypePlayerCollectionNum(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	collectionNum := unlock.UnlockValue
	if collectionNum == 0 {
		return false
	}
	return int32(len(player.CollectionModel.CollectionEntity)) >= collectionNum
}

func (u *UnlockService) checkUnlockTypePlayerPetLotteryDrawCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		count, _ := DailyCache.GetDailyPetRecruitCount(context.Background(), player.GetUserId())
		return count >= unlock.UnlockValue
	}
	return player.StaticData.GetPetRecruitCount() >= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypePlayerCollectionLotteryDrawCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		count, _ := DailyCache.GetDailyCollectionLotteryCount(context.Background(), player.GetUserId())
		return count >= unlock.UnlockValue
	}
	return player.StaticData.GetCollectionLotteryDrawCount() >= unlock.UnlockValue
}

func (u *UnlockService) checkUnlockTypeExpeditionCount(cfg gameConfig.UnlockInterface, player *model.PlayerModel) bool {
	unlock := cfg.(*gameConfig.UnlockIntValueBase)
	if unlock.UnlockParam == 0 {
		count, err := DailyCache.GetDailyExpeditionCount(context.Background(), player.GetUserId())
		if err != nil {
			return false
		}
		return count >= unlock.UnlockValue
	}

	return player.StaticData.GetExpeditionNum() >= unlock.UnlockValue
}
