package raid

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

type ResidentInstanceRaid struct {
	AdventureRaidOperation
}

const (
	residentBattleWin  = 1
	residentBattleLose = 0
)

var _ RaidOperatorInterface = (*ResidentInstanceRaid)(nil)

func (r *ResidentInstanceRaid) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	enterCfg := gameConfig.GetDungeonAdventureCfg(enterStageId)
	if enterCfg == nil {
		return false
	}
	instanceType, ok := enum.GetResidentInstanceTypeByDungeonType(enterCfg.Type)
	if !ok {
		return false
	}
	return canEnterResidentStage(instanceType, enterStageId, currentStageId, 0)
}

func CanEnterResidentInstanceStage(instanceType int32, enterStageId int32, currentStageId int32, commitLevelReward int32) bool {
	return canEnterResidentStage(instanceType, enterStageId, currentStageId, commitLevelReward)
}

func (r *ResidentInstanceRaid) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	OnLeaveRaidCommon(info)
}

func (r *ResidentInstanceRaid) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	dungeonCfg := gameConfig.GetDungeonAdventureCfg(stageId)
	if dungeonCfg == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	items := make([]*gameConfig.ItemConfig, 0)
	for _, lineup := range dungeonCfg.Lineup {
		for _, waveId := range lineup {
			waveCfg := gameConfig.GetDungeonMonsterWaveCfg(waveId)
			if waveCfg == nil {
				return nil, pb.ERROR_CODE_SYSTEM_ERROR
			}
			if waveCfg.DropGroup == 0 {
				continue
			}
			for i := int32(0); i < waveCfg.MonsterNum; i++ {
				items = append(items, gameConfig.DropGroupItems(waveCfg.DropGroup, nil)...)
			}
		}
	}
	return mergeItemConfigs(items), pb.ERROR_CODE_SUCCESS
}

func (r *ResidentInstanceRaid) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (r *ResidentInstanceRaid) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return errors.New("resident instance config not exist")
	}
	dungeonCfg := gameConfig.GetDungeonAdventureCfg(raidInfo.CurrentStageId)
	if dungeonCfg == nil {
		return errors.New("resident dungeon config not exist")
	}
	dungeonType, ok := enum.GetResidentDungeonType(instanceCfg.InstanceType)
	if !ok {
		return errors.New("resident dungeon type invalid")
	}
	if dungeonCfg.Type != dungeonType {
		return errors.New("resident dungeon type mismatch")
	}

	raidInfo.BattleId = battleIdGen.NextId()
	raidInfo.RandomSeed = tool.UnixNowMilli()
	raidInfo.CurrentSubStageId = raidInfo.CurrentStageId
	raidInfo.FormationType = int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ADVENTURE)

	subStageData := buildDungeonRaidSubStage(raidInfo, dungeonCfg)
	raidInfo.SubStageInfo[subStageData.SubStageId] = subStageData
	raidInfo.SubStageIds = append(raidInfo.SubStageIds, subStageData.SubStageId)
	return nil
}

func (r *ResidentInstanceRaid) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	return r.finishResidentInstance(player, info, playerInstanceModel, residentBattleWin)
}

func (r *ResidentInstanceRaid) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	return r.finishResidentInstance(player, info, instanceModel, residentBattleLose)
}

func (r *ResidentInstanceRaid) finishResidentInstance(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel, isWin int32) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	config := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	_ = itemService.RemoveItems(player, config.TicketID, enum.ITEM_CHANGE_REASON_CHALLENGE_INSTANCE)
	stageId := info.CurrentStageId
	commitLevelReward := int32(0)
	levelRewardItems := make([]*gameConfig.ItemConfig, 0)
	if isWin == residentBattleWin {
		dungeonCfg := gameConfig.GetDungeonAdventureCfg(info.CurrentStageId)
		if dungeonCfg == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		entity := playerInstanceModel.InstanceEntities[int32(info.InstanceID)]
		commitLevelReward = info.CurrentStageId
		if dungeonCfg.LevelReward != 0 && (entity == nil || entity.CommitLevelReward < info.CurrentStageId) {
			levelRewardItems = gameConfig.Drop(dungeonCfg.LevelReward)
		}
		nextCfg := gameConfig.GetDungeonAdventureCfgByTypeAndLevel(dungeonCfg.Type, dungeonCfg.Level+1)
		if nextCfg != nil {
			stageId = nextCfg.Id
		}
		if err := playerInstanceModel.UpdateInstanceInfo(int32(info.InstanceID), info.CurrentStageId, info.CurrentStageId); err != nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
	}
	if err := playerInstanceModel.UpdateInstanceInfo(int32(info.InstanceID), stageId, stageId); err != nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	addDungeonSettleDropItems(player, info, levelRewardItems...)
	if commitLevelReward > 0 {
		entity := playerInstanceModel.InstanceEntities[int32(info.InstanceID)]
		if entity != nil && entity.CommitLevelReward < commitLevelReward {
			if playerInstanceModel.Changed[int32(info.InstanceID)] == nil {
				playerInstanceModel.Changed[int32(info.InstanceID)] = make(map[string]interface{})
			}
			entity.CommitLevelReward = commitLevelReward
			playerInstanceModel.Changed[int32(info.InstanceID)]["commit_level_reward"] = commitLevelReward
		}
	}
	if isWin == residentBattleWin {
		switch info.InstanceID {
		case enum.COIN_INSTANCE_ID:
			passService.UpdateAllPassProgressBySystem(player, 204) // 204=金币副本
		case enum.CAPSULE_INSTANCE_ID:
			passService.UpdateAllPassProgressBySystem(player, 205) // 205=胶囊副本
		case enum.HERO_INSTANCE_ID:
			passService.UpdateAllPassProgressBySystem(player, 206) // 206=英雄副本
		case enum.PET_INSTANCE_ID:
			passService.UpdateAllPassProgressBySystem(player, 207) // 207=宠物副本
		}
	}
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      isWin,
		InstanceId: int32(info.InstanceID),
		StageId:    info.CurrentStageId,
	}
}

func canEnterResidentStage(instanceType int32, enterStageId int32, currentStageId int32, commitLevelReward int32) bool {
	if !enum.IsResidentInstanceType(instanceType) {
		return false
	}
	enterCfg := gameConfig.GetDungeonAdventureCfg(enterStageId)
	if enterCfg == nil {
		return false
	}
	dungeonType, ok := enum.GetResidentDungeonType(instanceType)
	if !ok || enterCfg.Type != dungeonType {
		return false
	}
	if currentStageId == 0 {
		return enterCfg.Level == 1
	}
	if commitLevelReward > 0 {
		commitCfg := gameConfig.GetDungeonAdventureCfg(commitLevelReward)
		if commitCfg == nil || commitCfg.Type != dungeonType {
			return false
		}
		if enterCfg.Level <= commitCfg.Level {
			return false
		}
	}
	currentCfg := gameConfig.GetDungeonAdventureCfg(currentStageId)
	if currentCfg == nil || currentCfg.Type != dungeonType {
		return false
	}
	return enterCfg.Id == currentCfg.Id
}

func canSweepResidentStage(commitLevelReward int32, sweepStageId int32) bool {
	if commitLevelReward <= 0 || sweepStageId <= 0 {
		return false
	}
	currentCfg := gameConfig.GetDungeonAdventureCfg(commitLevelReward)
	sweepCfg := gameConfig.GetDungeonAdventureCfg(sweepStageId)
	if currentCfg == nil || sweepCfg == nil {
		return false
	}
	if sweepCfg.Type != currentCfg.Type {
		return false
	}
	return sweepCfg.Level <= currentCfg.Level
}
