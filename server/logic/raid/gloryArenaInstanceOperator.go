package raid

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/tool"
)

type GloryArenaInstanceRaid struct{}

var _ RaidOperatorInterface = (*GloryArenaInstanceRaid)(nil)

func (g *GloryArenaInstanceRaid) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	return true
}

func (g *GloryArenaInstanceRaid) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	OnLeaveRaidCommon(info)
}

func (g *GloryArenaInstanceRaid) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	return g.BuildRaidWithPlayer(raidInfo)
}

func (g *GloryArenaInstanceRaid) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SUCCESS
}

func (g *GloryArenaInstanceRaid) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SUCCESS
}

func (g *GloryArenaInstanceRaid) BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	return BattleOperationCommon(raidInfo, playerId, x, y)
}

func (g *GloryArenaInstanceRaid) KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return KillMonsterCommon(raidInfo, monsterIds)
}

func (g *GloryArenaInstanceRaid) CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	return pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (g *GloryArenaInstanceRaid) EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, SubStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (g *GloryArenaInstanceRaid) CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	lastStageId := raidInfo.SubStageIds[len(raidInfo.SubStageIds)-1]
	if raidInfo.CurrentSubStageId != lastStageId {
		return false
	}
	stageInfo := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if stageInfo == nil {
		return false
	}
	for _, monster := range stageInfo.MonsterInfo {
		if monster.IsDead != 1 {
			return false
		}
	}
	return true
}

func (g *GloryArenaInstanceRaid) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	if player == nil || player.PlayerGloryArenaModel == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	_, err := player.PlayerGloryArenaModel.TrySettleBattle(info.TargetTd, true)
	if err != nil {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM, nil
	}
	winCount := player.PlayerGloryArenaModel.GetWinCount()
	if player.PlayerGloryArenaModel.ShouldSaveSelectedOpponentSnapshot() {
		opponentInfo := logicCommon.GetPlayerRedisInfo(info.TargetTd)
		player.PlayerGloryArenaModel.SaveDefeatedOpponentSnapshotFromPlayerInfo(info.TargetTd, opponentInfo)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_HONOR_ARENA_WIN_MESSAGE, &pb.PushHonorArenaWinMessage{
		WinCount: winCount,
		Hero:     player.PlayerGloryArenaModel.BuildSelectableHeroes(),
	})
	eventBusService.SubmitPassInstanceEvent(player.GetUserId(), player.GetUserServerId(), enum.GLORY_ARENA_INSTANCE_ID, info.CurrentStageId)
	cfg := gameConfig.GetGloryArenaPerGameCfg(player.PlayerGloryArenaModel.GetRoundWinCount())
	if cfg != nil {
		items := gameConfig.Drop(cfg.Drop)
		if items != nil && len(items) > 0 {
			_ = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_GLORY_ARENA_WIN)
		}
	}
	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc != nil {
		opsState, _ := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
		if opsState != nil {
			matchReq := &gloryArenaService.GloryArenaMatchRequest{
				PlayerId:       player.GetUserId(),
				PoolVersion:    opsState.GroupVersion,
				WinCount:       player.PlayerGloryArenaModel.GetWinCount(),
				SelfPower:      player.GetMainFormationPower(),
				DefeatedSet:    player.PlayerGloryArenaModel.GetDefeatedSet(),
				LastOpponents:  player.PlayerGloryArenaModel.GetCurrentMatchCandidates(),
				NeedCount:      gloryArenaService.DefaultGloryArenaMatchCount,
				ForceDifferent: true,
			}
			members, poolVersion, matchErr := gloryArenaSvc.GetChallengeList(matchReq)
			if matchErr == nil {
				opponentIDs := make([]int64, 0, len(members))
				for _, member := range members {
					if member == nil || member.PlayerId <= 0 {
						continue
					}
					opponentIDs = append(opponentIDs, member.PlayerId)
				}
				opponentInfos := gloryArenaSvc.LoadChallengePlayerInfos(opponentIDs)
				player.PlayerGloryArenaModel.SetCurrentMatchCandidates(opponentIDs, opponentInfos)
				player.PlayerGloryArenaModel.SetPoolVersion(poolVersion)
			}
		}
	}
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:            1,
		InstanceId:       int32(info.InstanceID),
		StageId:          info.CurrentStageId,
		ArenaChangePoint: 0,
	}
}

func (g *GloryArenaInstanceRaid) CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckCurrentSubStageOverCommon(raidInfo)
}

func (g *GloryArenaInstanceRaid) ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo {
	currentStage := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if currentStage == nil {
		return nil
	}
	for _, monster := range currentStage.MonsterInfo {
		monster.IsDead = 0
	}

	info := &pb.SubStageInfo{
		SubStageId:   currentStage.SubStageId,
		RoomId:       currentStage.RoomID,
		MonsterInfos: make([]*pb.MonsterInfo, 0),
		IsCyCle:      raidInfo.StageInfo.IsCycle,
	}
	for _, monster := range currentStage.MonsterInfo {
		monsterPb := &pb.MonsterInfo{
			Index:        monster.Id,
			SpawnId:      monster.SpawnId,
			WaveSequence: monster.WaveSequence,
			MonsterId:    monster.MonsterId,
			Drops:        make([]*pb.ItemBasicInfo, 0),
		}
		info.MonsterInfos = append(info.MonsterInfos, monsterPb)
	}
	return info
}

