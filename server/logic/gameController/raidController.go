package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/raid"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("raid", &RaidController{})

}

type RaidController struct {
}

func (s *RaidController) RegisterLogicMessage() {

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_INSTANCE_INFO_REQ, &pb.GetInstanceInfoReq{}, GetInstanceInfoHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_QUICK_BATTLE_REQ, &pb.QuickBattleReq{}, QuickBattleHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_COMMIT_INSTANCE_LEVEL_REWARD_REQ, &pb.CommitInstanceLevelRewardReq{}, CommitInstanceRewardHandler, enum.FUNCTION_ID_NONE)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ENTER_SCENE_REQ, &pb.EnterSceneReq{}, SceneEnterHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LOAD_SCENE_OVER_REQ, &pb.LoadSceneOverReq{}, SceneLoadOverHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LEAVE_SCENE_REQ, &pb.LeaveSceneReq{}, SceneLeaveHandle, enum.FUNCTION_ID_NONE)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BATTLE_OPERATION_REQ, &pb.BattleOperationReq{}, BattleOperationHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_Kill_MONSTER_REQ, &pb.KillMonsterReq{}, KillMonsterHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ENTER_NEXT_SUB_STAGE_REQ, &pb.EnterNextSubStageReq{}, EnterNextSubStageHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CHECK_ENTER_NEXT_SUB_STAGE_REQ, &pb.CheckEnterNextSubStageReq{}, CheckEnterNextSubStageHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_REQ, &pb.BattlePlayerDeadReq{}, BattlePlayerDeadHandler, enum.FUNCTION_ID_NONE)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NOTIFY_TO_GAME_CONFIRM_MESSAGE, &pb.NotifyToGameConfirmMessage{}, ConfirmBattleMessageHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NOTIFY_TO_GAME_KILL_MONSTER, &pb.NotifyToGameKillMonster{}, BattleNotifyGameKillMonsterHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NOTIFY_TO_GAME_HERO_DEAD, &pb.NotifyToGameHeroDead{}, BattleNotifyGameHeroDeadHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NOTIFY_TO_GAME_BATTLE_RESULT, &pb.NotifyToGameBattleResult{}, BattleNotifyGameBattleResultHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NOTIFY_TO_GAME_ENTER_SUB_STAGE, &pb.NotifyToGameEnterSubStage{}, BattleNotifyGameEnterSubStageHandler, enum.FUNCTION_ID_NONE)

	RegisterPlayerInnerTask(enum.INNER_MSG_PLAYER_LOGOUT, playerLogoutHandle)
}

var _ LogicControllerInterface = (*RaidController)(nil)

func CheckInstanceUnlock(player *model.PlayerModel, instanceType int32) bool {
	systemId := enum.GetSystemIdByInstanceId(enum.InstanceTypeEnum(instanceType))
	return unlockService.CheckSystemUnlock(int32(systemId), player)
}

