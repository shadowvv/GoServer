package raid

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type ArenaInstanceRaid struct {
}

var _ RaidOperatorInterface = (*ArenaInstanceRaid)(nil)

func (a *ArenaInstanceRaid) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	return true
}

func (a *ArenaInstanceRaid) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	for _, stageInfo := range info.SubStageInfo {
		for _, monster := range stageInfo.MonsterInfo {
			monster.IsDead = 0
		}
	}
}

func (a *ArenaInstanceRaid) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	if raidInfo.IsRobot {
		return a.BuildRaidWithBot(raidInfo)
	} else {
		return a.BuildRaidWithPlayer(raidInfo)
	}
}

func (a *ArenaInstanceRaid) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SUCCESS
}

func (a *ArenaInstanceRaid) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SUCCESS
}

func (a *ArenaInstanceRaid) BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	return BattleOperationCommon(raidInfo, playerId, x, y)
}

func (a *ArenaInstanceRaid) KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return KillMonsterCommon(raidInfo, monsterIds)
}

func (a *ArenaInstanceRaid) CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	return pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (a *ArenaInstanceRaid) EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, SubStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (a *ArenaInstanceRaid) CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool {
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

func (a *ArenaInstanceRaid) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	opponent := player.PlayerArenaModel.GetChallengeOpponent(info.TargetTd)
	if opponent == nil {
		areanLogs := player.PlayerArenaModel.GetAllArenaLog()
		find := false
		for _, log := range areanLogs {
			if log.AttackUserId == info.TargetTd {
				find = true
				break
			}
		}
		if !find {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		opponentInfo := logicCommon.GetPlayerRedisInfo(info.TargetTd)
		if opponentInfo == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		opponent = &model.PlayerChallengeBasicData{
			IsRobot: 0,
			Score:   opponentInfo.BasicInfo.ArenaScore,
			UserId:  info.TargetTd,
		}
	}
	if opponent == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}

	playerPointCfg := gameConfig.GetPointsParametersCfgByScore(player.PlayerArenaModel.GetScore())
	if playerPointCfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	pointChange := int32(0)
	addPoint := playerPointCfg.WinBase + playerPointCfg.Coeff1*(opponent.Score-player.PlayerArenaModel.GetScore())/10000
	if addPoint <= 0 {
		addPoint = 1
	}
	pointChange = addPoint
	player.PlayerArenaModel.AddScore(pointChange)
	operationLogService.OnUserArenaChange(player.GetUserId(), operationLogService.ARENA_OPER_CHALLENGE, 1, player.PlayerArenaModel.GetScore()-pointChange, pointChange)
	if opponent.IsRobot != 1 {
		defendChangeScore := a.ChangeOpponentScore(opponent, player, false)
		_ = easyDB.CreatePlayerEntity[model.PlayerArenaLogEntity](&model.PlayerArenaLogEntity{
			BattleId:          info.BattleId,
			AttackUserId:      player.GetUserId(),
			AttackScoreChange: pointChange,
			DefendUserId:      opponent.UserId,
			DefendScoreChange: defendChangeScore,
			DefendResolved:    0,
			ChallengeTime:     tool.UnixNowMilli(),
			Version:           player.PlayerArenaModel.GetVersion(),
		})
	}

	player.PlayerArenaModel.RefreshChallengeList()
	eventBusService.SubmitArenaScoreChangeEvent(player.GetUserId(), player.PlayerArenaModel.GetScore())

	items := gameConfig.Drop(gameConfig.GetArenaVictoryReward())
	_ = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_ARENA_WIN)
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:            1,
		InstanceId:       int32(info.InstanceID),
		StageId:          info.CurrentStageId,
		ArenaChangePoint: pointChange,
	}
}

func (a *ArenaInstanceRaid) ChangeOpponentScore(player *model.PlayerChallengeBasicData, opponent *model.PlayerModel, win bool) int32 {
	if player.IsRobot == 1 {
		return 0
	}
	opponentPointCfg := gameConfig.GetPointsParametersCfgByScore(player.Score)
	if opponentPointCfg == nil {
		return 0
	}
	changePoint := int32(0)
	if win {
		changePoint += opponentPointCfg.WinBase + opponentPointCfg.Coeff1*(opponent.PlayerArenaModel.GetScore()-player.Score)/10000
		operationLogService.OnUserArenaChange(player.UserId, operationLogService.ARENA_OPER_BE_CHALLENGED, 1, 0, changePoint)
	} else {
		changePoint -= opponentPointCfg.LoseBase - opponentPointCfg.Coeff2*(opponent.PlayerArenaModel.GetScore()-player.Score)/10000
		operationLogService.OnUserArenaChange(player.UserId, operationLogService.ARENA_OPER_BE_CHALLENGED, 0, 0, changePoint)
	}
	version := opponent.PlayerArenaModel.GetVersion()
	if version == "" {
		version = logicCommon.GetArenaRankVersionByTime(opponent.GetUserServerId(), tool.UnixNowMilli())
	}
	rankId, err := logicCommon.GetRankUniqueId(gameConfig.GetArenaRankId(), 0, 0, opponent.GetUserServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		return 0
	}
	updateRankReq := &rpcPb.NotifyUpdateRankInfo{
		PlayerId:          player.UserId,
		Score:             int64(changePoint),
		IncrementalUpdate: true,
	}
	_ = rpcController.SendMessageToRankBoard(player.UserId, rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, updateRankReq)

	allianceRankIds := logicCommon.GetCommonRankUniqueIdsByPointType(
		opponent.GetUserServerId(),
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA,
		opponent.PlayerArenaModel.GetVersion(),
	)
	for _, allianceRankId := range allianceRankIds {
		_ = rpcController.SendMessageToRankBoard(player.UserId, allianceRankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, updateRankReq)
	}

	logicCommon.UpdateAreanScoreRank(opponent.GetUserServerId(), version, player.UserId, player.Score+changePoint)
	return changePoint
}

