package raid

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

type AdventureRaidOperation struct {
}

const adventureMonsterInstanceIdFactor int32 = 100000

func (a *AdventureRaidOperation) BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error {
	return BattleOperationCommon(raidInfo, playerId, x, y)
}

func (a *AdventureRaidOperation) KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterIds []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE) {
	monsterIdList, dropItems, errCode := KillMonsterCommon(raidInfo, monsterIds)
	if errCode == pb.ERROR_CODE_SUCCESS {
		eventBusService.SubmitKillMonsterEvent(player.GetUserId(), int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID), monsterIdList)
	}
	return monsterIdList, dropItems, errCode
}

func (a *AdventureRaidOperation) CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckRaidEndCommon(raidInfo)
}

func (a *AdventureRaidOperation) CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool {
	return CheckCurrentSubStageOverCommon(raidInfo)
}

func (a *AdventureRaidOperation) CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE {
	if info == nil || info.SubStageInfo == nil {
		return pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	if info.SubStageInfo[subStageId] == nil {
		return pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	nextSubStageId := a.getNextSubStageId(info)
	if nextSubStageId == 0 || nextSubStageId != subStageId {
		return pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	if !CheckCurrentSubStageOverCommon(info) {
		return pb.ERROR_CODE_ENTER_SUB_STAGE_MONSTER_NOT_DEAD
	}
	return pb.ERROR_CODE_SUCCESS
}

func (a *AdventureRaidOperation) EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE) {
	if raidInfo == nil || raidInfo.SubStageInfo == nil {
		return nil, pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	stageInfo := raidInfo.SubStageInfo[subStageId]
	if stageInfo == nil {
		return nil, pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	nextSubStageId := a.getNextSubStageId(raidInfo)
	if nextSubStageId == 0 || nextSubStageId != subStageId {
		return nil, pb.ERROR_CODE_SUB_STAGE_IS_INVALID
	}
	if !CheckCurrentSubStageOverCommon(raidInfo) {
		return nil, pb.ERROR_CODE_ENTER_SUB_STAGE_MONSTER_NOT_DEAD
	}
	raidInfo.CurrentSubStageId = subStageId
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
	return resp, pb.ERROR_CODE_SUCCESS
}

func (a *AdventureRaidOperation) ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo {
	return nil
}

func (a *AdventureRaidOperation) getNextSubStageId(info *logicCommon.PlayerInstanceRaid) int32 {
	if info == nil || len(info.SubStageIds) == 0 {
		return 0
	}
	for index, subStageId := range info.SubStageIds {
		if subStageId != info.CurrentSubStageId {
			continue
		}
		if index+1 >= len(info.SubStageIds) {
			return 0
		}
		return info.SubStageIds[index+1]
	}
	return 0
}
