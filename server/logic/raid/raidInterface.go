package raid

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

// 副本系统服务接口
type RaidOperatorInterface interface {
	// 检查能否进入副本房间
	CanEnterInstanceStage(enterStageId int32, currentStageId int32) bool
	// 副本离开
	OnLeaveRaid(info *logicCommon.PlayerInstanceRaid)
	// 创建副本信息
	BuildInstanceRaid(raidInfo *logicCommon.PlayerInstanceRaid) error
	// 副本快速战斗
	GetWeepReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE)
	// 获取副本奖励
	GetInstanceCommitReward(stageId int32) ([]*gameConfig.ItemConfig, pb.ERROR_CODE)
	// 副本战斗操作
	BattleOperation(raidInfo *logicCommon.PlayerInstanceRaid, playerId int64, x, y int32) error
	// 击杀怪物
	KillMonster(player *model.PlayerModel, raidInfo *logicCommon.PlayerInstanceRaid, monsterId []int32) ([]int32, []*gameConfig.ItemConfig, pb.ERROR_CODE)
	// 检查进入下一个房间
	CheckEnterNextSubStage(info *logicCommon.PlayerInstanceRaid, subStageId int32, player *model.PlayerModel) pb.ERROR_CODE
	// 进入下一个副本房间
	EnterNextSubStage(raidInfo *logicCommon.PlayerInstanceRaid, SubStageId int32, player *model.PlayerModel) (*pb.EnterNextSubStageResp, pb.ERROR_CODE)
	// 检查副本是否结束
	CheckRaidEnd(raidInfo *logicCommon.PlayerInstanceRaid) bool
	// 副本结束
	OnRaidEnd(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, playerInstanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin)
	// 检查当前房间是否结束
	CheckCurrentSubStageOver(raidInfo *logicCommon.PlayerInstanceRaid) bool
	// 获取循环房间
	ResetCurrentStage(raidInfo *logicCommon.PlayerInstanceRaid, privilegeDropCount *int32, player *model.PlayerModel) *pb.SubStageInfo
	// 副本战斗玩家死亡
	OnBattlePlayerDead(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, instanceModel *model.PlayerInstanceModel) (pb.ERROR_CODE, *pb.PushStageBattleWin)
}