func (a *ArenaInstanceRaid) CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckCurrentSubStageOverCommon(raidInfo)
}

func (a *ArenaInstanceRaid) ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo {
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

func (a *ArenaInstanceRaid) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	opponent := player.PlayerArenaModel.GetChallengeOpponent(info.TargetTd)
	if opponent == nil {
		areanLogs := player.PlayerArenaModel.GetAllArenaLog()
		find := false
		for _, log := range areanLogs {
			if log.AttackUserId == info.TargetTd {
				find = true
				break
			}
		}
		if !find {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		opponentInfo := logicCommon.GetPlayerRedisInfo(info.TargetTd)
		if opponentInfo == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		opponent = &model.PlayerChallengeBasicData{
			IsRobot: 0,
			Score:   opponentInfo.BasicInfo.ArenaScore,
			UserId:  info.TargetTd,
		}
	}
	if opponent == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	playerPointCfg := gameConfig.GetPointsParametersCfgByScore(player.PlayerArenaModel.GetScore())
	if playerPointCfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	pointChange := int32(0)
	removePoint := playerPointCfg.LoseBase - playerPointCfg.Coeff2*(opponent.Score-player.PlayerArenaModel.GetScore())/10000
	if removePoint <= 0 {
		removePoint = 1
	}
	pointChange = -removePoint
	player.PlayerArenaModel.AddScore(pointChange)
	operationLogService.OnUserArenaChange(player.GetUserId(), operationLogService.ARENA_OPER_CHALLENGE, 0, player.PlayerArenaModel.GetScore()-pointChange, pointChange)
	if opponent.IsRobot != 1 {
		defendChangeScore := a.ChangeOpponentScore(opponent, player, true)
		_ = easyDB.CreatePlayerEntity[model.PlayerArenaLogEntity](&model.PlayerArenaLogEntity{
			BattleId:          info.BattleId,
			AttackUserId:      player.GetUserId(),
			AttackScoreChange: pointChange,
			DefendUserId:      opponent.UserId,
			DefendScoreChange: defendChangeScore,
			DefendResolved:    0,
			ChallengeTime:     tool.UnixNowMilli(),
			Version:           player.PlayerArenaModel.GetVersion(),
		})
	}

	player.PlayerArenaModel.RefreshChallengeList()
	eventBusService.SubmitArenaScoreChangeEvent(player.GetUserId(), player.PlayerArenaModel.GetScore())

	items := gameConfig.Drop(gameConfig.GetArenaDefeatReward())
	_ = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_ARENA_LOSE)
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:            0,
		InstanceId:       int32(info.InstanceID),
		StageId:          info.CurrentStageId,
		ArenaChangePoint: pointChange,
	}
}

func (a *ArenaInstanceRaid) BuildRaidWithBot(raidInfo *logicCommon.PlayerInstanceRaid) error {
	arenaConfig := gameConfig.GetBotCfg(int32(raidInfo.TargetTd))
	if arenaConfig == nil {
		return errors.New("bot config not exist")
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

	for i, monsterId := range arenaConfig.ArenaLineup {
		monsterCfg := gameConfig.GetMonsterCfg(monsterId)
		if monsterCfg == nil {
			continue
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
	return nil
}

func (a *ArenaInstanceRaid) BuildRaidWithPlayer(raidInfo *logicCommon.PlayerInstanceRaid) error {
	playerInfo := logicCommon.GetPlayerRedisInfo(raidInfo.TargetTd)
	if playerInfo == nil {
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
	subStageData.ComboSkills = GetEnemyComberSkillIds(raidInfo.TargetTd, playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF)], playerInfo.BattleInfo.FormationHeroes)
	raidInfo.SubStageInfo[subStageData.SubStageId] = subStageData
	raidInfo.SubStageIds = append(raidInfo.SubStageIds, subStageData.SubStageId)

	for _, heroInfo := range playerInfo.BattleInfo.FormationHeroes {
		monsterInfo := &logicCommon.MonsterInfo{
			SpawnId:      "1",
			WaveId:       0,
			WaveSequence: 1,
			Id:           int32(heroInfo.Id),
			MonsterId:    int32(heroInfo.Id),
			DropItems:    make([]*gameConfig.ItemConfig, 0),
		}
		subStageData.MonsterInfo[monsterInfo.Id] = monsterInfo

		// TODO: 获取英雄攻击距离和普通技能
		if _, ok := raidInfo.MonsterTemplates[heroInfo.Id]; !ok {
			cfg := gameConfig.GetHeroBaseCfg(int32(heroInfo.Id))
			if cfg == nil {
				continue
			}
			if heroInfo.Units == 0 {
				logger.ErrorBySprintf("arena hero units is 0,playerId:%d,heroId:%d", playerInfo.BasicInfo.Id, heroInfo.Id)
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
				AttrInfo:    heroInfo.Attr,
				SkillId:     heroInfo.Skill,
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
	}
	return nil
}
