package raid

import (
	"errors"
	"fmt"

	"github.com/drop/GoServer/server/logic/platform/eventService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/logicScene"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

var unlockService logicCommon.UnlockServiceInterface
var messageSender logicCommon.MessageSenderInterface
var sceneService *logicScene.SceneService
var battleIdGen *tool.IdGenerator
var passService logicCommon.PassServiceInterface
var mailService logicCommon.MailServiceInterface
var eventBusService *eventService.EventBus

func InitRaid(event *eventService.EventBus, unlock logicCommon.UnlockServiceInterface, sender logicCommon.MessageSenderInterface, scene *logicScene.SceneService, pass logicCommon.PassServiceInterface, mail logicCommon.MailServiceInterface, nodeId int32) {
	eventBusService = event
	unlockService = unlock
	messageSender = sender
	sceneService = scene
	passService = pass
	battleIdGen = tool.NewIdGenerator(int64(nodeId), int64(enum.ID_GENERATOR_BATTLE_ID))
	mailService = mail
	model.InitPlayerHeartbeatService(&RaidHeartbeatService{})
}

var raidOperationMap = map[int32]RaidOperatorInterface{
	int32(enum.InstanceType_MAIN):        &MainInstanceOperation{},
	int32(enum.InstanceType_TOWER):       &TowerInstanceRaid{},
	int32(enum.InstanceType_ARENA):       &ArenaInstanceRaid{},
	int32(enum.InstanceType_ENCOUNTER):   &AdventureInstanceRaid{},
	int32(enum.InstanceType_COIN):        &ResidentInstanceRaid{},
	int32(enum.InstanceType_CAPSULE):     &ResidentInstanceRaid{},
	int32(enum.InstanceType_HERO):        &ResidentInstanceRaid{},
	int32(enum.InstanceType_PET):         &ResidentInstanceRaid{},
	int32(enum.InstanceType_GLORY_ARENA): &GloryArenaInstanceRaid{},
}

type RaidHeartbeatService struct{}

var _ logicCommon.PlayerHeartbeatServiceInterface = (*RaidHeartbeatService)(nil)

func (r *RaidHeartbeatService) Heartbeat(player logicCommon.PlayerInterface, currentTime int64) {
	p, ok := player.(*model.PlayerModel)
	if !ok {
		return
	}
	timeout, errorCode, winResp := CheckBattleTimeout(p, currentTime)
	if !timeout {
		return
	}
	if errorCode != pb.ERROR_CODE_SUCCESS {
		platformLogger.ErrorWithUser("battle timeout settle failed", p, nil)
		return
	}
	if winResp != nil && messageSender != nil {
		messageSender.SendMessage(p, pb.MESSAGE_ID_PUSH_STAGE_BATTLE_WIN, winResp)
	}
}

func getRaidOperator(instanceCfg *gameConfig.InstanceCfg) RaidOperatorInterface {
	if instanceCfg == nil {
		return nil
	}
	return raidOperationMap[instanceCfg.InstanceType]
}

func CanEnterInstanceStage(instanceId int32, stageId int32, currentStageId int32) bool {
	instanceCfg := gameConfig.GetInstanceCfg(instanceId)
	if instanceCfg == nil {
		return false
	}
	operator := raidOperationMap[instanceCfg.InstanceType]
	if operator == nil {
		return false
	}
	return operator.CanEnterInstanceStage(stageId, currentStageId)
}

func GetQuickBattleStage(instanceType int32, currentStageId int32, commitLevelReward int32, targetStageId int32) (int32, bool) {
	if enum.IsResidentInstanceType(instanceType) {
		if !canSweepResidentStage(commitLevelReward, targetStageId) {
			return 0, false
		}
		return targetStageId, true
	}
	return currentStageId, true
}

func DelayDropUntilSettle(instanceId int32) bool {
	instanceCfg := gameConfig.GetInstanceCfg(instanceId)
	if instanceCfg == nil {
		return false
	}
	return instanceId == int32(enum.ADVENTURE_INSTANCE_ID) || enum.IsResidentInstanceType(instanceCfg.InstanceType)
}

func OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return
	}
	operator.OnLeaveRaid(info)
}

