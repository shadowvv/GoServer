package raid

import (
	"errors"

	"github.com/drop/GoServer/server/logic/adventure"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/service/logger"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
)

type MainInstanceOperation struct {
}

var _ RaidOperatorInterface = (*MainInstanceOperation)(nil)

func (i *MainInstanceOperation) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	// 因为主线不需要主动进入所以返回  false
	return false
}

func (i *MainInstanceOperation) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	OnLeaveRaidCommon(info)
}
func (i *MainInstanceOperation) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	return errors.New("main instance build other way")
}

func (i *MainInstanceOperation) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SYSTEM_ERROR
}

func (i *MainInstanceOperation) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_SYSTEM_ERROR
}

func (i *MainInstanceOperation) BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	return BattleOperationCommon(raidInfo, playerId, x, y)
}

func (i *MainInstanceOperation) KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	monsterIdList, dropItems, errCode := KillMonsterCommon(raidInfo, monsterIds)
	if errCode == pb.ERROR_CODE_SUCCESS {
		count, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_MAIN_REWARD)
		if player.StaticData.GetDailyPrivilegeDrop() > 0 && err == nil && count > 0 {
			player.StaticData.UpdateDailyPrivilegeDrop(player.StaticData.GetDailyPrivilegeDrop() - int32(len(monsterIdList)))
		}
		adventure.OnMainStageKillMonster(player, int32(len(monsterIdList)))
	}
	player.PlayerInstanceModel.OnKillMonster()
	eventBusService.SubmitKillMonsterEvent(player.GetUserId(), int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID), monsterIdList)
	return monsterIdList, dropItems, errCode
}
func (i *MainInstanceOperation) CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckRaidEndCommon(raidInfo)
}
func (i *MainInstanceOperation) CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckCurrentSubStageOverCommon(raidInfo)
}

func (i *MainInstanceOperation) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	info.RandomSeed = tool.UnixNowMilli()
	info.StageInfo.IsCycle = 0
	for _, stageInfo := range info.SubStageInfo {
		for _, monster := range stageInfo.MonsterInfo {
			monster.IsDead = 0
		}
	}
	if playerInstanceModel.CurrentMainInstanceInfo.CurrentStageId >= playerInstanceModel.NextMainInstanceInfo.CurrentStageId {
		logger.ErrorBySprintf("[debug] next stage is less current stage,playerId:%d,currentStageId:%d,nextStageId:%d", player.GetUserId(),
			playerInstanceModel.CurrentMainInstanceInfo.CurrentStageId, playerInstanceModel.NextMainInstanceInfo.CurrentStageId)
	}
	playerInstanceModel.CurrentMainInstanceInfo = playerInstanceModel.NextMainInstanceInfo
	playerInstanceModel.CurrentRaidInfo = playerInstanceModel.CurrentMainInstanceInfo
	mainInstance := playerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if mainInstance == nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}

	extraDrop := int32(0)
	count, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_MAIN_REWARD)
	if err == nil && count > 0 {
		extraDrop = player.StaticData.GetDailyPrivilegeDrop()
	}
	next, err := BuildNextMainInstanceData(info.PlayerId, playerInstanceModel.CurrentMainInstanceInfo.CurrentStageId, mainInstance.MaxStageId, mainInstance.MaxSubStageId, extraDrop)
	if err != nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, nil
	}
	playerInstanceModel.NextMainInstanceInfo = next

	passService.UpdateAllPassProgressBySystem(player, 203) // 203=主线关卡

	cfg := gameConfig.GetMainStageCfg(info.CurrentStageId)
	if cfg != nil {
		if mainInstance.MaxStageId < info.CurrentStageId {
			for _, mailId := range cfg.MailId {
				_, _ = mailService.SendMailByTemplateID(player.GetUserId(), mailId)
			}
		}
	}

	operationLogService.OnUserMainInstanceChange(player.GetUserId(), 1, info.CurrentStageId)
	playerInstanceModel.OnPassMainInstance(info.CurrentStageId, info.CurrentSubStageId, info.StageInfo.IsCycle)
	playerInstanceModel.UpdateMainInstanceInfo(playerInstanceModel.CurrentRaidInfo.CurrentStageId, playerInstanceModel.CurrentRaidInfo.CurrentSubStageId, 0)
	adventure.OnMainStageChanged(player)

	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      1,
		InstanceId: int32(info.InstanceID),
		StageId:    playerInstanceModel.CurrentRaidInfo.CurrentStageId,
	}
}

