package raid

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
)

func OnLeaveRaidCommon(info *logicCommon.PlayerInstanceRaid) {
	for _, stageInfo := range info.SubStageInfo {
		for _, monster := range stageInfo.MonsterInfo {
			monster.IsDead = 0
		}
	}
}

func BattleOperationCommon(info *logicCommon.PlayerInstanceRaid, id int64, x int32, y int32) error {
	return nil
}

func KillMonsterCommon(raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	items := make([]*gameConfig.ItemConfig, 0)
	stageInfo := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	monsterIdList := make([]int32, 0)
	if stageInfo == nil {
		return nil, nil, pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	for _, monsterId := range monsterIds {
		monsterInfo := stageInfo.MonsterInfo[monsterId]
		if monsterInfo == nil {
			continue
		}
		if monsterInfo.IsDead == 1 {
			continue
		}
		monsterIdList = append(monsterIdList, monsterInfo.MonsterId)
		monsterInfo.IsDead = 1
		items = append(items, monsterInfo.DropItems...)
		if _, ok := raidInfo.StageInfo.KillMonsterMap[monsterId]; !ok {
			raidInfo.StageInfo.KillMonsterId = append(raidInfo.StageInfo.KillMonsterId, monsterId)
			raidInfo.StageInfo.KillMonsterMap[monsterId] = true
		}
	}
	return monsterIdList, items, pb.ERROR_CODE_SUCCESS
}

func CheckRaidEndCommon(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	lastStageId := raidInfo.SubStageIds[len(raidInfo.SubStageIds)-1]
	if raidInfo.CurrentSubStageId != lastStageId {
		return false
	}
	stageInfo := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if stageInfo == nil {
		return false
	}
	for _, monster := range stageInfo.MonsterInfo {
		monsterCfg := gameConfig.GetMonsterCfg(monster.MonsterId)
		if monsterCfg == nil {
			continue
		}
		if monsterCfg.Type == int32(enum.MonsterType_BUCKET) {
			continue
		}
		if monster.IsDead != 1 {
			return false
		}
	}
	return true
}

func CheckCurrentSubStageOverCommon(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	stageInfo := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if stageInfo == nil {
		return false
	}
	for _, monster := range stageInfo.MonsterInfo {
		monsterCfg := gameConfig.GetMonsterCfg(monster.MonsterId)
		if monsterCfg == nil {
			continue
		}
		if monsterCfg.Type == int32(enum.MonsterType_BUCKET) {
			continue
		}
		if monster.IsDead != 1 {
			return false
		}
	}
	return true
}

func CycleCurrentStageCommon(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32) *pb.SubStageInfo {
	currentStage := raidInfo.SubStageInfo[raidInfo.CurrentSubStageId]
	if currentStage == nil {
		return nil
	}
	for _, monster := range currentStage.MonsterInfo {
		monster.IsDead = 0
		waveCfg := gameConfig.GetMonsterWaveCfg(monster.WaveId)
		if waveCfg != nil {
			monster.DropItems = gameConfig.Drop(waveCfg.EachDrop)
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
	return info
}