func BuildInstanceRaid(info *logicCommon.PlayerInstanceRaid) error {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return errors.New("invalid instance id")
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return errors.New("invalid instance id")
	}
	return operator.BuildInstanceRaid(info)
}

func GetWeepReward(instanceId, stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	instanceCfg := gameConfig.GetInstanceCfg(instanceId)
	if instanceCfg == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	operator := raidOperationMap[instanceCfg.InstanceType]
	if operator == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	return operator.GetWeepReward(stageId)
}

func GetInstanceCommitReward(instanceId, stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	instanceCfg := gameConfig.GetInstanceCfg(instanceId)
	if instanceCfg == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	operator := raidOperationMap[instanceCfg.InstanceType]
	if operator == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	return operator.GetInstanceCommitReward(stageId)
}

func BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return errors.New("invalid instance id")
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return errors.New("invalid instance id")
	}
	return operator.BattleOperation(raidInfo, playerId, x, y)
}

func KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return nil, nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return nil, nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	return operator.KillMonster(player, raidInfo, monsterIds)
}

func CheckRaidEnd(info *logicCommon.PlayerInstanceRaid) bool {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return false
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return false
	}
	return operator.CheckRaidEnd(info)
}

func OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	return operator.OnRaidEnd(player, info, instanceModel)
}

func CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return false
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return false
	}
	return operator.CheckCurrentSubStageOver(raidInfo)
}

func ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount int32, player *model.PlayerModel) *pb.SubStageInfo {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return nil
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return nil
	}
	return operator.ResetCurrentStage(raidInfo, &privilegeDropCount, player)
}

func CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	return operator.CheckEnterNextSubStage(info, subStageId, player)
}

func EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, SubStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	instanceCfg := gameConfig.GetInstanceCfg(int32(raidInfo.InstanceID))
	if instanceCfg == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	return operator.EnterNextSubStage(raidInfo, SubStageId, player)
}

func OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	instanceCfg := gameConfig.GetInstanceCfg(int32(info.InstanceID))
	if instanceCfg == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	operator := getRaidOperator(instanceCfg)
	if operator == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	return operator.OnBattlePlayerDead(player, info, instanceModel)
}

func CheckBattleTimeout(player *model.PlayerModel, now int64) (bool, pb.ERROR_CODE, *pb.PushStageBattleWin) {
	if player == nil || player.PlayerInstanceModel == nil || player.PlayerInstanceModel.CurrentRaidInfo == nil {
		return false, pb.ERROR_CODE_SUCCESS, nil
	}
	raidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	if raidInfo.IsOver || raidInfo.BattleEndTime <= 0 || now < raidInfo.BattleEndTime {
		return false, pb.ERROR_CODE_SUCCESS, nil
	}
	raidInfo.IsOver = true
	errorCode, winResp := OnBattlePlayerDead(player, raidInfo, player.PlayerInstanceModel)
	return true, errorCode, winResp
}

func BuildAllMainInstanceData(userId int64, currentStageId, CurrentSubStageId, maxStageId, maxSubStageId int32, currentInstanceInfo *logicCommon.InstanceStageInfo, privilegeDropCount int32) (current, next *logicCommon.PlayerInstanceRaid, err error) {
	mainStageCfg := gameConfig.GetMainStageCfg(currentStageId)
	if mainStageCfg == nil {
		return nil, nil, errors.New(fmt.Sprintf("no main stage config:%d", currentStageId))
	}
	current = buildInstanceData(userId, enum.MAIN_INSTANCE_ID, currentStageId, CurrentSubStageId, currentInstanceInfo, maxStageId, maxSubStageId, privilegeDropCount)

	nextCfg := gameConfig.GetMainStageCfg(mainStageCfg.BackStage)
	if nextCfg != nil {
		next = buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.BackStage, nextCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount)
	} else {
		next = buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.Id, mainStageCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount)
	}
	return current, next, nil
}

