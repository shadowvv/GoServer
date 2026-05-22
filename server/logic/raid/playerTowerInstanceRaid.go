package raid

import (
	"errors"

	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/operationLogService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

type TowerInstanceRaid struct {
}

var _ RaidOperatorInterface = (*TowerInstanceRaid)(nil)

func (p *TowerInstanceRaid) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	config := gameConfig.GetTowerCfg(enterStageId)
	if config == nil {
		return false
	}
	if currentStageId == 0 && config.Level == 1 {
		return true
	}
	currentConfig := gameConfig.GetTowerCfg(currentStageId)
	if currentConfig == nil {
		return false
	}
	if currentConfig.Level+1 == config.Level {
		return true
	}
	return false
}

func (p *TowerInstanceRaid) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	OnLeaveRaidCommon(info)
}

func (p *TowerInstanceRaid) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	towerConfig := gameConfig.GetTowerCfg(raidInfo.CurrentStageId)
	if towerConfig == nil {
		return errors.New("tower config not exist")
	}
	raidInfo.BattleId = battleIdGen.NextId()
	raidInfo.RandomSeed = tool.UnixNowMilli()
	raidInfo.CurrentSubStageId = raidInfo.CurrentStageId

	subStageData := &logicCommon.SubStageData{
		SubStageId:  raidInfo.CurrentStageId,
		RoomID:      0,
		MonsterInfo: make(map[int32]*logicCommon.MonsterInfo),
		ComboSkills: make([]int32, 0),
	}
	raidInfo.SubStageInfo[subStageData.SubStageId] = subStageData
	raidInfo.SubStageIds = append(raidInfo.SubStageIds, subStageData.SubStageId)

	monsterFormation := &logicCommon.FormationBasicInfo{
		Heroes:      make([]int64, 0),
		BattlePower: 0,
	}
	monsterHeroDetails := make(map[int64]*logicCommon.HeroBasicInfo)
	for i, monsterId := range towerConfig.Lineup {
		monsterCfg := gameConfig.GetMonsterCfg(monsterId)
		if monsterCfg == nil {
			continue
		}
		monsterFormation.Heroes = append(monsterFormation.Heroes, int64(monsterId))
		monsterHeroDetails[int64(monsterId)] = &logicCommon.HeroBasicInfo{
			Id:   int64(monsterId),
			Star: monsterCfg.Star,
		}
		monsterInfo := &logicCommon.MonsterInfo{
			SpawnId:      "1",
			WaveId:       0,
			WaveSequence: 1,
			Id:           monsterCfg.Id*1000 + 1*100 + int32(i),
			MonsterId:    monsterId,
			DropItems:    make([]*gameConfig.ItemConfig, 0),
		}
		subStageData.MonsterInfo[monsterInfo.Id] = monsterInfo

		if _, ok := raidInfo.MonsterTemplates[int64(monsterCfg.Id)]; !ok {
			monsterTemplate := &logicCommon.MonsterTemplate{
				MonsterId:   int64(monsterCfg.Id),
				UnitId:      monsterCfg.Units,
				AtkSpeed:    monsterCfg.AtkSpeed,
				MoveSpeed:   monsterCfg.MoveSpeed,
				PatrolRange: monsterCfg.PatrolRange,
				AggroRange:  monsterCfg.AggroRange,
				AttackRange: monsterCfg.AttackRange,
				BasicSkill:  monsterCfg.NormalAtk,
				AttrInfo:    make(map[int32]int64),
				SkillId:     make([]int32, 0),
			}
			for id, attr := range monsterCfg.Attr {
				monsterTemplate.AttrInfo[id] = attr
			}
			for _, skill := range monsterCfg.Skill {
				monsterTemplate.SkillId = append(monsterTemplate.SkillId, skill)
			}
			raidInfo.MonsterTemplates[int64(monsterCfg.Id)] = monsterTemplate
		}
	}
	raidInfo.SubStageInfo[subStageData.SubStageId].ComboSkills = GetEnemyComberSkillIds(0, monsterFormation, monsterHeroDetails)
	return nil
}

func (p *TowerInstanceRaid) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	levelConfig := gameConfig.GetTowerCfg(stageId)
	if levelConfig == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	dropItems := gameConfig.Drop(levelConfig.SweepReward)
	return dropItems, pb.ERROR_CODE_SUCCESS
}

func (p *TowerInstanceRaid) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	levelConfig := gameConfig.GetTowerCfg(stageId)
	if levelConfig == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if levelConfig.StageReward == 0 {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	dropItems := gameConfig.Drop(levelConfig.StageReward)
	return dropItems, pb.ERROR_CODE_SUCCESS
}

func (p *TowerInstanceRaid) BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	return BattleOperationCommon(raidInfo, playerId, x, y)
}
func (p *TowerInstanceRaid) KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	monsterIdList, dropItems, errCode := KillMonsterCommon(raidInfo, monsterIds)
	if errCode == pb.ERROR_CODE_SUCCESS {
		eventBusService.SubmitKillMonsterEvent(player.GetUserId(), int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID), monsterIdList)
	}
	return monsterIdList, dropItems, errCode
}

func (p *TowerInstanceRaid) CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckRaidEndCommon(raidInfo)
}

func (p *TowerInstanceRaid) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	operationLogService.OnUserTowerChange(player.GetUserId(), 1, info.CurrentStageId)
	config := gameConfig.GetInstanceCfg(int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID))
	if config != nil {
		_ = itemService.RemoveItems(player, config.TicketID, enum.ITEM_CHANGE_REASON_CHALLENGE_INSTANCE)
	}
	err := playerInstanceModel.UpdateInstanceInfo(int32(info.InstanceID), info.CurrentStageId, info.CurrentSubStageId)
	if err != nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	cfg := gameConfig.GetTowerCfg(info.CurrentStageId)
	if cfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	dropItems := gameConfig.Drop(cfg.LevelReward)
	if len(dropItems) > 0 {
		_ = itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_TOWER_STAGE_REWARD)
	}
	passService.UpdateAllPassProgressBySystem(player, 208) // 208=爬塔关卡
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      1,
		InstanceId: int32(info.InstanceID),
		StageId:    playerInstanceModel.CurrentRaidInfo.CurrentStageId,
	}
}

func (p *TowerInstanceRaid) CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckCurrentSubStageOverCommon(raidInfo)
}

func (p *TowerInstanceRaid) CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	return pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (p *TowerInstanceRaid) EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, SubStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (p *TowerInstanceRaid) ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo {
	return CycleCurrentStageCommon(raidInfo, privilegeDropCount)
}

func (p *TowerInstanceRaid) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	operationLogService.OnUserTowerChange(player.GetUserId(), 0, info.CurrentStageId)
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      0,
		InstanceId: int32(info.InstanceID),
		StageId:    info.CurrentStageId,
	}
}
