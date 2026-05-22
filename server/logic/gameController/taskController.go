package gameController

import (
	"fmt"

	"github.com/drop/GoServer/server/service/serviceInterface"

	"github.com/drop/GoServer/server/logic/platform/eventService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/task"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("task", &TaskController{})
}

var _ LogicControllerInterface = (*TaskController)(nil)

type TaskController struct {
}

func (l *TaskController) RegisterLogicMessage() {
	// 注册任务相关的消息
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TASK_DETAILS_REQ, &pb.TaskDetailsReq{}, TaskDetailsReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TASK_REWARD_REQ, &pb.TaskRewardReq{}, TaskRewardReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TASK_DAILY_REWARD_REQ, &pb.TaskDailyRewardReq{}, TaskDailyRewardReqHandle, enum.FUNCTION_ID_DAILY_QUEST)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BOUNTY_REWARD_REQ, &pb.BountyRewardReq{}, BountyRewardHandle, enum.FUNCTION_ID_BOUNTY)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_BOUNTY_DETAIL_REQ, &pb.GetBountyDetailReq{}, GetBountyDetailHandle, enum.FUNCTION_ID_BOUNTY)

	//  注册任务相关的内部消息处理函数
	RegisterPlayerInnerTask(enum.INNER_MSG_EVENT_TASK_PLAYER, InnerPlayerTaskHandle)
}

func TaskDetailsReqHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Task Details Req", player)

	req, ok := message.(*pb.TaskDetailsReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DETAILS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	taskAttributionId := req.Attribution
	if !player.TaskModel.CheckSystemUnlock(taskAttributionId) {
		platformLogger.InfoWithUser("task system not unlock", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DETAILS_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	var taskInfoList []*pb.TaskInfo
	for _, taskIdList := range player.TaskModel.TaskEntity[taskAttributionId] {
		for taskId, entity := range taskIdList {
			taskDetail := &pb.TaskInfo{
				TaskId:    taskId,
				Progress:  entity.ProgressData,
				TaskState: entity.Status,
			}
			taskInfoList = append(taskInfoList, taskDetail)
		}
	}
	resp := &pb.TaskDetailsResp{
		TaskInfoList: taskInfoList,
	}
	if req.Attribution == enum.TaskAffiliationDaily {
		resp.DailyBox = player.TaskActiveRewardModel.Entity.DailyBox
		resp.DailyPoint = player.TaskActiveRewardModel.Entity.DailyPoint
		resp.WeekBox = player.TaskActiveRewardModel.Entity.WeekBox
		resp.WeekPoint = player.TaskActiveRewardModel.Entity.WeekPoint
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TASK_DETAILS_RESP, resp)
}

func TaskRewardReqHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Task Reward Req", player)

	req, ok := message.(*pb.TaskRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// RewardType: 1=一键领取(主线+支线所有可领取), 2=单个任务领取
	if req.RewardType == 1 {
		// 一键领取
		rewards, pushUpdates, err := player.TaskModel.BatchRewardTasks()
		if err != nil {
			platformLogger.ErrorWithUser(err.Error(), player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
			return
		}
		for _, push := range pushUpdates {
			messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
			if push.Attribution == enum.TaskAffiliationMain {
				eventBusService.SubmitMainTaskChangeEvent(player.GetUserId())
			}
		}
		messageSender.SendMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, &pb.TaskRewardResp{
			ItemList: rewards,
		})
		return
	}

	// 单个任务领取
	rewards, pushUpdate, err := player.TaskModel.RewardSingleTask(req.Attribution, req.TaskId)
	if err != nil {
		platformLogger.ErrorWithUser(err.Error(), player, nil)
		switch err.Error() {
		case "task system not unlock":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		case "task core id error", "task core is nil":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		case "task not exist":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FOUND)
		case "task not finish":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
		default:
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
		}
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TASK_REWARD_RESP, &pb.TaskRewardResp{
		ItemList: rewards,
	})
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, pushUpdate)
	if req.Attribution == enum.TaskAffiliationMain {
		eventBusService.SubmitMainTaskChangeEvent(player.GetUserId())
	}
}

func TaskDailyRewardReqHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Task Daily Reward Req", player)

	req, ok := message.(*pb.TaskDailyRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DAILY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	res := make([]*pb.ItemBasicInfo, 0)
	allRewardItems := make([]*gameConfig.ItemConfig, 0)
	allAddRewardItems := make([]*gameConfig.ItemConfig, 0)
	if req.RewardType == 1 {
		// 一键领取全部
		rewardItemList, err := player.TaskModel.RewardDaily(player.TaskActiveRewardModel)
		if err != nil {
			platformLogger.InfoWithUser("Task Daily cfg is null", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DAILY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		allRewardItems = append(allRewardItems, rewardItemList...)

		rewardItemList = player.TaskActiveRewardModel.Reward()
		allRewardItems = append(allRewardItems, rewardItemList...)
	} else if req.RewardType == 2 {
		rewardItemList := player.TaskActiveRewardModel.Reward()
		allRewardItems = append(allRewardItems, rewardItemList...)

	} else {
		rewardItemList, err := player.TaskModel.RewardDaily(player.TaskActiveRewardModel)
		if err != nil {
			platformLogger.InfoWithUser("Task Daily cfg is null", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DAILY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		allRewardItems = append(allRewardItems, rewardItemList...)
	}
	for _, v := range allRewardItems {
		itemCfg := gameConfig.GetItemCfg(v.ID)
		if itemCfg == nil {
			continue
		}
		res = append(res, &pb.ItemBasicInfo{
			ItemId: v.ID,
			Count:  v.Num,
		})
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_CURRENCY) && itemCfg.IsAutoUse == 1 {
			continue
		}
		allAddRewardItems = append(allAddRewardItems, v)
	}

	err := itemService.AddItems(player, allAddRewardItems, enum.ITEM_CHANGE_REASON_DAILY_TASK)
	if err != nil {
		platformLogger.ErrorWithUser("add item failed", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TASK_DAILY_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_TASK_DAILY_REWARD_RESP, &pb.TaskDailyRewardResp{
		ItemList:   res,
		DailyBox:   player.TaskActiveRewardModel.Entity.DailyBox,
		DailyPoint: player.TaskActiveRewardModel.Entity.DailyPoint,
		WeekBox:    player.TaskActiveRewardModel.Entity.WeekBox,
		WeekPoint:  player.TaskActiveRewardModel.Entity.WeekPoint,
	})
}

func BountyRewardHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("bounty Reward Req", player)

	req, ok := message.(*pb.BountyRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	bountyId := req.BountyId
	taskId := req.TaskId
	detail := player.BountyModel.Entities[bountyId]

	res := make([]*pb.ItemBasicInfo, 0)
	allRewardItems := make([]*gameConfig.ItemConfig, 0)

	if taskId == 0 {
		for _, v := range detail.SlotList {
			taskId = player.TaskModel.TaskEntityBySlot[v].TaskID
			taskCoreId, err := gameConfig.GetCoreTaskId(enum.TaskAffiliationBounty, taskId)
			if err != nil {
				platformLogger.InfoWithUser("Task core id is error", player)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
				return
			}
			taskCore := gameConfig.GetCoreCfg(taskCoreId)
			if taskCore == nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
				return
			}
			taskType := taskCore.TaskType
			taskDetail := player.TaskModel.TaskEntity[enum.TaskAffiliationBounty][taskType][taskId]

			bountyTask := gameConfig.GetBountyTaskCfg(taskId)
			if bountyTask == nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
				return
			}
			if taskDetail.Status == enum.TaskStatusUnFinish {
				platformLogger.ErrorWithUser("task not finish", player, nil)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
				return
			}
			if taskDetail.Status == enum.TaskStatusFinishUnReward {
				allRewardItems = append(allRewardItems, bountyTask.TaskReward...)
				player.TaskModel.UpdateTaskStatus(taskId, taskType, enum.TaskAffiliationBounty, enum.TaskStatusFinishAndReward)
			}
		}
		if player.BountyModel.Entities[bountyId] != nil {
			if player.BountyModel.Entities[bountyId].Status == enum.TaskStatusUnFinish || player.BountyModel.Entities[bountyId].Status == enum.TaskStatusFinishUnReward {
				allRewardItems = append(allRewardItems, gameConfig.GetBountyBaseCfg(bountyId).Drop...)
				player.BountyModel.UpdateStatus(bountyId, enum.TaskStatusFinishAndReward)
				*player.BountyModel.CanUseBountySlotList = append(*player.BountyModel.CanUseBountySlotList, bountyId)
			} else {
				platformLogger.ErrorWithUser("task not finish or rewarded", player, nil)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
				return
			}
		}
	} else {
		taskCoreId, err := gameConfig.GetCoreTaskId(enum.TaskAffiliationBounty, taskId)
		if err != nil {
			platformLogger.InfoWithUser("Task core id is error", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		taskType := gameConfig.GetCoreCfg(taskCoreId).TaskType
		taskDetail := player.TaskModel.TaskEntity[enum.TaskAffiliationBounty][taskType][taskId]
		if taskDetail.Status == enum.TaskStatusFinishUnReward {
			allRewardItems = append(allRewardItems, gameConfig.GetBountyTaskCfg(taskId).TaskReward...)
			player.TaskModel.UpdateTaskStatus(taskId, taskType, enum.TaskAffiliationBounty, enum.TaskStatusFinishAndReward)
		} else {
			platformLogger.ErrorWithUser("task not finish or rewarded", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
			return
		}
	}

	err := itemService.AddItems(player, allRewardItems, enum.ITEM_CHANGE_REASON_BOUNTY_TASK)
	if err != nil {
		platformLogger.ErrorWithUser("add item failed", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
	}

	for _, v := range allRewardItems {
		res = append(res, &pb.ItemBasicInfo{
			ItemId: v.ID,
			Count:  v.Num,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_BOUNTY_REWARD_RESP, &pb.BountyRewardResp{
		ItemList: res,
	})
}

func GetBountyDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("bounty detail Req", player)

	req, ok := message.(*pb.GetBountyDetailReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_BOUNTY_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	bountyId := req.BountyId

	entity := player.BountyModel.Entities[bountyId]

	taskInfo := make([]*pb.TaskInfo, 0)

	for _, v := range entity.SlotList {
		taskDetail := player.TaskModel.TaskEntityBySlot[v]
		taskInfo = append(taskInfo, &pb.TaskInfo{
			TaskId:    taskDetail.TaskID,
			TaskState: taskDetail.Status,
			Progress:  taskDetail.ProgressData,
		})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_BOUNTY_DETAIL_RESP, &pb.GetBountyDetailResp{
		Info: &pb.BountyInfo{
			BountyId: bountyId,
			TaskList: taskInfo,
			EndTime:  entity.EndTime,
		},
	})
}

func InnerPlayerTaskHandle(messageTask serviceInterface.InnerTaskInterface) (any, error) {
	innerTask, ok := messageTask.(*dispatcherService.InnerTask)
	if !ok {
		return nil, fmt.Errorf("invalid task type")
	}

	req, ok := innerTask.ReqParameter.(eventService.GameEvent)

	playerId := req.GetObjectID()
	eventType := req.GetEventType()

	p := sessionManager.GetPlayerBasicInfoByUserId(playerId)
	if p == nil {
		return nil, fmt.Errorf("player not exist")
	}
	player := p.(*model.PlayerModel)

	var err error = nil
	for taskAttributionId, taskAttributionMap := range player.TaskModel.TaskEntity {
		for _, taskType := range enum.EventToObjectiveTypes[eventType] {
			if _, ok := taskAttributionMap[taskType]; ok {
				switch taskType {
				case enum.ObjectiveTypeAnyHeroReachWhatLevel:
					task.AnyHeroReachWhatLevel(req.(*eventService.HeroLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAnyHeroLevelUpHowMany:
					task.AnyHeroLevelUpHowMany(req.(*eventService.HeroLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeKillWhatMonsterHowMany:
					task.KillWhatMonsterHowMany(req.(*eventService.KillMonsterEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeKillAnyMonsterHowMany:
					task.KillAnyMonsterHowMany(req.(*eventService.KillMonsterEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeWhereKillWhatMonsterHowMany:
					task.WhereKillWhatMonsterHowMany(req.(*eventService.KillMonsterEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypePassWhatMainLevel:
					task.PassWhatMainLevel(req.(*eventService.PassInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypePassHowManyMainLevel:
					task.PassHowManyMainLevel(req.(*eventService.PassInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeGetTypeOrQualityItemsHowMany:
					task.GetTypeOrQualityItemsHowMany(req.(*eventService.ItemCollectEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeGetWhatItemsHowMany:
					task.GetWhatItemHowMany(req.(*eventService.ItemCollectEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAccessoryLuckyHowMany:
					task.AccessoryLuckyHowMany(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHeroLotteryHowMany:
					task.HeroLotteryHowMany(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeLoopBoxLotteryHowMany:
					task.LoopBoxLotteryHowMany(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAccessorySystemLevelReachWhat:
					task.AccessorySystemLevelReachWhat(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHeroStarUpHowMany:
					task.HeroStarUpHowMany(req.(*eventService.HeroStarUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyHeroReachWhatStar:
					task.HowManyHeroReachWhatStar(req.(*eventService.HeroStarUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyHeroReachWhatLevel:
					task.HowManyHeroReachWhatLevel(req.(*eventService.HeroLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeTowerChallengeHowMany:
					task.TowerChallengeHowMany(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeArenaParticipateHowMany:
					task.ArenaParticipateHowMany(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeQuickClaimMachineRewardHowMany:
					task.QuickClaimMachineRewardHowMany(req.(*eventService.QuickClaimMachineRewardEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeBuildLevelUpHowMany:
					task.BuildLevelUpHowMany(req.(*eventService.BuildLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeTowerChallengePassWhatLevel:
					task.TowerChallengePassWhatLevel(req.(*eventService.PassInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeWhatBuildLevelUpWhat:
					task.WhatBuildLevelUpWhat(req.(*eventService.BuildLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeDispatchKillMonsterHowMany:
					task.DispatchKillMonsterHowMany(req.(*eventService.DispatchKillMonsterEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeCumulativeDispatchKillMonsterHowMany:
					task.CumulativeDispatchKillMonsterHowMany(req.(*eventService.DispatchKillMonsterEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypePlayerPowerReachWhat:
					task.PlayerPowerReachWhat(req.(*eventService.PlayerPowerChangeEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAllBuildLevelReachWhat:
					task.AllBuildLevelReachWhat(req.(*eventService.BuildLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHeroQuantityReachWhat:
					task.HeroQuantityReachWhat(req.(*eventService.AddHeroAlbumEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyHeroReachWhatPotential:
					task.HowManyHeroReachWhatPotential(req.(*eventService.AddHeroAlbumEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeLoopBoxSystemLevelReachWhat:
					task.LoopBoxSystemLevelReachWhat(req.(*eventService.LoopBoxLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyEquipStrongReachWhatLevel:
					task.HowManyEquipStrongReachWhatLevel(req.(*eventService.EquipmentStrongEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyPetReachWhatLevel:
					task.HowManyPetReachWhatLevel(req.(*eventService.PetLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeJoinAlliance:
					task.JoinAlliance(req.(*eventService.AllianceJoinEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAdventureParticipateHowMany:
					task.AdventureParticipateHowMany(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeGloryArenaChallengeHowMany:
					task.GloryArenaChallengeHowMany(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeHowManyPetReachWhatStar:
					task.HowManyPetReachWhatStar(req.(*eventService.PetStarUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeEquipmentForgeHowMany:
					task.EquipmentForgeHowMany(req.(*eventService.EquipmentForgeEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeWearHowManyEquipmentQuality:
					task.WearHowManyEquipmentQuality(req.(*eventService.EquipmentWearEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeArenaScoreReachWhat:
					task.ArenaScoreReachWhat(req.(*eventService.ArenaScoreChangeEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeAdChestOpenHowMany:
					task.AdChestOpenHowMany(req.(*eventService.AdChestOpenEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeMainTaskPassWhatNum:
					task.MainTaskPassWhatNum(req.(*eventService.MainTaskChangeEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeStoneClassTotalLevelReachWhat:
					task.StoneClassTotalLevelReachWhat(req.(*eventService.StoneAttrLevelUpEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeCollectionLotteryHowMany:
					task.CollectionLotteryHowMany(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeCollectionLotteryHowManyCumulative:
					task.CollectionLotteryHowManyCumulative(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypePetRecruitHowMany:
					task.PetRecruitHowMany(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypePetRecruitHowManyCumulative:
					task.PetRecruitHowManyCumulative(req.(*eventService.LuckyLotteryEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeDungeonParticipateHowMany:
					task.DungeonParticipateHowMany(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeDungeonParticipateHowManyCumulative:
					task.DungeonParticipateHowManyCumulative(req.(*eventService.JoinInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeDungeonPassWhatStage:
					task.DungeonPassWhatStage(req.(*eventService.PassInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeGloryArenaWinHowMany:
					task.GloryArenaWinHowMany(req.(*eventService.PassInstanceEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeStrongEquipmentHowMany:
					task.StrongEquipmentHowMany(req.(*eventService.EquipmentStrongEvent), player, taskAttributionId, taskType)
				case enum.ObjectiveTypeWearHowManyEquipmentLevel:
					task.WearHowManyEquipmentLevel(req.(*eventService.EquipmentWearEvent), player, taskAttributionId, taskType)
				}
			}
		}
	}
	return true, err
}