func BuildPreMainInstanceData(userId int64, currentStageId, maxStageId, maxSubStageId int32, privilegeDropCount int32) (*logicCommon.PlayerInstanceRaid, error) {
	mainStageCfg := gameConfig.GetMainStageCfg(currentStageId)
	if mainStageCfg == nil {
		return nil, errors.New(fmt.Sprintf("no main stage config:%d", currentStageId))
	}
	preCfg := gameConfig.GetMainStageCfg(mainStageCfg.PreStage)
	if preCfg != nil {
		return buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.PreStage, preCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount), nil
	} else {
		logger.ErrorBySprintf("no main stage config:%d", mainStageCfg.PreStage)
		return buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.Id, mainStageCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount), nil
	}
}

func BuildNextMainInstanceData(userId int64, currentStageId, maxStageId, maxSubStageId int32, privilegeDropCount int32) (*logicCommon.PlayerInstanceRaid, error) {
	mainStageCfg := gameConfig.GetMainStageCfg(currentStageId)
	if mainStageCfg == nil {
		return nil, errors.New(fmt.Sprintf("no main stage config:%d", currentStageId))
	}
	preCfg := gameConfig.GetMainStageCfg(mainStageCfg.BackStage)
	if preCfg != nil {
		return buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.BackStage, preCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount), nil
	} else {
		logger.ErrorBySprintf("no main stage config:%d", mainStageCfg.BackStage)
		return buildInstanceData(userId, enum.MAIN_INSTANCE_ID, mainStageCfg.Id, mainStageCfg.SubStageId[0], logicCommon.NewInstanceStageInfo(), maxStageId, maxSubStageId, privilegeDropCount), nil
	}
}

func buildInstanceData(userId int64, instanceId enum.InstanceId, currentStageId, currentSubStageId int32, stageInfo *logicCommon.InstanceStageInfo, maxStageId, maxSubStageId, privilegeDropCount int32) *logicCommon.PlayerInstanceRaid {

	mainStageCfg := gameConfig.GetMainStageCfg(currentStageId)
	if mainStageCfg == nil {
		return nil
	}
	instanceData := &logicCommon.PlayerInstanceRaid{
		PlayerId:          userId,
		BattleId:          battleIdGen.NextId(),
		InstanceID:        instanceId,
		CurrentStageId:    currentStageId,
		CurrentSubStageId: currentSubStageId,
		RandomSeed:        tool.UnixNowMilli(),
		SubStageInfo:      make(map[int32]*logicCommon.SubStageData),
		SubStageIds:       make([]int32, 0),
		MonsterTemplates:  make(map[int64]*logicCommon.MonsterTemplate),
		StageInfo:         stageInfo,
	}
	for _, subStageId := range mainStageCfg.SubStageId {
		subStageCfg := gameConfig.GetSubStageCfg(subStageId)
		if subStageCfg == nil {
			continue
		}
		repeat := false
		if maxStageId >= currentStageId || maxSubStageId >= subStageCfg.Id {
			repeat = true
		}
		subStageInfo := buildSubStageData(subStageCfg, instanceData.MonsterTemplates, stageInfo, repeat, &privilegeDropCount)
		if subStageInfo == nil {
			continue
		}
		instanceData.SubStageInfo[subStageId] = subStageInfo
		instanceData.SubStageIds = append(instanceData.SubStageIds, subStageId)
	}
	return instanceData
}

func buildSubStageData(subStageCfg *gameConfig.SubStageCfg, monsterTemplates map[int64]*logicCommon.MonsterTemplate, stageInfo *logicCommon.InstanceStageInfo, repeat bool, privilegeDropCount *int32) *logicCommon.SubStageData {
	if len(subStageCfg.RoomId) <= 0 {
		return nil
	}
	index := tool.RandInt(1, len(subStageCfg.RoomId)) - 1
	if index < 0 {
		index = 0
	}
	roomId := subStageCfg.RoomId[index]
	subStageData := &logicCommon.SubStageData{
		SubStageId:  subStageCfg.Id,
		RoomID:      roomId,
		MonsterInfo: make(map[int32]*logicCommon.MonsterInfo),
	}

	for i, monsterWaveIdList := range subStageCfg.MonsterWaveId {
		for j, monsterWaveId := range monsterWaveIdList {
			buildMonsterWave(subStageCfg.MonsterSpawn[i][j], monsterWaveId, int32(i+1), subStageData.MonsterInfo, monsterTemplates, stageInfo, repeat, privilegeDropCount)
		}
	}

	for i, barrelWaveId := range subStageCfg.BarrelWaveId {
		buildMonsterWave(subStageCfg.BarrelSpawn[i], barrelWaveId, 1, subStageData.MonsterInfo, monsterTemplates, stageInfo, repeat, privilegeDropCount)
	}
	return subStageData
}

