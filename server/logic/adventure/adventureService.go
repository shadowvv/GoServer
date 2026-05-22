package adventure

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"
)

var messageSender logicCommon.MessageSenderInterface
var unlockService logicCommon.UnlockServiceInterface

// InitAdventureService 注入消息发送器，用于秘境信息变化推送。
func InitAdventureService(sender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface) {
	messageSender = sender
	unlockService = unlock
}

// OnMainStageKillMonster 处理主线杀怪带来的每日秘境进度与入口触发。
func OnMainStageKillMonster(player *model.PlayerModel, monsterCount int32) {
	if player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil || monsterCount <= 0 {
		return
	}
	if !unlockService.CheckSystemUnlock(int32(enum.FUNCTION_ID_MYSTICREALM), player) {
		return
	}
	m := player.PlayerAdventureModel
	now := tool.UnixNowMilli()
	m.ResetDaily(now)
	removeExpiredEntries(player, m, now)
	if m.Entity.DailySettleCount >= gameConfig.GetDailyAdventureLimit() || int32(len(m.GetActiveEntries(now))) >= gameConfig.GetRealmStorageCap() {
		return
	}

	progress := int32(0)
	base := gameConfig.GetMonsterAdventureProgressValue()
	for i := int32(0); i < monsterCount; i++ {
		progress += base * tool.RandInt32(100, 200) / 100
	}
	m.AddProgress(progress)

	progressCfg := gameConfig.GetAdventureProgressCfgByDailyTriggerCount(m.Entity.DailyTriggerCount)
	if progressCfg == nil || m.Entity.Progress < progressCfg.Progress {
		return
	}
	adventureId := getTriggerAdventureId(player, m, progressCfg)
	if adventureId == 0 {
		return
	}
	dungeonId := getAdventureDungeonId(adventureId, getCurrentMainStageId(player))
	if dungeonId == 0 {
		return
	}
	m.SetProgress(0)
	uniqueId := m.AddEntry(adventureId, dungeonId, now, now+int64(gameConfig.GetTimeLimitedAdventure())*1000)
	m.AddDailyTriggerCount()
	platformLogger.InfoWithUser(fmt.Sprintf("[adventure] entry created uniqueId:%s,adventureId:%d,dungeonId:%d,dailyTriggerCount:%d", uniqueId, adventureId, dungeonId, m.Entity.DailyTriggerCount), player)
}

// BuildAdventureInfoResp 构建客户端打开秘境外层界面所需的当前状态。
func BuildAdventureInfoResp(player *model.PlayerModel) *pb.AdventureInfoResp {
	if player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil {
		return &pb.AdventureInfoResp{
			Entries: make([]*pb.AdventureEntryInfo, 0),
		}
	}
	m := player.PlayerAdventureModel
	now := tool.UnixNowMilli()
	m.ResetDaily(now)
	removeExpiredEntries(player, m, now)
	return &pb.AdventureInfoResp{
		Progress:          m.Entity.Progress,
		DailyTriggerCount: m.Entity.DailyTriggerCount,
		DailySettleCount:  m.Entity.DailySettleCount,
		Entries:           buildAdventureEntryInfos(m.GetActiveEntries(now)),
	}
}