func (i *MainInstanceOperation) CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	if subStageId == info.SubStageIds[len(info.SubStageIds)-1] {
		mainCfg := gameConfig.GetMainStageCfg(info.CurrentStageId)
		if mainCfg == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR
		}
		if mainCfg.Unlock != 0 && !unlockService.CheckUnlock(mainCfg.Unlock, player) {
			return pb.ERROR_CODE_NEXT_STAGE_IS_LOCKED
		}
	}
	return pb.ERROR_CODE_SUCCESS
}

func (i *MainInstanceOperation) ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo {
	killMonsterId := make(map[int32]bool)
	if player != nil && player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)] != nil {
		instance := player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
		killMonsterId = instance.Info.KillMonsterMap
	}

	currentStage := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if currentStage == nil {
		return nil
	}
	for _, monster := range currentStage.MonsterInfo {
		monster.IsDead = 0
		waveCfg := gameConfig.GetMonsterWaveCfg(monster.WaveId)
		if waveCfg != nil {
			if _, ok := killMonsterId[monster.Id]; ok {
				monster.DropItems = gameConfig.Drop(waveCfg.EachDrop)
				if monster.WaveId == 20506 {
					logger.ErrorBySprintf("DEBUG stage2 boss eachDrop,playerId:%d", raidInfo.PlayerId)
				}
			} else {
				monster.DropItems = gameConfig.Drop(waveCfg.FirstDrop)
			}
			if waveCfg.PrivilegeDrop > 0 && *privilegeDropCount > 0 {
				monster.DropItems = append(monster.DropItems, gameConfig.Drop(waveCfg.PrivilegeDrop)...)
				*privilegeDropCount--
			}
		}
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
		for _, dropItem := range monster.DropItems {
			monsterPb.Drops = append(monsterPb.Drops, &pb.ItemBasicInfo{
				ItemId: dropItem.ID,
				Count:  dropItem.Num,
			})
		}
		info.MonsterInfos = append(info.MonsterInfos, monsterPb)
	}
	raidInfo.IsOver = false
	adventure.OnMainStageChanged(player)
	return info
}

func (i *MainInstanceOperation) EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	resetCycle := true
	lastCommonStage := int32(0)
	if len(raidInfo.SubStageIds) > 2 {
		lastCommonStage = raidInfo.SubStageIds[len(raidInfo.SubStageIds)-2]
	} else {
		lastCommonStage = raidInfo.SubStageIds[len(raidInfo.SubStageIds)-1]
	}
	if subStageId == lastCommonStage {
		mainCfg := gameConfig.GetMainStageCfg(player.PlayerInstanceModel.CurrentRaidInfo.CurrentStageId)
		if mainCfg == nil {
			return nil, pb.ERROR_CODE_SYSTEM_ERROR
		}
		if mainCfg.Unlock != 0 && !unlockService.CheckUnlock(mainCfg.Unlock, player) {
			player.PlayerInstanceModel.CurrentRaidInfo.StageInfo.IsCycle = 1
			resetCycle = false
		}
	}
	currentStage := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if currentStage != nil && raidInfo.StageInfo.IsCycle != 1 {
		for _, monster := range currentStage.MonsterInfo {
			monsterCfg := gameConfig.GetMonsterCfg(monster.MonsterId)
			if monsterCfg == nil {
				continue
			}
			if monsterCfg.Type == int32(enum.MonsterType_BUCKET) {
				continue
			}
			if monster.IsDead == 1 {
				continue
			}
			return nil, pb.ERROR_CODE_ENTER_SUB_STAGE_MONSTER_NOT_DEAD
		}
	}
	if resetCycle {
		raidInfo.StageInfo.IsCycle = 0
	}
	player.PlayerInstanceModel.OnPassMainInstance(0, raidInfo.CurrentSubStageId, raidInfo.StageInfo.IsCycle)
	raidInfo.CurrentSubStageId = subStageId
	stageInfo := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if stageInfo == nil {
		return nil, pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	info := &pb.SubStageInfo{
		SubStageId:   stageInfo.SubStageId,
		RoomId:       stageInfo.RoomID,
		MonsterInfos: make([]*pb.MonsterInfo, 0),
		IsCyCle:      raidInfo.StageInfo.IsCycle,
	}
	for _, monster := range stageInfo.MonsterInfo {
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
	resp := &pb.EnterNextSubStageResp{
		SubStageInfo: info,
	}

	player.PlayerInstanceModel.UpdateMainInstanceInfo(0, player.PlayerInstanceModel.CurrentRaidInfo.CurrentSubStageId, player.PlayerInstanceModel.CurrentRaidInfo.StageInfo.IsCycle)
	if subStageId == raidInfo.SubStageIds[len(raidInfo.SubStageIds)-1] {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_MAIN_INSTANCE_NEXT_STAGE_INFO, &pb.PushMainInstanceNextStageInfo{
			Info: BuildRaidPB(player, player.PlayerInstanceModel.NextMainInstanceInfo),
		})
	}
	adventure.OnMainStageChanged(player)
	return resp, pb.ERROR_CODE_SUCCESS
}