func GetInstanceInfoHandler(message proto.Message, player *model.PlayerModel) {
	instanceInfo := make([]*pb.InstanceInfo, 0)
	allInstance := gameConfig.GetAllInstance()
	for instanceId, instanceCfg := range allInstance {
		if instanceId == int32(enum.InstanceType_MAIN) {
			continue
		}
		if !CheckInstanceUnlock(player, instanceCfg.InstanceType) {
			continue
		}
		instanceData := player.PlayerInstanceModel.InstanceEntities[instanceId]
		if instanceData == nil && len(instanceCfg.RecoveryTicketID) > 0 {
			err := player.PlayerInstanceModel.UpdateInstanceInfo(instanceId, 0, 0)
			if err != nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_INSTANCE_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			instanceData = player.PlayerInstanceModel.InstanceEntities[instanceId]
			_ = itemService.ResetItems(player, instanceCfg.RecoveryTicketID, enum.ITEM_CHANGE_REASON_SYSTEM_UNLOCK_REWARD)
		}
		stageId := int32(0)
		rewardCommited := int32(0)
		if instanceData != nil {
			stageId = instanceData.StageId
			rewardCommited = instanceData.CommitLevelReward
		}
		instanceInfo = append(instanceInfo, &pb.InstanceInfo{
			InstanceId:     instanceId,
			StageId:        stageId,
			RewardCommited: rewardCommited,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_INSTANCE_INFO_RESP, &pb.GetInstanceInfoResp{
		Infos: instanceInfo,
	})
}

func SceneEnterHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EnterSceneReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	config := gameConfig.GetInstanceCfg(req.InstanceId)
	if config == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID) == req.InstanceId {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_ENTER_SCENE_REPEAT)
		return
	}
	if player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.MAIN_INSTANCE_ID {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if !CheckInstanceUnlock(player, config.InstanceType) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}
	check, err := itemService.CheckItemsCount(player, config.TicketID)
	if !check || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	currentStageId := int32(0)
	commitLevelReward := int32(0)
	raidData := &logicCommon.PlayerInstanceRaid{
		PlayerId:         player.GetUserId(),
		InstanceID:       enum.InstanceId(req.InstanceId),
		CurrentStageId:   req.StageId,
		SubStageInfo:     make(map[int32]*logicCommon.SubStageData),
		SubStageIds:      make([]int32, 0),
		MonsterTemplates: make(map[int64]*logicCommon.MonsterTemplate),
	}
	entity := player.PlayerInstanceModel.InstanceEntities[req.InstanceId]
	if entity != nil {
		raidData.StageInfo = entity.Info
		currentStageId = entity.StageId
		commitLevelReward = entity.CommitLevelReward
	} else {
		raidData.StageInfo = logicCommon.NewInstanceStageInfo()
	}

	if !raid.CanEnterInstanceStage(req.InstanceId, req.StageId, currentStageId) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if enum.IsResidentInstanceType(config.InstanceType) && !raid.CanEnterResidentInstanceStage(config.InstanceType, req.StageId, currentStageId, commitLevelReward) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	raid.OnLeaveRaid(player.PlayerInstanceModel.CurrentRaidInfo)

	err = raid.BuildInstanceRaid(raidData)
	if err != nil {
		platformLogger.ErrorWithUser("enter instance error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	loginMutexService.EnterMutex(player.GetUserAccount(), player.GetUserId())
	err = raid.EnterScene(player, raidData)
	loginMutexService.ExitMutex(player.GetUserAccount(), player.GetUserId())

	if err != nil {
		platformLogger.ErrorWithUser("enter scene error", player, err)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_ENTER_SCENE_RESP, &pb.EnterSceneResp{
		Info: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	if enum.IsResidentInstanceType(config.InstanceType) {
		player.StaticData.UpdateResidentInstanceJoinCount(player.StaticData.GetResidentInstanceJoinCount() + 1)
	}
	eventBusService.SubmitJoinInstanceEvent(player.GetUserId(), player.GetUserServerId(), raidData.InstanceID, raidData.CurrentStageId)
}

func SceneLoadOverHandle(message proto.Message, player *model.PlayerModel) {
	err := raid.PlayerSceneLoadOver(player)
	if err != nil {
		platformLogger.ErrorWithUser("enter scene load over error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOAD_SCENE_OVER_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}
	resp := &pb.LoadSceneOverResp{}
	raidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	if raidInfo == nil {
		platformLogger.ErrorWithUser("enter scene load over error", player, nil)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOAD_SCENE_OVER_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if raidInfo.InstanceID == enum.MAIN_INSTANCE_ID && raidInfo.CurrentSubStageId == player.PlayerInstanceModel.CurrentRaidInfo.SubStageIds[len(raidInfo.SubStageIds)-1] {
		resp.NextInfo = raid.BuildRaidPB(player, player.PlayerInstanceModel.NextMainInstanceInfo)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOAD_SCENE_OVER_RESP, resp)
}

func SceneLeaveHandle(message proto.Message, player *model.PlayerModel) {
	loginMutexService.EnterMutex(player.GetUserAccount(), player.GetUserId())
	defer loginMutexService.ExitMutex(player.GetUserAccount(), player.GetUserId())

	currentRaidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	leaveInstanceId := currentRaidInfo.InstanceID
	leaveStageId := currentRaidInfo.CurrentStageId
	if currentRaidInfo.InstanceID == enum.MAIN_INSTANCE_ID {
		platformLogger.ErrorWithUser("can not leave main instance", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_SCENE_RESP, pb.ERROR_CODE_ENTER_SCENE_REPEAT)
		return
	}
	raid.OnLeaveRaid(currentRaidInfo)

	err := raid.EnterScene(player, player.PlayerInstanceModel.CurrentMainInstanceInfo)
	if err != nil {
		platformLogger.ErrorWithUser("enter scene error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_SCENE_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LEAVE_SCENE_RESP, &pb.LeaveSceneResp{
		Info: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	if leaveInstanceId == enum.ADVENTURE_INSTANCE_ID {
		operationLogService.OnUserAdventureInstanceLeave(player.GetUserId(), leaveStageId)
	}
}

func QuickBattleHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.QuickBattleReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	config := gameConfig.GetInstanceCfg(req.InstanceId)
	if config == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if !CheckInstanceUnlock(player, config.InstanceType) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}
	if config.IsSweep == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	entity := player.PlayerInstanceModel.InstanceEntities[req.InstanceId]
	if entity == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	stageId, ok := raid.GetQuickBattleStage(config.InstanceType, entity.StageId, entity.CommitLevelReward, req.StageId)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	check, err := itemService.CheckItemsCount(player, config.TicketID)
	if !check || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItems(player, config.TicketID, enum.ITEM_CHANGE_REASON_SWEEP_INSTANCE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	dropItems, errorCode := raid.GetWeepReward(entity.InstanceId, stageId)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, errorCode)
		return
	}
	err = itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_SWEEP_INSTANCE)
	if err != nil {
		platformLogger.ErrorWithUser("weep instance add item error", player, err)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, &pb.QuickBattleResp{})
	if enum.IsResidentInstanceType(config.InstanceType) {
		player.StaticData.UpdateResidentInstanceJoinCount(player.StaticData.GetResidentInstanceJoinCount() + 1)
	}
	eventBusService.SubmitJoinInstanceEvent(player.GetUserId(), player.GetUserServerId(), enum.InstanceId(req.InstanceId), stageId)
}

func CommitInstanceRewardHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CommitInstanceLevelRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	config := gameConfig.GetInstanceCfg(req.InstanceId)
	if config == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if !CheckInstanceUnlock(player, config.InstanceType) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}
	entity := player.PlayerInstanceModel.InstanceEntities[req.InstanceId]
	if entity == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_QUICK_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if entity.CommitLevelReward >= entity.CurrentSubStageId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COMMIT_INSTANCE_LEVEL_REWARD_RESP, pb.ERROR_CODE_INSTANCE_LEVEL_REWARD_IS_COMMITED)
		return
	}
	dropItems, errorCode := raid.GetInstanceCommitReward(entity.InstanceId, entity.StageId)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COMMIT_INSTANCE_LEVEL_REWARD_RESP, errorCode)
		return
	}
	player.PlayerInstanceModel.CommitedLevelReward(req.InstanceId)
	err := itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_COMMIT_INSTANCE_LEVEL_REWARD)
	if err != nil {
		platformLogger.ErrorWithUser("commit instance reward error", player, err)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_COMMIT_INSTANCE_LEVEL_REWARD_RESP, &pb.CommitInstanceLevelRewardResp{})
}

func BattleOperationHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.BattleOperationReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BATTLE_OPERATION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	err := raid.BattleOperation(player.PlayerInstanceModel.CurrentRaidInfo, player.GetUserId(), req.X, req.Y)
	if err != nil {
		platformLogger.ErrorWithUser("battle operation error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BATTLE_OPERATION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_BATTLE_OPERATION_RESP, &pb.BattleOperationResp{})
}

func KillMonsterHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.KillMonsterReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if len(req.MonsterId) == 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, &pb.KillMonsterResp{})
		return
	}
	if req.StageId != player.PlayerInstanceModel.CurrentRaidInfo.CurrentStageId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, pb.ERROR_CODE_STAGE_HAS_SETTLE)
		return
	}
	if player.PlayerInstanceModel.CurrentRaidInfo.IsOver {
		messageSender.SendMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, &pb.KillMonsterResp{})
		return
	}
	// 常规副本和奇遇副本的怪物击杀掉落物品延迟到副本结算时再掉落
	currentRaidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	delayDropInstance := raid.DelayDropUntilSettle(int32(currentRaidInfo.InstanceID))
	_, dropItems, errCode := raid.KillMonster(player, currentRaidInfo, req.MonsterId)
	if errCode != pb.ERROR_CODE_SUCCESS {
		platformLogger.ErrorWithUser("kill monster failed", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, errCode)
		return
	}
	if !delayDropInstance {
		_ = itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_MONSTER_DROP)
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, &pb.KillMonsterResp{})

	if raid.CheckRaidEnd(currentRaidInfo) {
		currentRaidInfo.IsOver = true
		errorCode, winResp := raid.OnRaidEnd(player, currentRaidInfo, player.PlayerInstanceModel)
		if errorCode != pb.ERROR_CODE_SUCCESS {
			platformLogger.ErrorWithUser("raid end failed", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_Kill_MONSTER_RESP, errorCode)
			return
		}
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_STAGE_BATTLE_WIN, winResp)
		eventBusService.SubmitPassInstanceEvent(player.GetUserId(), player.GetUserServerId(), enum.InstanceId(winResp.InstanceId), winResp.StageId)
		return
	}

	if currentRaidInfo.StageInfo.IsCycle == 1 && raid.CheckCurrentSubStageOver(currentRaidInfo) {
		var priCount int32 = 0
		count, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_MAIN_REWARD)
		if err == nil && count > 0 {
			priCount = player.StaticData.GetDailyPrivilegeDrop()
		}
		subStageInfo := raid.ResetCurrentStage(currentRaidInfo, priCount, player)
		subStageInfo.HeroInfos = raid.GetActiveHeroInfos(player, raid.GetRaidFormationType(currentRaidInfo))
		subStageInfo.SelfComboSkillIds = raid.GetComboSkillIds(player, raid.GetRaidFormationType(currentRaidInfo))
		subStageInfo.PlayerLevel = player.GetLevel()
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_CYCLE_STAGE_INFO, &pb.PushCycleStageInfo{
			SubStageInfo: subStageInfo,
		})
	}
}

func CheckEnterNextSubStageHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CheckEnterNextSubStageReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHECK_ENTER_NEXT_SUB_STAGE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	errorCode := raid.CheckEnterNextSubStage(player.PlayerInstanceModel.CurrentRaidInfo, req.SubStageId, player)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHECK_ENTER_NEXT_SUB_STAGE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_CHECK_ENTER_NEXT_SUB_STAGE_RESP, &pb.CheckEnterNextSubStageResp{})
}

func EnterNextSubStageHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EnterNextSubStageReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_NEXT_SUB_STAGE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	currentRaidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	if currentRaidInfo.CurrentSubStageId == req.SubStageId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_NEXT_SUB_STAGE_RESP, pb.ERROR_CODE_ENTER_SUB_STAGE_REPEAT)
		return
	}
	resp, errorCode := raid.EnterNextSubStage(currentRaidInfo, req.SubStageId, player)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ENTER_NEXT_SUB_STAGE_RESP, errorCode)
		return
	}
	resp.SubStageInfo.HeroInfos = raid.GetActiveHeroInfos(player, raid.GetRaidFormationType(currentRaidInfo))
	resp.SubStageInfo.SelfComboSkillIds = raid.GetComboSkillIds(player, raid.GetRaidFormationType(currentRaidInfo))
	resp.SubStageInfo.PlayerLevel = player.GetLevel()
	messageSender.SendMessage(player, pb.MESSAGE_ID_ENTER_NEXT_SUB_STAGE_RESP, resp)
}

func BattlePlayerDeadHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.BattlePlayerDeadReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	currentRaidInfo := player.PlayerInstanceModel.CurrentRaidInfo
	if req.StageId != currentRaidInfo.CurrentStageId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_RESP, pb.ERROR_CODE_STAGE_HAS_SETTLE)
		return
	}
	if currentRaidInfo.IsOver {
		messageSender.SendMessage(player, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_RESP, &pb.BattlePlayerDeadResp{
			LastInstanceInfo: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
		})
		return
	}
	currentRaidInfo.IsOver = true
	errorCode, winResp := raid.OnBattlePlayerDead(player, currentRaidInfo, player.PlayerInstanceModel)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_RESP, errorCode)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_BATTLE_PLAYER_DEAD_RESP, &pb.BattlePlayerDeadResp{
		LastInstanceInfo: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_STAGE_BATTLE_WIN, winResp)
}

func playerLogoutHandle(task serviceInterface.InnerTaskInterface) (any, error) {
	p := sessionManager.GetPlayerBasicInfoByUserId(task.GetReqId())
	if p == nil {
		return nil, nil
	}
	playerModel := p.(*model.PlayerModel)
	gamePlatform.RemovePlayerFromGame(playerModel)
	return nil, nil
}

func BattleNotifyGameEnterSubStageHandler(message proto.Message, player *model.PlayerModel) {

}

func BattleNotifyGameBattleResultHandler(message proto.Message, player *model.PlayerModel) {

}

func BattleNotifyGameHeroDeadHandler(message proto.Message, player *model.PlayerModel) {

}

func BattleNotifyGameKillMonsterHandler(message proto.Message, player *model.PlayerModel) {

}

func ConfirmBattleMessageHandler(message proto.Message, player *model.PlayerModel) {

}