func buildMonsterWave(spawnId string, waveId int32, sequence int32, monsters map[int32]*logicCommon.MonsterInfo, monsterTemplates map[int64]*logicCommon.MonsterTemplate, stageInfo *logicCommon.InstanceStageInfo, repeat bool, privilegeDropCount *int32) {
	waveCfg := gameConfig.GetMonsterWaveCfg(waveId)
	if waveCfg == nil {
		return
	}
	monsterCfg := gameConfig.GetMonsterCfg(waveCfg.MonsterId)
	if monsterCfg == nil {
		return
	}
	for i := 0; i < int(waveCfg.MonsterNum); i++ {
		monsterInfo := &logicCommon.MonsterInfo{
			SpawnId:      spawnId,
			WaveId:       waveCfg.Id,
			WaveSequence: sequence,
			Id:           monsterCfg.Id*1000 + sequence*100 + int32(i),
			MonsterId:    waveCfg.MonsterId,
		}
		if _, ok := stageInfo.KillMonsterMap[monsterInfo.Id]; ok || repeat {
			if monsterInfo.WaveId == 20506 {
				logger.ErrorBySprintf("DEBUG stage2 boss eachDrop,playerId:%d", 0)
			}
			monsterInfo.DropItems = gameConfig.Drop(waveCfg.EachDrop)
		} else {
			monsterInfo.DropItems = gameConfig.Drop(waveCfg.FirstDrop)
		}
		if waveCfg.PrivilegeDrop > 0 && *privilegeDropCount > 0 {
			monsterInfo.DropItems = append(monsterInfo.DropItems, gameConfig.Drop(waveCfg.PrivilegeDrop)...)
			*privilegeDropCount--
		}

		monsters[monsterInfo.Id] = monsterInfo
	}
	if _, ok := monsterTemplates[int64(monsterCfg.Id)]; !ok {
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
		monsterTemplates[int64(monsterCfg.Id)] = monsterTemplate
	}
}

