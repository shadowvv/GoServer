package task

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/eventService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
)

func PlayerLoginLoadTask(player *model.PlayerModel) []*pb.TaskInfoStruct {
	res := make([]*pb.TaskInfoStruct, 0)

	for attr, taskTypeList := range player.TaskModel.TaskEntity {
		if attr == enum.TaskAffiliationBounty || attr == enum.TaskAffiliationPassCard || attr == enum.TaskAffiliationTrial || attr == enum.TaskAffiliationCityAge || attr == enum.TaskAffiliationAct {
			continue
		}
		taskInfo := &pb.TaskInfoStruct{
			Attribution:  attr,
			TaskInfoList: make([]*pb.TaskInfo, 0),
		}
		for _, idList := range taskTypeList {
			for _, entity := range idList {

				taskInfo.TaskInfoList = append(taskInfo.TaskInfoList, &pb.TaskInfo{
					TaskId:    entity.TaskID,
					TaskState: entity.Status,
					Progress:  entity.ProgressData,
				})

			}
		}
		if attr == enum.TaskAffiliationDaily {
			taskInfo.DailyPoint = player.TaskActiveRewardModel.Entity.DailyPoint
			taskInfo.DailyBox = player.TaskActiveRewardModel.Entity.DailyBox
			taskInfo.WeekBox = player.TaskActiveRewardModel.Entity.WeekBox
			taskInfo.WeekPoint = player.TaskActiveRewardModel.Entity.WeekPoint
		}
		res = append(res, taskInfo)
	}
	return res
}

func PlayerLoginLoadBounty(player *model.PlayerModel) []*pb.BountyInfo {
	res := make([]*pb.BountyInfo, 0)

	for id, v := range player.BountyModel.Entities {
		if v.Status > enum.TaskStatusFinishUnReward {
			continue
		}
		if v.EndTime < tool.UnixNowMilli() {
			continue
		}
		bountyInfo := &pb.BountyInfo{
			BountyId: id,
			EndTime:  v.EndTime,
		}
		taskList := make([]*pb.TaskInfo, 0)
		for _, slotId := range v.SlotList {
			taskDetail := player.TaskModel.TaskEntityBySlot[slotId]
			if taskDetail == nil {
				logger.ErrorWithZapFields("bounty task entity nil", zap.Int32("bounty id", id), zap.Int32("slot id", slotId))
				continue
			}
			taskInfo := &pb.TaskInfo{
				TaskId:    taskDetail.TaskID,
				TaskState: taskDetail.Status,
				Progress:  taskDetail.ProgressData,
			}
			taskList = append(taskList, taskInfo)
		}
		bountyInfo.TaskList = taskList
		bountyInfo.Status = v.Status
		res = append(res, bountyInfo)
	}
	return res
}

func PlayerLoginLoadPassTask(player *model.PlayerModel) []*pb.PassCardTask {
	res := make([]*pb.PassCardTask, 0)

	for _, v := range player.PassTaskModel.Entities {
		passTask := &pb.PassCardTask{
			PassCardId:           v.PassCardId,
			PassCardTaskInfoList: make([]*pb.PassCardtTaskInfo, 0),
		}
		for id, value := range v.TaskSlotList {
			taskEntity := player.TaskModel.TaskEntityBySlot[value]
			if taskEntity == nil {
				logger.ErrorWithZapFields("pass task entity nil", zap.Int32("pass card id", v.PassCardId), zap.Int32("task slot id", value))
				continue
			}
			taskInfo := &pb.PassCardtTaskInfo{
				TaskId:    taskEntity.TaskID,
				Progress:  taskEntity.ProgressData,
				FinishNum: v.TaskFinishCount[id],
			}
			passTask.PassCardTaskInfoList = append(passTask.PassCardTaskInfoList, taskInfo)
		}
	}
	return res
}

func TaskFinishCheck(player *model.PlayerModel, taskAttribution int32, taskType int32, id int32) {
	task := player.TaskModel
	entity := task.TaskEntity[taskAttribution][taskType][id]
	taskId, err := gameConfig.GetCoreTaskId(taskAttribution, id)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", taskAttribution), zap.Int32("task id", id))
		return
	}
	if entity.ProgressData >= gameConfig.GetCoreCfg(taskId).TaskNum {
		task.UpdateTaskStatus(id, taskType, taskAttribution, enum.TaskStatusFinishUnReward)
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
			Attribution: taskAttribution,
			TaskId:      id,
			TaskState:   entity.Status,
			Progress:    entity.ProgressData,
		})
	} else if taskAttribution == enum.TaskAffiliationMain || taskAttribution == enum.TaskAffiliationCityAge {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
			Attribution: taskAttribution,
			TaskId:      id,
			TaskState:   entity.Status,
			Progress:    entity.ProgressData,
		})
	}
}

