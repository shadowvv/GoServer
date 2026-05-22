package raid

import (
	"errors"

	"github.com/drop/GoServer/server/logic/adventure"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

const (
	adventureBattleWin  = 1
	adventureBattleLose = 0
)

type AdventureInstanceRaid struct {
	AdventureRaidOperation
}

var _ RaidOperatorInterface = (*AdventureInstanceRaid)(nil)

func (a *AdventureInstanceRaid) CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool {
	return true
}

func (a *AdventureInstanceRaid) OnLeaveRaid(info *logicCommon.PlayerInstanceRaid) {
	OnLeaveRaidCommon(info)
}

func (a *AdventureInstanceRaid) GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (a *AdventureInstanceRaid) GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE) {
	return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
}

func (a *AdventureInstanceRaid) BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	return BuildAdventureInstanceRaid(raidInfo)
}

func (a *AdventureInstanceRaid) OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	return a.finishAdventure(player, info, adventureBattleWin)
}

func (a *AdventureInstanceRaid) OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	return a.finishAdventure(player, info, adventureBattleLose)
}

// finishAdventure 结算冒险副本，没有输赢概念
func (a *AdventureInstanceRaid) finishAdventure(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, isWin int32) (pb.ERROR_CODE, *pb.PushStageBattleWin) {
	errCode := adventure.OnAdventureSettle(player, info)
	if errCode != pb.ERROR_CODE_SUCCESS {
		if info != nil {
			info.IsOver = true
		}
		if info != nil && isWin == adventureBattleWin {
			adventure.CancelStartAdventure(player, info.AdventureUniqueId)
		}
		return errCode, nil
	}
	addDungeonSettleDropItems(player, info)
	return pb.ERROR_CODE_SUCCESS, &pb.PushStageBattleWin{
		IsWin:      isWin,
		InstanceId: int32(info.InstanceID),
		StageId:    info.CurrentStageId,
	}
}

func BuildAdventureInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error {
	dungeonCfg := gameConfig.GetDungeonAdventureCfg(raidInfo.CurrentStageId)
	if dungeonCfg == nil {
		return errors.New("adventure dungeon config not exist")
	}
	formationType := GetAdventureFormationType(dungeonCfg.Type)

	raidInfo.BattleId = battleIdGen.NextId()
	raidInfo.RandomSeed = tool.UnixNowMilli()
	raidInfo.CurrentSubStageId = raidInfo.CurrentStageId
	raidInfo.FormationType = int32(formationType)

	subStageData := buildDungeonRaidSubStage(raidInfo, dungeonCfg)
	raidInfo.SubStageInfo[subStageData.SubStageId] = subStageData
	raidInfo.SubStageIds = append(raidInfo.SubStageIds, subStageData.SubStageId)
	return nil
}

func GetAdventureFormationType(adventureType int32) pb.HeroFormationType {
	return pb.HeroFormationType_HERO_FORMATION_TYPE_ADVENTURE
}