func BuildRaidPB(playerModel *model.PlayerModel, raidData *logicCommon.PlayerInstanceRaid) *pb.SceneBasicInfo {
	if raidData == nil {
		platformLogger.ErrorWithUser("raidData is nil", playerModel, nil)
		return nil
	}
	sceneBasicInfo := &pb.SceneBasicInfo{
		InstanceId:         int32(playerModel.PlayerInstanceModel.CurrentRaidInfo.InstanceID),
		StageId:            raidData.CurrentStageId,
		CurrentSubStageId:  raidData.CurrentSubStageId,
		SubStageInfos:      make([]*pb.SubStageInfo, 0),
		AllMonsterTemplate: make([]*pb.MonsterTemplateInfo, 0),
		RandomSeed:         raidData.RandomSeed,
	}
	for _, subStageId := range raidData.SubStageIds {
		subStageInfo := raidData.SubStageInfo[subStageId]
		if subStageInfo == nil {
			continue
		}
		info := &pb.SubStageInfo{
			SubStageId:         subStageInfo.SubStageId,
			RoomId:             subStageInfo.RoomID,
			MonsterInfos:       make([]*pb.MonsterInfo, 0),
			HeroInfos:          GetActiveHeroInfos(playerModel, GetRaidFormationType(playerModel.PlayerInstanceModel.CurrentRaidInfo)),
			SelfComboSkillIds:  GetComboSkillIds(playerModel, GetRaidFormationType(playerModel.PlayerInstanceModel.CurrentRaidInfo)),
			EnemyComboSkillIds: subStageInfo.ComboSkills,
			PlayerLevel:        playerModel.GetLevel(),
		}
		sceneBasicInfo.SubStageInfos = append(sceneBasicInfo.SubStageInfos, info)
		if subStageId == raidData.CurrentSubStageId {
			info.IsCyCle = raidData.StageInfo.IsCycle
			for _, monster := range subStageInfo.MonsterInfo {
				monsterPb := &pb.MonsterInfo{
					Index:        monster.Id,
					SpawnId:      monster.SpawnId,
					WaveSequence: monster.WaveSequence,
					MonsterId:    monster.MonsterId,
					Drops:        make([]*pb.ItemBasicInfo, 0),
				}
				for _, dropItem := range monster.DropItems {
					monsterPb.Drops = append(monsterPb.Drops, &pb.ItemBasicInfo{
						ItemId: dropItem.ID,
						Count:  dropItem.Num,
					})
				}
				info.MonsterInfos = append(info.MonsterInfos, monsterPb)
			}
		}
	}
	for _, monsterInfo := range raidData.MonsterTemplates {
		template := &pb.MonsterTemplateInfo{
			MonsterId:   int32(monsterInfo.MonsterId),
			UnitId:      monsterInfo.UnitId,
			AtkSpeed:    monsterInfo.AtkSpeed,
			MoveSpeed:   monsterInfo.MoveSpeed,
			PatrolRange: monsterInfo.PatrolRange,
			AggroRange:  monsterInfo.AggroRange,
			AttackRange: monsterInfo.AttackRange,
			BasicSkill:  monsterInfo.BasicSkill,
			Attrs:       make(map[int32]int64),
			SkillIds:    make([]int32, 0),
		}
		for k, v := range monsterInfo.AttrInfo {
			template.Attrs[k] = v
		}
		for _, skillId := range monsterInfo.SkillId {
			template.SkillIds = append(template.SkillIds, skillId)
		}
		sceneBasicInfo.AllMonsterTemplate = append(sceneBasicInfo.AllMonsterTemplate, template)
	}
	return sceneBasicInfo
}

func GetActiveHeroInfos(playerModel *model.PlayerModel, id pb.HeroFormationType) []*pb.HeroBasicInfo {
	var heroInfos []*pb.HeroBasicInfo
	if playerModel == nil || playerModel.HeroFormationModel == nil {
		return heroInfos
	}
	heroFormationDetails := playerModel.HeroFormationModel.Entities
	for _, f := range heroFormationDetails[int32(id)] {
		if f.IsActive {
			for _, heroOwnId := range f.HeroOwnIDList {
				heroInfo := buildFormationHeroBasicInfo(playerModel, id, f.FormationID, heroOwnId)
				if heroInfo == nil {
					continue
				}
				heroInfos = append(heroInfos, heroInfo)
			}
			return heroInfos
		}
	}
	logger.ErrorBySprintf("no active formation for formation type: %d", id)
	for _, f := range heroFormationDetails[0] {
		if f.IsActive {
			for _, heroOwnId := range f.HeroOwnIDList {
				heroInfo := buildFormationHeroBasicInfo(playerModel, pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN, f.FormationID, heroOwnId)
				if heroInfo == nil {
					continue
				}
				heroInfos = append(heroInfos, heroInfo)
			}
			return heroInfos
		}
	}
	return heroInfos
}