func (i *MainInstanceOperation) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	currentSubStageId := info.CurrentSubStageId
	instanceModel.UpdateLastDeadMainInstanceStageId(info.CurrentStageId)
	info.RandomSeed = tool.UnixNowMilli()
	info.CurrentSubStageId = info.SubStageIds[0]
	info.StageInfo.IsCycle = 0
	for _, stageInfo := range info.SubStageInfo {
		for _, monster := range stageInfo.MonsterInfo {
			monster.IsDead = 0
		}
	}
	info.IsOver = false

	operationLogService.OnUserMainInstanceChange(player.GetUserId(), 0, info.CurrentStageId)
	if currentSubStageId != info.SubStageIds[len(info.SubStageIds)-1] {
		if gameConfig.GetBaseCfg() != nil && info.CurrentStageId != gameConfig.GetBaseCfg().Stage {
			instanceModel.NextMainInstanceInfo = instanceModel.CurrentMainInstanceInfo
		}
		extraDrop := int32(0)
		count, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_MAIN_REWARD)
		if err == nil && count > 0 {
			extraDrop = player.StaticData.GetDailyPrivilegeDrop()
		}
		mainInstance := instanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
		if mainInstance == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		pre, err := BuildPreMainInstanceData(info.PlayerId, instanceModel.CurrentMainInstanceInfo.CurrentStageId, mainInstance.MaxStageId, mainInstance.MaxSubStageId, extraDrop)
		if err != nil {
			return pb.ERROR_CODE_SYSTEM_ERROR, nil
		}
		instanceModel.CurrentRaidInfo = pre
		instanceModel.CurrentMainInstanceInfo = instanceModel.CurrentRaidInfo
	}
	instanceModel.CurrentMainInstanceInfo.StageInfo.IsCycle = 1
	currentSubStageId = instanceModel.CurrentMainInstanceInfo.SubStageIds[0]
	subStageNum := len(instanceModel.CurrentMainInstanceInfo.SubStageIds)
	if subStageNum > 2 {
		currentSubStageId = instanceModel.CurrentMainInstanceInfo.SubStageIds[subStageNum-2]
	}
	instanceModel.CurrentRaidInfo.CurrentSubStageId = currentSubStageId
	instanceModel.UpdateMainInstanceInfo(instanceModel.CurrentRaidInfo.CurrentStageId, instanceModel.CurrentRaidInfo.CurrentSubStageId, 1)
	adventure.OnMainStageChanged(player)
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      0,
		InstanceId: int32(info.InstanceID),
		StageId:    instanceModel.CurrentRaidInfo.CurrentSubStageId,
	}
}