// StartAdventure 将指定秘境入口切到战斗中状态，并生成通用副本框架需要的 raid 数据。
func StartAdventure(player *model.PlayerModel, uniqueId string) (*logicCommon.PlayerInstanceRaid, pb.ERROR_CODE) {
	if player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	m := player.PlayerAdventureModel
	now := tool.UnixNowMilli()
	m.ResetDaily(now)
	removeExpiredEntries(player, m, now)

	entry := m.GetEntry(uniqueId, now)
	if entry == nil {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if !m.CanSettle(entry.AdventureId) {
		platformLogger.InfoWithUser(fmt.Sprintf("[adventure] start rejected by daily limit uniqueId:%s,adventureId:%d,dailySettleCount:%d", uniqueId, entry.AdventureId, m.Entity.DailySettleCount), player)
		return nil, pb.ERROR_CODE_STAGE_HAS_SETTLE
	}
	dungeonCfg := gameConfig.GetDungeonAdventureCfg(entry.DungeonId)
	if dungeonCfg == nil {
		return nil, pb.ERROR_CODE_CFG_NOT_FOUND
	}
	if dungeonCfg.Type != entry.AdventureId {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	adventureCfg := gameConfig.GetAdventureCfg(entry.AdventureId)
	if adventureCfg == nil {
		return nil, pb.ERROR_CODE_CFG_NOT_FOUND
	}
	if !m.MarkEntryFighting(uniqueId, now) {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	platformLogger.InfoWithUser(fmt.Sprintf("[adventure] start uniqueId:%s,adventureId:%d,dungeonId:%d", uniqueId, entry.AdventureId, entry.DungeonId), player)

	return &logicCommon.PlayerInstanceRaid{
		PlayerId:          player.GetUserId(),
		BattleId:          0,
		InstanceID:        enum.ADVENTURE_INSTANCE_ID,
		CurrentStageId:    entry.DungeonId,
		CurrentSubStageId: 1,
		BattleEndTime:     now + int64(adventureCfg.TimeLimit)*1000,
		SubStageInfo:      make(map[int32]*logicCommon.SubStageData),
		SubStageIds:       make([]int32, 0),
		MonsterTemplates:  make(map[int64]*logicCommon.MonsterTemplate),
		StageInfo:         logicCommon.NewInstanceStageInfo(),
		AdventureUniqueId: uniqueId,
	}, pb.ERROR_CODE_SUCCESS
}

// CancelStartAdventure 在开打流程未完成时回滚入口状态；进入战斗后的输赢都走结算。
func CancelStartAdventure(player *model.PlayerModel, uniqueId string) {
	if player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil {
		return
	}
	player.PlayerAdventureModel.MarkEntryWait(uniqueId)
}

// OnAdventureSettle 按结算发生时的自然日扣除每日总次数和类型次数。
func OnAdventureSettle(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid) pb.ERROR_CODE {
	if player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil || raidInfo == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if raidInfo.AdventureUniqueId == "" {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	m := player.PlayerAdventureModel
	now := tool.UnixNowMilli()
	m.ResetDaily(now)

	dungeonCfg := gameConfig.GetDungeonAdventureCfg(raidInfo.CurrentStageId)
	if dungeonCfg == nil {
		return pb.ERROR_CODE_CFG_NOT_FOUND
	}
	for _, entry := range m.GetEntries() {
		if entry == nil || entry.UniqueId != raidInfo.AdventureUniqueId || entry.DungeonId != raidInfo.CurrentStageId || entry.Status != model.ADVENTURE_ENTRY_STATUS_FIGHTING {
			continue
		}
		if dungeonCfg.Type != entry.AdventureId {
			return pb.ERROR_CODE_INVALID_REQUEST_PARAM
		}
		if !m.CanSettle(dungeonCfg.Type) {
			platformLogger.InfoWithUser(fmt.Sprintf("[adventure] settle rejected by daily limit uniqueId:%s,adventureId:%d,dailySettleCount:%d", entry.UniqueId, dungeonCfg.Type, m.Entity.DailySettleCount), player)
			return pb.ERROR_CODE_STAGE_HAS_SETTLE
		}
		m.AddSettleCount(dungeonCfg.Type)
		m.MarkEntrySettled(entry.UniqueId)
		pushChange := false
		if !m.CanSettle(dungeonCfg.Type) {
			count := m.RemoveWaitEntriesByAdventureId(dungeonCfg.Type)
			if count > 0 {
				pushChange = true
				platformLogger.InfoWithUser(fmt.Sprintf("[adventure] clear wait entries by adventure limit adventureId:%d,count:%d", dungeonCfg.Type, count), player)
			}
		}
		if m.Entity.DailySettleCount >= gameConfig.GetDailyAdventureLimit() {
			m.SetProgress(0)
			count := m.RemoveWaitEntries()
			if count > 0 {
				pushChange = true
				platformLogger.InfoWithUser(fmt.Sprintf("[adventure] clear wait entries by daily settle limit count:%d", count), player)
			}
		}
		platformLogger.InfoWithUser(fmt.Sprintf("[adventure] settle uniqueId:%s,adventureId:%d,dungeonId:%d,dailySettleCount:%d", entry.UniqueId, dungeonCfg.Type, entry.DungeonId, m.Entity.DailySettleCount), player)
		if pushChange {
			PushAdventureInfoChange(player)
		}
		return pb.ERROR_CODE_SUCCESS
	}
	return pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

// PushAdventureInfoChange 推送秘境外层状态变化，当前只在主线 stage 变化时调用。
func PushAdventureInfoChange(player *model.PlayerModel) {
	if messageSender == nil || player == nil || player.PlayerAdventureModel == nil || player.PlayerAdventureModel.Entity == nil {
		return
	}
	if !unlockService.CheckSystemUnlock(int32(enum.FUNCTION_ID_MYSTICREALM), player) {
		return
	}
	m := player.PlayerAdventureModel
	now := tool.UnixNowMilli()
	removeExpiredEntries(player, m, now)
	messageSender.SendMessageByPlayerId(player.GetUserId(), pb.MESSAGE_ID_PUSH_ADVENTURE_INFO_CHANGE, &pb.PushAdventureInfoChange{
		Progress:          m.Entity.Progress,
		DailyTriggerCount: m.Entity.DailyTriggerCount,
		Entries:           buildAdventureEntryInfos(m.GetActiveEntries(now)),
	})
}

func OnMainStageChanged(player *model.PlayerModel) {
	if player.PlayerAdventureModel.Entity.DailySettleCount >= gameConfig.GetDailyAdventureLimit() {
		return
	}
	PushAdventureInfoChange(player)
}

func removeExpiredEntries(player *model.PlayerModel, m *model.PlayerAdventureModel, now int64) {
	count := m.RemoveExpiredEntries(now)
	if count > 0 {
		platformLogger.InfoWithUser(fmt.Sprintf("[adventure] entry expired count:%d", count), player)
	}
}

func buildAdventureEntryInfos(entries []*model.AdventureEntry) []*pb.AdventureEntryInfo {
	res := make([]*pb.AdventureEntryInfo, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		res = append(res, &pb.AdventureEntryInfo{
			UniqueId:    entry.UniqueId,
			AdventureId: entry.AdventureId,
			ExpireTime:  entry.ExpireTime,
		})
	}
	return res
}

func getTriggerAdventureId(player *model.PlayerModel, m *model.PlayerAdventureModel, progressCfg *gameConfig.AdventureProgressCfg) int32 {
	if progressCfg.Adventure != 0 {
		if m.CanSettle(progressCfg.Adventure) && checkAdventureUnlock(player, progressCfg.Adventure) {
			return progressCfg.Adventure
		}
		return 0
	}
	totalWeight := int32(0)
	candidates := make([]*gameConfig.AdventureCfg, 0)
	for _, cfg := range gameConfig.GetAllAdventureCfg() {
		if cfg == nil || cfg.Weight <= 0 || !m.CanSettle(cfg.Id) || !checkAdventureUnlock(player, cfg.Id) {
			continue
		}
		totalWeight += cfg.Weight
		candidates = append(candidates, cfg)
	}
	if totalWeight <= 0 {
		return 0
	}
	randWeight := tool.RandInt32(1, totalWeight)
	for _, cfg := range candidates {
		if randWeight <= cfg.Weight {
			return cfg.Id
		}
		randWeight -= cfg.Weight
	}
	return 0
}

func checkAdventureUnlock(player *model.PlayerModel, adventureId int32) bool {
	cfg := gameConfig.GetAdventureCfg(adventureId)
	if cfg == nil {
		return false
	}
	if len(cfg.Unlock) == 0 {
		return true
	}
	for _, unlock := range cfg.Unlock {
		if unlock != 0 && !unlockService.CheckUnlock(unlock, player) {
			return false
		}
	}
	return true
}

func getCurrentMainStageId(player *model.PlayerModel) int32 {
	if player == nil || player.PlayerInstanceModel == nil {
		return 0
	}
	instance := player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instance != nil && instance.StageId > 0 {
		return instance.StageId
	}
	if player.PlayerInstanceModel.CurrentMainInstanceInfo != nil {
		return player.PlayerInstanceModel.CurrentMainInstanceInfo.CurrentStageId
	}
	return 0
}

func getAdventureDungeonId(adventureId int32, mainStageId int32) int32 {
	if mainStageId <= 0 {
		return 0
	}
	candidates := make([]int32, 0)
	for _, cfg := range gameConfig.GetAllDungeonAdventureCfg() {
		if cfg == nil || cfg.Type != adventureId || len(cfg.MainStage) != 2 {
			continue
		}
		if cfg.MainStage[0] <= mainStageId && mainStageId <= cfg.MainStage[1] {
			candidates = append(candidates, cfg.Id)
		}
	}
	if len(candidates) == 0 {
		return 0
	}
	return candidates[tool.RandInt32(0, int32(len(candidates)-1))]
}