func buildFormationHeroBasicInfo(playerModel *model.PlayerModel, formationType pb.HeroFormationType, formationID int32, heroOwnID int64) *pb.HeroBasicInfo {
	if playerModel == nil || playerModel.HeroDetailsModel == nil {
		return nil
	}

	heroDetail := playerModel.HeroDetailsModel.Entities[heroOwnID]
	if heroDetail != nil {
		heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
		if heroBaseCfg == nil {
			return nil
		}
		return &pb.HeroBasicInfo{
			HeroId: int32(heroDetail.HeroID),
			BasicInfo: &pb.CharactorBasicInfo{
				Attrs:         playerModel.GetHeroAttrForBattle(heroOwnID, int32(formationType), formationID),
				SkillIds:      model.GetHeroSkills(heroDetail, playerModel.EquipmentModel),
				UnitsId:       gameConfig.GetHeroUnitsId(int32(heroDetail.HeroID), heroDetail.StarLevel, heroDetail.EvolutionPath),
				AttackRange:   heroBaseCfg.AttackRange,
				PetBattleInfo: model.GetHeroPetBattleInfo(heroOwnID, playerModel.PetModel),
			},
		}
	}

	if formationType != pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA || playerModel.PlayerGloryArenaModel == nil {
		return nil
	}

	selectedHero := getGloryArenaSelectedHeroByOwnID(playerModel.PlayerGloryArenaModel.SelectedHeroes, heroOwnID)
	if selectedHero == nil || selectedHero.Id <= 0 {
		return nil
	}

	attrs := make(map[int32]int64, len(selectedHero.Attr))
	for attrID, value := range selectedHero.Attr {
		attrs[attrID] = value
	}
	skillIDs := append([]int32(nil), selectedHero.Skill...)

	attackRange := selectedHero.AttackRange
	if attackRange <= 0 {
		if heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(selectedHero.Id)); heroBaseCfg != nil {
			attackRange = heroBaseCfg.AttackRange
		}
	}
	unitsID := selectedHero.Units
	if unitsID <= 0 {
		unitsID = gameConfig.GetHeroUnitsId(int32(selectedHero.Id), selectedHero.Star, selectedHero.ClassId)
	}

	return &pb.HeroBasicInfo{
		HeroId: int32(selectedHero.Id),
		BasicInfo: &pb.CharactorBasicInfo{
			Attrs:         attrs,
			SkillIds:      &pb.SkillsInfo{SkillList: skillIDs},
			UnitsId:       unitsID,
			AttackRange:   attackRange,
			PetBattleInfo: selectedHero.PetInfo,
		},
	}
}

func getGloryArenaSelectedHeroByOwnID(selectedHeroes map[int32]*model.PlayerGloryArenaSelectedOpponentEntity, heroOwnID int64) *logicCommon.HeroBasicInfo {
	if len(selectedHeroes) == 0 || heroOwnID <= 0 {
		return nil
	}
	for _, selected := range selectedHeroes {
		if selected == nil || selected.SelectedHero == nil {
			continue
		}
		if selected.SelectedHero.Uid == heroOwnID {
			return selected.SelectedHero
		}
	}
	return nil
}

func GetComboSkillIds(player *model.PlayerModel, id pb.HeroFormationType) []int32 {
	heroIdMap := make(map[int32]int32)
	comboSkillMap := make(map[int32]int32)
	comboSkillIds := make([]int32, 0)
	for _, heroFormation := range player.HeroFormationModel.Entities[int32(id)] {
		if !heroFormation.IsActive {
			continue
		}
		for _, heroOwnId := range heroFormation.HeroOwnIDList {
			heroDetail := player.HeroDetailsModel.Entities[heroOwnId]
			if heroDetail == nil {
				continue
			}
			heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
			if heroBaseCfg == nil {
				continue
			}
			heroIdMap[int32(heroDetail.HeroID)] = heroDetail.StarLevel
			for _, v := range heroBaseCfg.ComboSkill {
				comboSkillMap[v]++
			}
		}
	}
	leaderIdForComboSkillMap := make(map[int32]int32)
	for key, v := range comboSkillMap {
		comboSkillCfg := gameConfig.GetComboSkillCfg(key)
		if comboSkillCfg == nil {
			continue
		}
		if v != int32(len(comboSkillCfg.LimitedHero)) {
			continue
		}
		flag := true
		for id, heroId := range comboSkillCfg.LimitedHero {
			if _, ok := heroIdMap[heroId]; !ok || (comboSkillCfg.StarLimit[id] > heroIdMap[heroId]) {
				flag = false
				break
			}
		}
		if flag {
			if _, ok := leaderIdForComboSkillMap[comboSkillCfg.LeaderId]; ok {
				lastComboSkillCfg := gameConfig.GetComboSkillCfg(leaderIdForComboSkillMap[comboSkillCfg.LeaderId])
				if lastComboSkillCfg != nil {
					if lastComboSkillCfg.Level <= comboSkillCfg.Level {
						leaderIdForComboSkillMap[comboSkillCfg.LeaderId] = comboSkillCfg.Id
					}
				}
			} else {
				leaderIdForComboSkillMap[comboSkillCfg.LeaderId] = comboSkillCfg.Id
			}
		}
	}
	for _, v := range leaderIdForComboSkillMap {
		comboSkillIds = append(comboSkillIds, v)
	}
	return comboSkillIds
}