func KillAnyMonsterHowMany(event *eventService.KillMonsterEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		num := int32(0)
		for _, v := range event.MonsterList {
			if gameConfig.GetCoreCfg(taskId).TaskPara[0] == 0 || v.MonsterType == gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				num += v.Count
			}
		}
		if num > 0 {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+num)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}

func GetWhatItemHowMany(events *eventService.ItemCollectEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		num := int32(0)
		for _, event := range events.ItemInfoList {
			if event.ItemId == gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				num += int32(event.Count)
			}
		}
		if num > 0 {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+num)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}
func KillWhatMonsterHowMany(event *eventService.KillMonsterEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		num := int32(0)
		for _, v := range event.MonsterList {
			if gameConfig.GetCoreCfg(taskId).TaskPara[0] == 0 || v.MonsterType == gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				num += v.Count
			}
		}
		if num > 0 {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+num)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}

func WhereKillWhatMonsterHowMany(event *eventService.KillMonsterEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		num := int32(0)
		for _, v := range event.MonsterList {
			if event.SceneId == gameConfig.GetCoreCfg(taskId).TaskPara[0] && v.MonsterId == gameConfig.GetCoreCfg(taskId).TaskPara[1] {
				num += v.Count
			}
		}
		if num > 0 {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+num)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}
func GetTypeOrQualityItemsHowMany(events *eventService.ItemCollectEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		num := int32(0)
		for _, event := range events.ItemInfoList {
			if event.ItemType == gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				if gameConfig.GetCoreCfg(taskId).TaskPara[1] == 0 || event.ItemQuality == gameConfig.GetCoreCfg(taskId).TaskPara[1] {
					num += int32(event.Count)
				}
			}
		}
		if num > 0 {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+num)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}
func AnyHeroReachWhatLevel(event *eventService.HeroLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		if event.NewLevel >= gameConfig.GetCoreCfg(taskId).TaskPara[0] {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, gameConfig.GetCoreCfg(taskId).TaskNum)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			task.UpdateTaskStatus(id, taskType, taskAttribution, enum.TaskStatusFinishUnReward)
			messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
				Attribution: taskAttribution,
				TaskId:      id,
				TaskState:   entity.Status,
				Progress:    entity.ProgressData,
			})
		}
	}
}