func (g *GloryArenaInstanceRaid) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	if player == nil || player.PlayerGloryArenaModel == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	_, err := player.PlayerGloryArenaModel.TrySettleBattle(info.TargetTd, false)
	if err != nil {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM, nil
	}

	dropId := gameConfig.GetGloryArenaDefeatDrop()
	if dropId != 0 {
		items := gameConfig.Drop(dropId)
		if items != nil && len(items) > 0 {
			_ = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_GLORY_ARENA_LOSE)
		}
	}
	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc != nil {
		opsState, _ := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
		if opsState != nil {
			matchReq := &gloryArenaService.GloryArenaMatchRequest{
				PlayerId:       player.GetUserId(),
				PoolVersion:    opsState.GroupVersion,
				WinCount:       player.PlayerGloryArenaModel.GetWinCount(),
				SelfPower:      player.GetMainFormationPower(),
				DefeatedSet:    player.PlayerGloryArenaModel.GetDefeatedSet(),
				LastOpponents:  player.PlayerGloryArenaModel.GetCurrentMatchCandidates(),
				NeedCount:      gloryArenaService.DefaultGloryArenaMatchCount,
				ForceDifferent: true,
			}
			members, poolVersion, matchErr := gloryArenaSvc.GetChallengeList(matchReq)
			if matchErr == nil {
				opponentIDs := make([]int64, 0, len(members))
				for _, member := range members {
					if member == nil || member.PlayerId <= 0 {
						continue
					}
					opponentIDs = append(opponentIDs, member.PlayerId)
				}
				opponentInfos := gloryArenaSvc.LoadChallengePlayerInfos(opponentIDs)
				player.PlayerGloryArenaModel.SetCurrentMatchCandidates(opponentIDs, opponentInfos)
				player.PlayerGloryArenaModel.SetPoolVersion(poolVersion)
			}
		}
	}

	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:            0,
		InstanceId:       int32(info.InstanceID),
		StageId:          info.CurrentStageId,
		ArenaChangePoint: 0,
	}
}

func (g *GloryArenaInstanceRaid) BuildRaidWithPlayer(raidInfo *logicCommon.PlayerInstanceRaid) error {
	playerInfo := logicCommon.GetPlayerRedisInfo(raidInfo.TargetTd)
	if playerInfo == nil || playerInfo.BattleInfo == nil {
		return errors.New("player not exist")
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

	formation := playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA)]
	if formation == nil {
		formation = playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF)]
	}
	if formation != nil {
		subStageData.ComboSkills = GetEnemyComberSkillIds(raidInfo.TargetTd, formation, playerInfo.BattleInfo.FormationHeroes)
	}

	heroIDs := make([]int64, 0)
	if formation != nil {
		heroIDs = append(heroIDs, formation.Heroes...)
	}
	if len(heroIDs) == 0 {
		for heroID := range playerInfo.BattleInfo.FormationHeroes {
			heroIDs = append(heroIDs, heroID)
		}
	}

	for _, heroID := range heroIDs {
		heroInfo := playerInfo.BattleInfo.FormationHeroes[heroID]
		if heroInfo == nil {
			continue
		}
		monsterInfo := &logicCommon.MonsterInfo{
			SpawnId:      "1",
			WaveId:       0,
			WaveSequence: 1,
			Id:           int32(heroInfo.Id),
			MonsterId:    int32(heroInfo.Id),
			DropItems:    make([]*gameConfig.ItemConfig, 0),
		}
		subStageData.MonsterInfo[monsterInfo.Id] = monsterInfo

		if _, ok := raidInfo.MonsterTemplates[heroInfo.Id]; ok {
			continue
		}
		cfg := gameConfig.GetHeroBaseCfg(int32(heroInfo.Id))
		if cfg == nil || heroInfo.Units == 0 {
			continue
		}
		monsterTemplate := &logicCommon.MonsterTemplate{
			MonsterId:   heroInfo.Id,
			UnitId:      heroInfo.Units,
			AtkSpeed:    int32(heroInfo.Attr[enum.AttributeBasicAttackSpeed]),
			MoveSpeed:   int32(heroInfo.Attr[enum.AttributeBasicMoveSpeed]),
			PatrolRange: cfg.PatrolRange,
			AggroRange:  cfg.AggroRange,
			AttackRange: cfg.AttackRange,
			BasicSkill:  heroInfo.NormalAtk,
			AttrInfo:    make(map[int32]int64),
			SkillId:     make([]int32, 0),
		}
		if heroInfo.PetInfo != nil {
			monsterTemplate.PetInfo = &pb.PetBattleInfo{
				PetId:     heroInfo.PetInfo.PetId,
				Level:     heroInfo.PetInfo.Level,
				Star:      heroInfo.PetInfo.Star,
				SkillList: make([]int32, 0),
			}
			copy(monsterTemplate.PetInfo.SkillList, heroInfo.PetInfo.SkillList)
		}
		for id, attr := range heroInfo.Attr {
			monsterTemplate.AttrInfo[id] = attr
		}
		for _, skill := range heroInfo.Skill {
			monsterTemplate.SkillId = append(monsterTemplate.SkillId, skill)
		}
		raidInfo.MonsterTemplates[heroInfo.Id] = monsterTemplate
	}
	if len(subStageData.MonsterInfo) == 0 {
		return errors.New("player formation empty")
	}
	return nil
}