func GetEnemyComberSkillIds(userId int64, formationInfo *logicCommon.FormationBasicInfo, formationHeroes map[int64]*logicCommon.HeroBasicInfo) []int32 {
	heroIdMap := make(map[int32]int32)
	comboSkillMap := make(map[int32]int32)
	comboSkillIds := make([]int32, 0)
	for _, heroId := range formationInfo.Heroes {
		heroDetail := formationHeroes[heroId]
		if heroDetail == nil {
			continue
		}
		heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.Id))
		if heroBaseCfg == nil {
			continue
		}
		heroIdMap[int32(heroDetail.Id)] = heroDetail.Star
		for _, v := range heroBaseCfg.ComboSkill {
			comboSkillMap[v]++
		}
	}
	for key, v := range comboSkillMap {
		comboSkillCfg := gameConfig.GetComboSkillCfg(key)
		if comboSkillCfg == nil {
			continue
		}
		if v != int32(len(comboSkillCfg.LimitedHero)) {
			continue
		}
		flag := true
		for id, heroId := range comboSkillCfg.LimitedHero {
			if _, ok := heroIdMap[heroId]; !ok || (comboSkillCfg.StarLimit[id] > heroIdMap[heroId]) {
				flag = false
				break
			}
		}
		if flag {
			comboSkillIds = append(comboSkillIds, key)
		}
	}
	return comboSkillIds
}

func GetFormationType(instance int32) pb.HeroFormationType {
	switch instance {
	case int32(enum.MAIN_INSTANCE_ID):
		return pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN
	case int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID):
		return pb.HeroFormationType_HERO_FORMATION_TYPE_FIVE_VS_FIVE
	case int32(enum.ARENA_INSTANCE_ID):
		return pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_ATK
	default:
		logger.ErrorBySprintf("invalid instance id for get formation type, instance id: %d", instance)
		return pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN
	}
}

func GetRaidFormationType(raidInfo *logicCommon.PlayerInstanceRaid) pb.HeroFormationType {
	if raidInfo.FormationType != 0 {
		return pb.HeroFormationType(raidInfo.FormationType)
	}
	return GetFormationType(int32(raidInfo.InstanceID))
}

func LoginEnterScene(player *model.PlayerModel) error {
	scenePlayer := logicScene.NewScenePlayer(player.GetUserId())
	raidData := player.PlayerInstanceModel.CurrentRaidInfo
	if raidData == nil {
		return errors.New("[platform] raidData is nil")
	}
	err := sceneService.EnterScene(scenePlayer, int32(raidData.InstanceID))
	if err != nil {
		return err
	}
	player.SceneId = scenePlayer.SceneId
	return nil
}

func EnterScene(player *model.PlayerModel, raidData *logicCommon.PlayerInstanceRaid) error {
	scenePlayer, err := sceneService.LeaveScene(player.GetUserId())
	if err != nil {
		return err
	}
	if raidData.InstanceID == enum.MAIN_INSTANCE_ID {
		player.PlayerInstanceModel.CurrentRaidInfo = player.PlayerInstanceModel.CurrentMainInstanceInfo
	} else {
		player.PlayerInstanceModel.CurrentRaidInfo = raidData
	}
	err = sceneService.EnterScene(scenePlayer, int32(raidData.InstanceID))
	if err != nil {
		return err
	}
	player.SceneId = scenePlayer.SceneId
	return nil
}

func PlayerSceneLoadOver(player *model.PlayerModel) error {
	err := sceneService.PlayerSceneLoadOver(player.GetUserId())
	return err
}