func AnyHeroLevelUpHowMany(event *eventService.HeroLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		levelUpCount := event.NewLevel - event.OldLevel
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+levelUpCount)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func PassWhatMainLevel(event *eventService.PassInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		if event.InstanceTypeId == enum.MAIN_INSTANCE_ID {
			if event.InstanceId > gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				task.UpdateTaskProgressData(id, taskType, taskAttribution, gameConfig.GetCoreCfg(taskId).TaskNum)
				task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
				task.UpdateTaskStatus(id, taskType, taskAttribution, enum.TaskStatusFinishUnReward)
				messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
					Attribution: taskAttribution,
					TaskId:      id,
					TaskState:   entity.Status,
					Progress:    entity.ProgressData,
				})
			}
		}
	}
}
func PassHowManyMainLevel(event *eventService.PassInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.MAIN_INSTANCE_ID {
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}

func AccessoryLuckyHowMany(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "accessory" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func HeroLotteryHowMany(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != string(enum.LOTTERY_TYPE_HERO) {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func LoopBoxLotteryHowMany(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "loopBox" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func AccessorySystemLevelReachWhat(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "accessory" {
		return
	}
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HeroStarUpHowMany(event *eventService.HeroStarUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			heroId := event.HeroId
			heroBaseCfg := gameConfig.GetHeroBaseCfg(heroId)
			if heroBaseCfg == nil {
				logger.ErrorWithZapFields("get hero base cfg error", zap.Int32("hero id", heroId))
				return
			}
			if heroBaseCfg.HeroStar >= event.StarLevel {
				continue
			}
			if entity.Status != 0 {
				continue
			}
			task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, id)
		}
	}
}

func HowManyHeroReachWhatStar(event *eventService.HeroStarUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HowManyHeroReachWhatLevel(event *eventService.HeroLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func LoopBoxSystemLevelReachWhat(event *eventService.LoopBoxLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HowManyHeroReachWhatPotential(event *eventService.AddHeroAlbumEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func StoneClassTotalLevelReachWhat(event *eventService.StoneAttrLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HowManyEquipStrongReachWhatLevel(event *eventService.EquipmentStrongEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HowManyPetReachWhatLevel(event *eventService.PetLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func JoinAlliance(event *eventService.AllianceJoinEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func HowManyPetReachWhatStar(event *eventService.PetStarUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func WearHowManyEquipmentQuality(event *eventService.EquipmentWearEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func ArenaScoreReachWhat(event *eventService.ArenaScoreChangeEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func MainTaskPassWhatNum(event *eventService.MainTaskChangeEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}

func AdventureParticipateHowMany(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.ADVENTURE_INSTANCE_ID {
			task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
		}
	}
}

func GloryArenaChallengeHowMany(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.GLORY_ARENA_INSTANCE_ID {
			task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
		}
	}
}

func AdChestOpenHowMany(event *eventService.AdChestOpenEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+event.OpenCount)
		task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
	}
}

func EquipmentForgeHowMany(event *eventService.EquipmentForgeEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+event.ForgeCount)
		task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
	}
}

func TowerChallengeHowMany(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID {
			task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
		}
	}
}

func ArenaParticipateHowMany(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.ARENA_INSTANCE_ID {
			task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
		}
	}
}

func QuickClaimMachineRewardHowMany(event *eventService.QuickClaimMachineRewardEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func BuildLevelUpHowMany(event *eventService.BuildLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func TowerChallengePassWhatLevel(event *eventService.PassInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		if event.InstanceTypeId == enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID {
			if event.InstanceId >= gameConfig.GetCoreCfg(taskId).TaskPara[0] {
				task.UpdateTaskProgressData(id, taskType, taskAttribution, gameConfig.GetCoreCfg(taskId).TaskNum)
				task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
				task.UpdateTaskStatus(id, taskType, taskAttribution, enum.TaskStatusFinishUnReward)
				messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
					Attribution: taskAttribution,
					TaskId:      id,
					TaskState:   entity.Status,
					Progress:    entity.ProgressData,
				})
			}
		}
	}
}

func WhatBuildLevelUpWhat(event *eventService.BuildLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
			return
		}
		if cfg := gameConfig.GetCoreCfg(taskId); cfg != nil {
			if int32(event.BuildId) == cfg.TaskPara[0] {
				task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
				task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
				TaskFinishCheck(player, taskAttribution, taskType, id)
			}
		}
	}
}

func DispatchKillMonsterHowMany(event *eventService.DispatchKillMonsterEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func CumulativeDispatchKillMonsterHowMany(event *eventService.DispatchKillMonsterEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func PlayerPowerReachWhat(event *eventService.PlayerPowerChangeEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, int32(event.Power))
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func AllBuildLevelReachWhat(event *eventService.BuildLevelUpEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func HeroQuantityReachWhat(event *eventService.AddHeroAlbumEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func CollectionLotteryHowMany(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "collect" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func CollectionLotteryHowManyCumulative(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "collect" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func PetRecruitHowMany(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "pet" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func PetRecruitHowManyCumulative(event *eventService.LuckyLotteryEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	if event.LotteryType != "pet" {
		return
	}
	for id, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(id, taskType, taskAttribution, entity.ProgressData+event.LotteryNum)
		task.UpdateUpdateTime(id, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, id)
	}
}

func DungeonParticipateHowMany(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if cfg := gameConfig.GetCoreCfg(entity.TaskID); cfg != nil {
			if int32(event.InstanceTypeId) == cfg.TaskPara[0] {
				task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
				task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
				TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
			}
		}
	}
}

func DungeonParticipateHowManyCumulative(event *eventService.JoinInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if cfg := gameConfig.GetCoreCfg(entity.TaskID); cfg != nil {
			if int32(event.InstanceTypeId) == cfg.TaskPara[0] {
				task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
				task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
				TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
			}
		}
	}
}
func DungeonPassWhatStage(event *eventService.PassInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if cfg := gameConfig.GetCoreCfg(entity.TaskID); cfg != nil {
			if int32(event.InstanceTypeId) == cfg.TaskPara[0] && cfg.TaskPara[1] == event.InstanceId {
				task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, cfg.TaskNum)
				task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
				TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
			}
		}
	}
}

func GloryArenaWinHowMany(event *eventService.PassInstanceEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		if event.InstanceTypeId == enum.GLORY_ARENA_INSTANCE_ID {
			task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
			task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
			TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
		}
	}
}

func StrongEquipmentHowMany(event *eventService.EquipmentStrongEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status != enum.TaskStatusUnFinish {
			continue
		}
		task.UpdateTaskProgressData(entity.TaskID, taskType, taskAttribution, entity.ProgressData+1)
		task.UpdateUpdateTime(entity.TaskID, taskType, taskAttribution, tool.UnixNowMilli())
		TaskFinishCheck(player, taskAttribution, taskType, entity.TaskID)
	}
}

func WearHowManyEquipmentLevel(event *eventService.EquipmentWearEvent, player *model.PlayerModel, taskAttribution int32, taskType int32) {
	task := player.TaskModel
	for _, entity := range task.TaskEntity[taskAttribution][taskType] {
		if entity.Status == enum.TaskStatusUnFinish {
			task.NeedCheckTaskList = append(task.NeedCheckTaskList, entity)
		}
	}
}
