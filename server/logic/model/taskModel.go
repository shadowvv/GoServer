package model

import (
	"errors"
	"math/rand"

	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type TaskEntity struct {
	UserID          int64 `gorm:"column:user_id;primaryKey"`
	SlotId          int32 `gorm:"column:slot_id;primaryKey" json:"slot_id"`
	TaskID          int32 `gorm:"column:task_id;" json:"task_id"`
	TaskAttribution int32 `gorm:"column:task_attribution;primaryKey" json:"task_attribution"`
	ProgressData    int32 `gorm:"column:progress_data;" json:"progress_data"`
	Status          int32 `gorm:"column:status;" json:"status"` // 0进行中,1=完成,2=已领奖
	AddTime         int64 `gorm:"column:add_time;" json:"add_time"`
	UpdateTime      int64 `gorm:"column:update_time;" json:"update_time"`
}

func (t *TaskEntity) TableName() string {
	return "task"
}

func NewTaskEntity(userID int64, slotId, taskID int32, taskAttribution int32, progressData int32, status int32, addTime int64, updateTime int64) *TaskEntity {

	return &TaskEntity{
		UserID:          userID,
		SlotId:          slotId,
		TaskID:          taskID,
		TaskAttribution: taskAttribution,
		ProgressData:    progressData,
		Status:          status,
		AddTime:         addTime,
		UpdateTime:      updateTime,
	}
}

var _ logicCommon.PlayerModelInterface = (*TaskModel)(nil)

type TaskModel struct {
	UserId            int64
	TaskEntity        map[int32]map[int32]map[int32]*TaskEntity  //taskAttribution taskType taskID
	Changed           map[int32]map[int32]map[string]interface{} // taskAttribution -> slotId -> changes
	TaskEntityBySlot  map[int32]*TaskEntity
	NeedCheckTaskList []*TaskEntity
	player            *PlayerModel
}

func (t *TaskEntity) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("user_id", t.UserID)
	enc.AddInt32("task_id", t.TaskID)
	enc.AddInt32("task_attribution", t.TaskAttribution)
	enc.AddInt32("progress_data", t.ProgressData)
	enc.AddInt32("status", t.Status)
	enc.AddInt64("add_time", t.AddTime)
	enc.AddInt64("update_time", t.UpdateTime)
	return nil
}

func (t *TaskModel) CheckSystemUnlock(attribution int32) bool {
	flag := true

	if attribution == enum.TaskAffiliationMain {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_MAIN_QUEST, t.player)
	} else if attribution == enum.TaskAffiliationSide {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_SUB_QUEST, t.player)
	} else if attribution == enum.TaskAffiliationDaily {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_DAILY_QUEST, t.player)
	} else if attribution == enum.TaskAffiliationBounty {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_BOUNTY, t.player)
	} else if attribution == enum.TaskAffiliationTrial {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_TRIAL, t.player)
	} else if attribution == enum.TaskAffiliationCityAge {
		flag = unlockService.CheckSystemUnlock(enum.FUNCTION_ID_CITYAGE, t.player)
	}

	return flag
}

func (t *TaskModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if unlockService.CheckSystemUnlock(enum.FUNCTION_ID_MAIN_QUEST, t.player) {
		if t.TaskEntity[enum.TaskAffiliationMain] == nil || len(t.TaskEntity[enum.TaskAffiliationMain]) == 0 {
			mainTaskCfg := gameConfig.GetMainCfg(20001)
			if mainTaskCfg != nil {
				err := t.AddTask(NewTaskEntity(t.UserId, mainTaskCfg.TaskAttribution*10000+1, mainTaskCfg.Id, mainTaskCfg.TaskAttribution, 0, enum.TaskStatusUnFinish, tool.UnixNowMilli(), 0))
				if err != nil {
					logger.ErrorWithZapFields("add main task error")
				}
			}
		}
	}
	if len(t.NeedCheckTaskList) > 0 {
		for i := 0; i < len(t.NeedCheckTaskList); i++ {
			taskEntity := t.NeedCheckTaskList[i]
			taskCoreId, err := gameConfig.GetCoreTaskId(taskEntity.TaskAttribution, taskEntity.TaskID)
			if err != nil {
				logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", taskEntity.TaskAttribution), zap.Int32("task id", taskEntity.TaskID))
				continue
			}
			taskCfg := gameConfig.GetCoreCfg(taskCoreId)
			if taskCfg == nil {
				continue
			}
			if enum.NeedCheckTask[taskCfg.TaskType] {
				t.CheckTaskProgressData(taskEntity)
			}
		}
		t.NeedCheckTaskList = make([]*TaskEntity, 0)
	}

	// 收集需要标记为待删除的旧任务和需要添加的新任务
	var toAdd []*TaskEntity    // 需要添加的新任务
	var toDelete []*TaskEntity // 需要删除的旧任务

	for attributionId, attributionMap := range t.TaskEntity {
		for taskTypeId, taskMap := range attributionMap {
			for taskId, entity := range taskMap {
				if taskTypeId == enum.ObjectiveTypeWhatDispatchMapUnlockWhatStage {
					t.NeedCheckTaskList = append(t.NeedCheckTaskList, entity)
				}
				if entity == nil {
					continue
				}

				// 已经在待删除状态的
				if entity.Status == enum.TaskStatusDeleteInMemory {
					toDelete = append(toDelete, entity)
					continue
				}

				if attributionId == enum.TaskAffiliationBounty || attributionId == enum.TaskAffiliationPassCard {
					continue
				}
				// 检查是否需要生成新任务
				if entity.Status == enum.TaskStatusFinishAndReward && attributionId != enum.TaskAffiliationDaily {

					if next := t.CheckNewTask(attributionId, taskTypeId, taskId); next != nil {
						logger.InfoWithZapFields("TaskModel mark deleted in memory", zap.Object("task", entity))
						// 收集操作，不立即执行（避免破坏遍历）
						entity.Status = enum.TaskStatusDeleteInMemory
						toAdd = append(toAdd, next)

						// 在Changed中用旧任务的key存储新任务的数据
						t.UpdateChangedWithNewData(entity, next)
					}
				}
			}
		}
	}

	// 检查是否有新的支线任务组需要添加
	if unlockService.CheckSystemUnlock(enum.FUNCTION_ID_SUB_QUEST, t.player) {
		t.CheckNewSideTaskGroup(&toAdd)
	}

	if unlockService.CheckSystemUnlock(enum.FUNCTION_ID_DAILY_QUEST, t.player) {
		if t.TaskEntity[enum.TaskAffiliationDaily] == nil || len(t.TaskEntity[enum.TaskAffiliationDaily]) == 0 {
			t.RefreshDailyTasks()
		}
	}
	// 检查每日任务是否需要跨天刷新（只需检查一个entity的时间即可）
	for _, taskIdMap := range t.TaskEntity[enum.TaskAffiliationDaily] {
		for _, entity := range taskIdMap {
			if !tool.IsSameDay(tool.MilliToTime(entity.UpdateTime), tool.MilliToTime(tool.UnixNowMilli())) {
				// 清空TaskMap中旧的索引（按旧TaskID），SlotMap由RefreshDailyTasks覆盖
				t.TaskEntity[enum.TaskAffiliationDaily] = make(map[int32]map[int32]*TaskEntity)
				t.RefreshDailyTasks()
			}
			break
		}
		break
	}

	// 添加新任务到内存
	for _, newEntity := range toAdd {
		t.AddTaskEntityToMemory(newEntity)
	}
	// 删除旧任务（内存）
	for _, delEntity := range toDelete {
		t.DeteleTaskEntityFormMemory(delEntity)
	}
}

func (t *TaskModel) CheckTaskProgressData(entity *TaskEntity) {
	taskCoreId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
		return
	}
	taskCfg := gameConfig.GetCoreCfg(taskCoreId)
	if taskCfg == nil {
		logger.ErrorWithZapFields("get core task cfg error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
		return
	}
	num := int32(0)
	switch taskCfg.TaskType {
	case enum.ObjectiveTypeAnyHeroReachWhatLevel:
		if t.player.StaticData.Entity.HeroHistoryMaxLevel >= taskCfg.TaskPara[0] {
			num = t.player.StaticData.Entity.HeroHistoryMaxLevel
		}
	case enum.ObjectiveTypePassWhatMainLevel:
		maxStageId := t.player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)].MaxStageId
		if maxStageId >= taskCfg.TaskPara[0] {
			num = taskCfg.TaskNum
		}
	case enum.ObjectiveTypeHowManyHeroReachWhatStar:
		for _, v := range t.player.HeroDetailsModel.Entities {
			if v.StarLevel >= taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeHowManyHeroReachWhatLevel:
		for _, v := range t.player.HeroDetailsModel.Entities {
			if v.Level >= taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeTowerChallengePassWhatLevel:
		if t.player.PlayerInstanceModel.InstanceEntities[int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID)] == nil {
			return
		}
		maxStageId := t.player.PlayerInstanceModel.InstanceEntities[int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID)].MaxStageId
		if maxStageId >= taskCfg.TaskPara[0] {
			num = taskCfg.TaskNum
		}
	case enum.ObjectiveTypeWhatBuildLevelUpWhat:
		if t.player.ArchitectureModel.Entities[taskCfg.TaskPara[0]] == nil {
			return
		}
		num = t.player.ArchitectureModel.Entities[taskCfg.TaskPara[0]].Level
	case enum.ObjectiveTypeCumulativeDispatchKillMonsterHowMany: // 从玩家常量表里获取
		num = t.player.StaticData.Entity.ExpeditionNum
	case enum.ObjectiveTypePlayerPowerReachWhat:
		attrNum := int64(0)
		for _, v := range t.player.HeroFormationModel.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
			if v.IsActive {
				for _, value := range v.HeroOwnIDList {
					attrNum += t.player.GetHeroAttr(value)[enum.AttributeBasicCombatPower]
				}
				break
			}
		}
		num = int32(attrNum)
	case enum.ObjectiveTypeAllBuildLevelReachWhat:
		for _, v := range t.player.ArchitectureModel.Entities {
			num += v.Level
		}
	case enum.ObjectiveTypeHeroQuantityReachWhat:
		num = int32(len(t.player.HeroAlbumModel.Entities))
	case enum.ObjectiveTypeHowManyHeroReachWhatPotential:
		for _, v := range t.player.HeroDetailsModel.Entities {
			if v.IsDeleted {
				continue
			}
			cfg := gameConfig.GetHeroBaseCfg(int32(v.HeroID))
			if cfg != nil && cfg.HeroPotential == taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeWhatDispatchMapUnlockWhatStage:
		cfg := gameConfig.GetCityDispatchCfg(taskCfg.TaskPara[0], taskCfg.TaskPara[1])
		if cfg == nil {
			return
		}
		flag := unlockService.CheckUnlock(cfg.Unlock, t.player)
		if flag {
			num = taskCfg.TaskNum
		}
	case enum.ObjectiveTypeLoopBoxSystemLevelReachWhat:
		num = t.player.LoopBoxModel.LoopBoxEntity.SystemLevel
	case enum.ObjectiveTypeAccessorySystemLevelReachWhat:
		for _, v := range t.player.AccessoryLuckyModel.Entities {
			if v.LuckyLevel > num {
				num = v.LuckyLevel
			}
		}
	case enum.ObjectiveTypeHowManyEquipStrongReachWhatLevel:
		for _, v := range t.player.EquipmentModel.Entities {
			if v.IsDeleted {
				continue
			}
			if v.StrongLevel >= taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeHowManyPetReachWhatLevel:
		for _, v := range t.player.PetModel.Entities {
			if v.IsDeleted {
				continue
			}
			if v.Level >= taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeJoinAlliance:
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(t.UserId)
		if allianceInfo != nil && allianceInfo.AllianceId > 0 {
			num = taskCfg.TaskNum
		}
	case enum.ObjectiveTypeHowManyPetReachWhatStar:
		for _, v := range t.player.PetModel.Entities {
			if v.IsDeleted {
				continue
			}
			if v.Star >= taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeWearHowManyEquipmentQuality:
		for _, v := range t.player.EquipmentModel.Entities {
			if v.IsDeleted || v.HeroOwnID <= 0 {
				continue
			}
			cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(v.EquipmentID)
			if cfg == nil {
				continue
			}
			if cfg.EquipmentQuality == taskCfg.TaskPara[0] {
				num++
			}
		}
	case enum.ObjectiveTypeArenaScoreReachWhat:
		num = t.player.PlayerArenaModel.GetScore()
	case enum.ObjectiveTypeMainTaskPassWhatNum:
		for _, taskTypeMap := range t.TaskEntity[enum.TaskAffiliationMain] {
			for _, mainTask := range taskTypeMap {
				cfg := gameConfig.GetMainCfg(mainTask.TaskID)
				if cfg == nil {
					continue
				}
				finishNum := cfg.Num - 1
				if mainTask.Status != enum.TaskStatusUnFinish {
					finishNum = cfg.Num
				}
				if finishNum > num {
					num = finishNum
				}
			}
		}
	case enum.ObjectiveTypeStoneClassTotalLevelReachWhat:
		for _, stone := range t.player.StoneModel.Entities {
			for _, level := range stone.AttrLevel {
				num += level
			}
		}
	case enum.ObjectiveTypeCollectionLotteryHowManyCumulative:
		for _, v := range t.player.LotteryModel.LotteryEntities {
			lotterCfg := gameConfig.GetSummonPoolCfg(v.Id)
			if lotterCfg == nil {
				continue
			}
			if lotterCfg.Gashatype == 2 {
				num += v.AllCount
			}
		}
	case enum.ObjectiveTypePetRecruitHowManyCumulative:
		num = t.player.StaticData.GetPetRecruitCount()
	case enum.ObjectiveTypeDungeonParticipateHowManyCumulative:
		num = t.player.StaticData.GetResidentInstanceJoinCount()
	case enum.ObjectiveTypeWearHowManyEquipmentLevel:
		for _, v := range t.player.EquipmentModel.Entities {
			if v.IsDeleted || v.HeroOwnID <= 0 {
				continue
			}
			cfg := gameConfig.GetEquipmentBaseCfg(v.EquipmentID)
			if cfg == nil {
				continue
			}
			if cfg.Tier == taskCfg.TaskPara[0] {
				num++
			}
		}
	}

	if num > entity.ProgressData {
		t.UpdateTaskProgressData(entity.TaskID, taskCfg.TaskType, entity.TaskAttribution, num)
		if entity.ProgressData >= taskCfg.TaskNum {
			t.UpdateTaskStatus(entity.TaskID, taskCfg.TaskType, entity.TaskAttribution, enum.TaskStatusFinishUnReward)
			messageSender.SendMessage(t.player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
				Attribution: entity.TaskAttribution,
				TaskId:      entity.TaskID,
				TaskState:   entity.Status,
				Progress:    entity.ProgressData,
			})
		} else if entity.TaskAttribution == enum.TaskAffiliationMain || entity.TaskAttribution == enum.TaskAffiliationCityAge {
			messageSender.SendMessage(t.player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
				Attribution: entity.TaskAttribution,
				TaskId:      entity.TaskID,
				TaskState:   entity.Status,
				Progress:    entity.ProgressData,
			})
		}
	}
}

func (t *TaskModel) RefreshDailyTasks() []*TaskEntity {
	var newTasks []*TaskEntity

	// 1. 统计需要多少新任务
	dailyTaskMap := t.TaskEntity[enum.TaskAffiliationDaily]

	allDailyTasksCfg := gameConfig.GetAllDailyTaskIDs()
	if allDailyTasksCfg == nil {
		return newTasks
	}
	totalNeeded := len(allDailyTasksCfg)

	slotNum := 0
	for _, taskIdList := range dailyTaskMap {
		slotNum += len(taskIdList)
	}

	// 2. 生成不重复的任务ID列表
	taskIDs := t.generateUniqueDailyTaskIDs(int32(totalNeeded))
	if len(taskIDs) == 0 {
		return newTasks
	}

	totalNeeded = len(taskIDs)
	// 3. 遍历并替换所有每日任务（slot固定，用TaskEntityBySlot判断）
	for i := 0; i < totalNeeded; i++ {
		taskID := taskIDs[i]
		slotId := enum.TaskAffiliationDaily*10000 + int32(i) + 1
		newTask := &TaskEntity{
			UserID:          t.UserId,
			SlotId:          slotId,
			TaskID:          taskID,
			TaskAttribution: enum.TaskAffiliationDaily,
			ProgressData:    0,
			Status:          enum.TaskStatusUnFinish,
			AddTime:         tool.UnixNowMilli(),
			UpdateTime:      tool.UnixNowMilli(),
		}
		// 用TaskEntityBySlot判断slot是否已存在
		if oldEntity := t.TaskEntityBySlot[slotId]; oldEntity != nil {
			// 先添加新任务到内存（覆盖旧slot）
			t.AddTaskEntityToMemory(newTask)
			t.UpdateChangedWithNewData(oldEntity, newTask)
			// 标记旧任务状态（用于后续清理TaskEntity索引）
			oldEntity.Status = enum.TaskStatusDeleteInMemory
		} else {
			// slot不存在，添加到内存并尝试INSERT
			t.AddTaskEntityToMemory(newTask)
			err := easyDB.CreatePlayerEntity(newTask)
			if err != nil {
				// INSERT失败（可能数据库已有记录），改用UPDATE
				if t.Changed[enum.TaskAffiliationDaily] == nil {
					t.Changed[enum.TaskAffiliationDaily] = make(map[int32]map[string]interface{})
				}
				t.Changed[enum.TaskAffiliationDaily][slotId] = map[string]interface{}{
					"task_id":       newTask.TaskID,
					"progress_data": newTask.ProgressData,
					"status":        newTask.Status,
					"add_time":      newTask.AddTime,
					"update_time":   newTask.UpdateTime,
				}
			}
		}
		newTasks = append(newTasks, newTask)
	}

	return newTasks
}

// generateUniqueDailyTaskIDs 生成指定数量的不重复每日任务ID
func (t *TaskModel) generateUniqueDailyTaskIDs(neededCount int32) []int32 {
	// 需要从配置中获取所有每日任务的id列表
	// 假设有方法获取所有每日任务ID
	allDailyTaskIDs := gameConfig.GetAllDailyTaskIDs() // 需要实现这个方法

	if len(allDailyTaskIDs) == 0 || neededCount <= 0 {
		return nil
	}
	if int32(len(allDailyTaskIDs)) < neededCount {
		neededCount = int32(len(allDailyTaskIDs))
	}

	// 复制并打乱
	shuffledIDs := make([]int32, len(allDailyTaskIDs))
	copy(shuffledIDs, allDailyTaskIDs)

	// Fisher-Yates洗牌
	for i := len(shuffledIDs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffledIDs[i], shuffledIDs[j] = shuffledIDs[j], shuffledIDs[i]
	}

	// 过滤有效任务
	var validTaskIDs []int32

	for _, taskID := range shuffledIDs {
		if int32(len(validTaskIDs)) >= neededCount {
			break
		}

		dailyCfg := gameConfig.GetDailyCfg(taskID)
		if dailyCfg == nil {
			continue
		}

		if dailyCfg.Unlock != 0 && !unlockService.CheckUnlock(dailyCfg.Unlock, t.player) {
			continue
		}

		validTaskIDs = append(validTaskIDs, taskID)
	}

	return validTaskIDs
}

func (t *TaskModel) CheckNewTask(attributionId int32, taskTypeId int32, taskId int32) *TaskEntity {
	// 获取当前实体，做必要的空检查
	currMapAttribution, ok := t.TaskEntity[attributionId]
	if !ok {
		return nil
	}
	currMapType, ok := currMapAttribution[taskTypeId]
	if !ok {
		return nil
	}
	currEntity, ok := currMapType[taskId]
	if !ok {
		return nil
	}

	// 按归属计算下一个任务（仅计算，不修改内存）
	if attributionId == enum.TaskAffiliationMain {
		taskCfg := gameConfig.GetMainCfg(taskId)
		if taskCfg == nil {
			return nil
		}
		taskNextCfg := gameConfig.GetMainCfg(taskCfg.FollowTasks)
		if taskNextCfg == nil {
			return nil
		}

		if taskNextCfg.Unlock != 0 && !unlockService.CheckUnlock(taskNextCfg.Unlock, t.player) {
			return nil
		}
		return NewTaskEntity(currEntity.UserID, currEntity.SlotId, taskNextCfg.Id, enum.TaskAffiliationMain, 0, 0, tool.UnixNowMilli(), 0)
	} else if attributionId == enum.TaskAffiliationSide {
		taskCfg := gameConfig.GetSecondaryCfg(taskId)
		if taskCfg == nil {
			return nil
		}
		taskNextCfg := gameConfig.GetSecondaryCfg(taskCfg.FollowTasks)
		if taskNextCfg == nil {
			return nil
		}

		if taskNextCfg.Unlock != 0 && !unlockService.CheckUnlock(taskNextCfg.Unlock, t.player) {
			return nil
		}
		return NewTaskEntity(currEntity.UserID, currEntity.SlotId, taskNextCfg.Id, enum.TaskAffiliationSide, 0, 0, tool.UnixNowMilli(), 0)
	}

	return nil
}

func (t *TaskModel) SaveModelToDB() {
	if t.Changed == nil || len(t.Changed) == 0 {
		return
	}

	for attr, slotMap := range t.Changed {
		for slotId, change := range slotMap {
			// 通过SlotId查找entity
			entity := t.TaskEntityBySlot[slotId]
			if entity == nil {
				logger.ErrorWithZapFields("SaveModelToDB: entity not found by slotId", zap.Int32("attribution", attr), zap.Int32("slot_id", slotId))
				continue
			}
			easyDB.UpdatePlayerEntity[TaskEntity](entity, change, t.UserId)
		}
	}

	// 清空 Changed
	t.Changed = make(map[int32]map[int32]map[string]interface{})
}

func NewTaskModel(userId int64, entity map[int32]map[int32]map[int32]*TaskEntity, entityBySlot map[int32]*TaskEntity, player *PlayerModel) *TaskModel {
	return &TaskModel{
		UserId:            userId,
		TaskEntity:        entity,
		Changed:           make(map[int32]map[int32]map[string]interface{}),
		TaskEntityBySlot:  entityBySlot,
		NeedCheckTaskList: make([]*TaskEntity, 0),
		player:            player,
	}
}

func (t *TaskModel) AddTask(entity *TaskEntity) error {
	t.AddTaskEntityToMemory(entity)
	err := easyDB.CreatePlayerEntity(entity)
	if err != nil {
		return err
	}
	return nil
}

func (t *TaskModel) UpdateTaskStatus(taskID int32, taskType int32, taskAttribution int32, status int32) {
	if t.TaskEntity[taskAttribution] == nil {
		t.TaskEntity[taskAttribution] = make(map[int32]map[int32]*TaskEntity)
	}
	if t.TaskEntity[taskAttribution][taskType] == nil {
		t.TaskEntity[taskAttribution][taskType] = make(map[int32]*TaskEntity)
	}
	entity, ok := t.TaskEntity[taskAttribution][taskType][taskID]
	if !ok {
		return
	}

	// 如果任务状态从非完成变为完成(1)，上报任务完成日志
	if entity.Status != enum.TaskStatusFinishUnReward && status == enum.TaskStatusFinishUnReward {
		operationLogService.OnUserFinishTask(t.UserId, taskAttribution, taskID)
	}

	entity.Status = status
	if t.Changed[taskAttribution] == nil {
		t.Changed[taskAttribution] = make(map[int32]map[string]interface{})
	}
	if t.Changed[taskAttribution][entity.SlotId] == nil {
		t.Changed[taskAttribution][entity.SlotId] = make(map[string]interface{})
	}
	t.Changed[taskAttribution][entity.SlotId]["status"] = status
}

func (t *TaskModel) UpdateTaskProgressData(taskID int32, taskType int32, taskAttribution int32, progressData int32) {
	if t.TaskEntity[taskAttribution] == nil {
		t.TaskEntity[taskAttribution] = make(map[int32]map[int32]*TaskEntity)
	}
	if t.TaskEntity[taskAttribution][taskType] == nil {
		t.TaskEntity[taskAttribution][taskType] = make(map[int32]*TaskEntity)
	}
	entity, ok := t.TaskEntity[taskAttribution][taskType][taskID]
	if !ok {
		return
	}
	entity.ProgressData = progressData
	if t.Changed[taskAttribution] == nil {
		t.Changed[taskAttribution] = make(map[int32]map[string]interface{})
	}
	if t.Changed[taskAttribution][entity.SlotId] == nil {
		t.Changed[taskAttribution][entity.SlotId] = make(map[string]interface{})
	}
	t.Changed[taskAttribution][entity.SlotId]["progress_data"] = progressData
}

func (t *TaskModel) UpdateUpdateTime(taskID int32, taskType int32, taskAttribution int32, updateTime int64) {
	if t.TaskEntity[taskAttribution] == nil {
		t.TaskEntity[taskAttribution] = make(map[int32]map[int32]*TaskEntity)
	}
	if t.TaskEntity[taskAttribution][taskType] == nil {
		t.TaskEntity[taskAttribution][taskType] = make(map[int32]*TaskEntity)
	}
	entity, ok := t.TaskEntity[taskAttribution][taskType][taskID]
	if !ok {
		return
	}
	entity.UpdateTime = updateTime
	if t.Changed[taskAttribution] == nil {
		t.Changed[taskAttribution] = make(map[int32]map[string]interface{})
	}
	if t.Changed[taskAttribution][entity.SlotId] == nil {
		t.Changed[taskAttribution][entity.SlotId] = make(map[string]interface{})
	}
	t.Changed[taskAttribution][entity.SlotId]["update_time"] = updateTime
}

func (t *TaskModel) DeteleTaskEntityFormMemory(entity *TaskEntity) {
	taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
		return
	}
	if taskId == 0 {
		return
	}
	if gameConfig.GetCoreCfg(taskId) == nil {
		return
	}
	taskType := gameConfig.GetCoreCfg(taskId).TaskType

	taskAttribution := entity.TaskAttribution
	taskID := entity.TaskID

	if t.TaskEntity[taskAttribution] == nil {
		t.TaskEntity[taskAttribution] = make(map[int32]map[int32]*TaskEntity)
	}
	if t.TaskEntity[taskAttribution][taskType] == nil {
		t.TaskEntity[taskAttribution][taskType] = make(map[int32]*TaskEntity)
	}
	delete(t.TaskEntity[taskAttribution][taskType], taskID)
	if len(t.TaskEntity[taskAttribution][taskType]) == 0 {
		delete(t.TaskEntity[taskAttribution], taskType)
	}
	if len(t.TaskEntity[taskAttribution]) == 0 {
		delete(t.TaskEntity, taskAttribution)
	}
}

func (t *TaskModel) AddTaskEntityToMemory(entity *TaskEntity) {
	taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
		return
	}
	if taskId == 0 {
		return
	}
	if gameConfig.GetCoreCfg(taskId) == nil {
		return
	}
	taskType := gameConfig.GetCoreCfg(taskId).TaskType

	if t.TaskEntity[entity.TaskAttribution] == nil {
		t.TaskEntity[entity.TaskAttribution] = make(map[int32]map[int32]*TaskEntity)
	}
	if t.TaskEntity[entity.TaskAttribution][taskType] == nil {
		t.TaskEntity[entity.TaskAttribution][taskType] = make(map[int32]*TaskEntity)
	}
	t.TaskEntity[entity.TaskAttribution][taskType][entity.TaskID] = entity
	t.TaskEntityBySlot[entity.SlotId] = entity
	t.NeedCheckTaskList = append(t.NeedCheckTaskList, entity)
}

func (t *TaskModel) UpdateChangedWithNewData(oldEntity *TaskEntity, newEntity *TaskEntity) {
	if t.Changed == nil {
		t.Changed = make(map[int32]map[int32]map[string]interface{})
	}

	if t.Changed[oldEntity.TaskAttribution] == nil {
		t.Changed[oldEntity.TaskAttribution] = make(map[int32]map[string]interface{})
	}
	// 关键：用新任务的slotId作为key，存储新任务的全部数据
	t.Changed[oldEntity.TaskAttribution][newEntity.SlotId] = map[string]interface{}{
		"task_id":          newEntity.TaskID,
		"task_attribution": newEntity.TaskAttribution,
		"progress_data":    newEntity.ProgressData,
		"status":           newEntity.Status,
		"add_time":         newEntity.AddTime,
		"update_time":      newEntity.UpdateTime,
	}
}

// 检查是否有新的支线任务组需要添加
func (t *TaskModel) CheckNewSideTaskGroup(toAdd *[]*TaskEntity) {
	maxGroup := len(gameConfig.GetSideTaskMap())
	if maxGroup <= 0 {
		return
	}

	userID := t.UserId

	exTaskGroup := make(map[int32]bool)
	num := 0
	for _, taskIdMap := range t.TaskEntity[enum.TaskAffiliationSide] {
		for _, e := range taskIdMap {
			num++
			if cfg := gameConfig.GetSecondaryCfg(e.TaskID); cfg != nil {
				exTaskGroup[cfg.TaskGroup] = true
			}
		}
	}

	// 遍历每个组，查找未开启的组并尝试开第一个任务
	for group := int32(1); group <= int32(maxGroup); group++ {

		if exTaskGroup[group] {
			continue
		}

		// 该组尚未开启，获取该组的第一个候选任务 id
		firstTaskID := gameConfig.GetSideTaskMap()[group][0].Id
		if firstTaskID <= 0 {
			continue
		}
		secCfg := gameConfig.GetSideTaskMap()[group][0]
		if secCfg == nil {
			continue
		}

		if secCfg.Unlock != 0 && !unlockService.CheckUnlock(secCfg.Unlock, t.player) {
			continue
		}
		// 构造新任务并加入内存
		newEntity := NewTaskEntity(userID, enum.TaskAffiliationSide*10000+int32(num)+1, firstTaskID, enum.TaskAffiliationSide, 0, enum.TaskStatusUnFinish, tool.UnixNowMilli(), 0)
		*toAdd = append(*toAdd, newEntity)
		num++

		err := easyDB.CreatePlayerEntity[TaskEntity](newEntity)
		if err != nil {
			logger.ErrorWithZapFields("CreateEntity TaskEntity error", zap.Int64("user_id", newEntity.UserID), zap.Int32("task_id", newEntity.TaskID), zap.Error(err))
			continue
		}
		messageSender.SendMessage(t.player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
			Attribution: newEntity.TaskAttribution,
			TaskId:      newEntity.TaskID,
			TaskState:   newEntity.Status,
			Progress:    newEntity.ProgressData,
		})
	}
}

func LoadTaskModel(player *PlayerModel) (*TaskModel, error) {
	entities := make(map[int32]map[int32]map[int32]*TaskEntity)
	entitiesBySlot := make(map[int32]*TaskEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[TaskEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		return NewTaskModel(player.GetUserId(), make(map[int32]map[int32]map[int32]*TaskEntity), make(map[int32]*TaskEntity), player), err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		if entities[row.TaskAttribution] == nil {
			entities[row.TaskAttribution] = make(map[int32]map[int32]*TaskEntity)
		}
		taskId, err := gameConfig.GetCoreTaskId(row.TaskAttribution, row.TaskID)
		if err != nil {
			logger.InfoWithZapFields("get core task id error", zap.String("account", player.User.GetAccount()), zap.Int32("attribution", row.TaskAttribution), zap.Int32("task id", row.TaskID))
			continue
		}
		if gameConfig.GetCoreCfg(taskId) == nil {
			continue
		}
		taskCfg := gameConfig.GetCoreCfg(taskId)
		if taskCfg == nil {
			continue
		}
		taskType := taskCfg.TaskType
		if entities[row.TaskAttribution][taskType] == nil {
			entities[row.TaskAttribution][taskType] = make(map[int32]*TaskEntity)
		}

		entities[row.TaskAttribution][taskType][row.TaskID] = row
		entitiesBySlot[row.SlotId] = row
	}
	return NewTaskModel(player.GetUserId(), entities, entitiesBySlot, player), nil
}

func (t *TaskModel) RewardDaily(a *TaskActiveRewardModel) ([]*gameConfig.ItemConfig, error) {
	res := make([]*gameConfig.ItemConfig, 0)

	for taskType, taskIdMap := range t.TaskEntity[enum.TaskAffiliationDaily] {
		for _, entity := range taskIdMap {
			taskId, err := gameConfig.GetCoreTaskId(entity.TaskAttribution, entity.TaskID)
			if err != nil {
				logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", entity.TaskAttribution), zap.Int32("task id", entity.TaskID))
				return nil, errors.New("get core task id error")
			}
			if taskId == 0 {
				return nil, errors.New("task core is null")
			}
			taskCore := gameConfig.GetCoreCfg(taskId)
			if taskCore == nil {
				return nil, errors.New("task core is null")
			}
			if entity.Status == enum.TaskStatusFinishUnReward {
				if entity.ProgressData >= taskCore.TaskNum {
					dailyCfg := gameConfig.GetDailyCfg(entity.TaskID)
					if dailyCfg == nil {
						return nil, errors.New("cfg is null")
					}

					a.UpdateDailyUpdateTime(tool.UnixNowMilli())
					a.UpdateWeekUpdateTime(tool.UnixNowMilli())
					for _, v := range dailyCfg.TaskReward {
						itemCfg := gameConfig.GetItemCfg(v.ID)
						if itemCfg == nil {
							continue
						}
						res = append(res, v)
						if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_CURRENCY) && itemCfg.IsAutoUse == 1 {
							a.UpdateDailyPoint(a.Entity.DailyPoint + int32(v.Num))
							a.UpdateWeekPoint(a.Entity.WeekPoint + int32(v.Num))
						}
					}
					for i, v := range dailyCfg.ActId {
						if flag, _ := t.player.PlayerActivityModel.CheckActivityOpen(v); flag {
							res = append(res, dailyCfg.Item[i])
						}
					}
					t.UpdateTaskStatus(entity.TaskID, taskType, enum.TaskAffiliationDaily, enum.TaskStatusFinishAndReward)
					t.UpdateUpdateTime(entity.TaskID, taskType, enum.TaskAffiliationDaily, tool.UnixNowMilli())
				} else {
					t.UpdateTaskStatus(entity.TaskID, taskType, enum.TaskAffiliationDaily, enum.TaskStatusUnFinish)
					t.UpdateUpdateTime(entity.TaskID, taskType, enum.TaskAffiliationDaily, tool.UnixNowMilli())
					messageSender.SendMessage(t.player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
						Attribution: entity.TaskAttribution,
						TaskId:      entity.TaskID,
						TaskState:   entity.Status,
						Progress:    entity.ProgressData,
					})
				}
			}
		}
	}

	return res, nil
}

func (t *TaskModel) CheckTaskStatus(attribution int32, taskId int32) (int32, bool) {
	taskId, err := gameConfig.GetCoreTaskId(attribution, taskId)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", attribution), zap.Int32("task id", taskId))
		return -1, false
	}
	if taskId == 0 {
		return -1, false
	}
	cfg := gameConfig.GetCoreCfg(taskId)
	if cfg == nil {
		return -1, false
	}
	return t.TaskEntity[attribution][cfg.TaskType][taskId].Status, true
}

func (t *TaskModel) ReFreshTask(attribution int32, taskId int32) bool {
	taskId, err := gameConfig.GetCoreTaskId(attribution, taskId)
	if err != nil {
		logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", attribution), zap.Int32("task id", taskId))
		return false
	}
	if taskId == 0 {
		return false
	}
	cfg := gameConfig.GetCoreCfg(taskId)
	if cfg == nil {
		return false
	}
	t.UpdateTaskProgressData(taskId, cfg.TaskType, attribution, 0)
	t.UpdateTaskStatus(taskId, cfg.TaskType, attribution, enum.TaskStatusUnFinish)
	t.UpdateUpdateTime(taskId, taskId, cfg.TaskType, tool.UnixNowMilli())
	return true
}

// BatchRewardTasks 一键领取所有主线+支线可领取任务，返回合并后的奖励
func (t *TaskModel) BatchRewardTasks() ([]*pb.ItemBasicInfo, []*pb.PushTaskUpdate, error) {
	rewardMap := make(map[int32]int64) // itemId -> count 用于合并
	pushUpdates := make([]*pb.PushTaskUpdate, 0)
	addItems := make([]*gameConfig.ItemConfig, 0)
	// 领取主线任务
	if t.CheckSystemUnlock(enum.TaskAffiliationMain) {
		addItems = append(addItems, t.collectAndRewardTasks(enum.TaskAffiliationMain, rewardMap, &pushUpdates)...)
	}
	// 领取支线任务
	if t.CheckSystemUnlock(enum.TaskAffiliationSide) {
		addItems = append(addItems, t.collectAndRewardTasks(enum.TaskAffiliationSide, rewardMap, &pushUpdates)...)
	}
	if addItems != nil || len(addItems) > 0 {
		err := itemService.AddItems(t.player, addItems, enum.ITEM_CHANGE_REASON_TASK)
		if err != nil {
			return nil, nil, err
		}
	}

	// 转换奖励列表
	allRewards := make([]*pb.ItemBasicInfo, 0, len(rewardMap))
	for itemId, count := range rewardMap {
		allRewards = append(allRewards, &pb.ItemBasicInfo{
			ItemId: itemId,
			Count:  count,
		})
	}

	return allRewards, pushUpdates, nil
}

// collectAndRewardTasks 收集并领取指定归属的所有可领取任务
func (t *TaskModel) collectAndRewardTasks(attributionId int32, rewardMap map[int32]int64, pushUpdates *[]*pb.PushTaskUpdate) []*gameConfig.ItemConfig {
	addItems := make([]*gameConfig.ItemConfig, 0)
	for taskTypeId, taskMap := range t.TaskEntity[attributionId] {
		for taskId, entity := range taskMap {
			if entity == nil || entity.Status != enum.TaskStatusFinishUnReward {
				continue
			}

			taskCoreId, err := gameConfig.GetCoreTaskId(attributionId, taskId)
			if err != nil {
				continue
			}
			taskCore := gameConfig.GetCoreCfg(taskCoreId)
			if taskCore == nil || entity.ProgressData < taskCore.TaskNum {
				continue
			}

			// 获取奖励并合并
			taskCfgReward := gameConfig.GetTaskRewardsByAttribution(attributionId, taskId)
			for _, v := range taskCfgReward {
				rewardMap[v.ID] += v.Num
				addItems = append(addItems, &gameConfig.ItemConfig{ID: v.ID, Num: v.Num})
			}

			// 修改任务状态
			t.UpdateTaskStatus(taskId, taskTypeId, attributionId, enum.TaskStatusFinishAndReward)
			t.UpdateUpdateTime(taskId, taskTypeId, attributionId, tool.UnixNowMilli())

			// 检查新任务
			newTask := t.CheckNewTask(attributionId, taskTypeId, taskId)
			if newTask != nil {
				t.AddTaskEntityToMemory(newTask)
				t.UpdateChangedWithNewData(entity, newTask)
				t.SaveModelToDB()
				t.DeteleTaskEntityFormMemory(entity)
				*pushUpdates = append(*pushUpdates, &pb.PushTaskUpdate{
					Attribution: attributionId,
					TaskId:      newTask.TaskID,
					Progress:    0,
					TaskState:   enum.TaskStatusUnFinish,
				})
			} else {
				*pushUpdates = append(*pushUpdates, &pb.PushTaskUpdate{
					Attribution: attributionId,
					TaskId:      taskId,
					TaskState:   entity.Status,
					Progress:    entity.ProgressData,
				})
			}
		}
	}
	return addItems
}

// RewardSingleTask 领取单个任务奖励，返回奖励列表和推送更新
func (t *TaskModel) RewardSingleTask(attributionId, taskId int32) ([]*pb.ItemBasicInfo, *pb.PushTaskUpdate, error) {
	if !t.CheckSystemUnlock(attributionId) {
		return nil, nil, errors.New("task system not unlock")
	}

	taskCfgReward := gameConfig.GetTaskRewardsByAttribution(attributionId, taskId)

	taskCoreId, err := gameConfig.GetCoreTaskId(attributionId, taskId)
	if err != nil {
		return nil, nil, errors.New("task core id error")
	}
	taskCore := gameConfig.GetCoreCfg(taskCoreId)
	if taskCore == nil {
		return nil, nil, errors.New("task core is nil")
	}

	playerTask := t.TaskEntity[attributionId][taskCore.TaskType][taskId]
	if playerTask == nil {
		return nil, nil, errors.New("task not exist")
	}
	if playerTask.Status != enum.TaskStatusFinishUnReward && playerTask.ProgressData != taskCore.TaskNum {
		return nil, nil, errors.New("task not finish")
	}

	// 发放奖励
	for _, v := range taskCfgReward {
		err := itemService.AddItem(t.player, &gameConfig.ItemConfig{ID: v.ID, Num: v.Num}, enum.ITEM_CHANGE_REASON_TASK)
		if err != nil {
			return nil, nil, err
		}
	}

	// 发放同时修改任务状态
	t.UpdateTaskStatus(taskId, taskCore.TaskType, attributionId, enum.TaskStatusFinishAndReward)
	t.UpdateUpdateTime(taskId, taskCore.TaskType, attributionId, tool.UnixNowMilli())

	// 转换奖励列表
	allReward := make([]*pb.ItemBasicInfo, 0, len(taskCfgReward))
	for _, v := range taskCfgReward {
		allReward = append(allReward, &pb.ItemBasicInfo{
			ItemId: v.ID,
			Count:  v.Num,
		})
	}

	// 检查是否有新任务
	newTask := t.CheckNewTask(attributionId, taskCore.TaskType, taskId)
	var pushUpdate *pb.PushTaskUpdate
	if newTask != nil {
		t.AddTaskEntityToMemory(newTask)
		t.UpdateChangedWithNewData(playerTask, newTask)
		t.SaveModelToDB()
		t.DeteleTaskEntityFormMemory(playerTask)
		pushUpdate = &pb.PushTaskUpdate{
			Attribution: attributionId,
			TaskId:      newTask.TaskID,
			Progress:    0,
			TaskState:   enum.TaskStatusUnFinish,
		}
	} else {
		pushUpdate = &pb.PushTaskUpdate{
			Attribution: attributionId,
			TaskId:      taskId,
			TaskState:   playerTask.Status,
			Progress:    playerTask.ProgressData,
		}
	}

	return allReward, pushUpdate, nil
}

type TaskActiveRewardEntity struct {
	UserId          int64 `gorm:"column:user_id;primaryKey" json:"user_id"`
	DailyPoint      int32 `gorm:"column:daily_point;" json:"daily_point"`
	DailyUpdateTime int64 `gorm:"column:daily_update_time;" json:"daily_update_time"`
	DailyBox        int32 `gorm:"column:daily_box;" json:"daily_box"`
	WeekPoint       int32 `gorm:"column:week_point;" json:"week_point"`
	WeekUpdateTime  int64 `gorm:"column:week_update_time;" json:"week_update_time"`
	WeekBox         int32 `gorm:"column:week_box;" json:"week_box"`
}

func (t *TaskActiveRewardEntity) TableName() string {
	return "task_active_reward"
}

type TaskActiveRewardModel struct {
	UserId  int64
	Entity  *TaskActiveRewardEntity
	Changed map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*TaskActiveRewardModel)(nil)

func NewTaskActiveRewardModel(entity *TaskActiveRewardEntity, userId int64) *TaskActiveRewardModel {
	return &TaskActiveRewardModel{
		UserId:  userId,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func (t *TaskActiveRewardModel) CreatTaskActiveRewardEntity(entity *TaskActiveRewardEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (t *TaskActiveRewardModel) AddTaskActiveRewardEntity(userId int64) error {
	t.Entity = &TaskActiveRewardEntity{
		UserId:          userId,
		DailyPoint:      0,
		DailyUpdateTime: 0,
		DailyBox:        0,
		WeekPoint:       0,
		WeekUpdateTime:  0,
		WeekBox:         0,
	}
	return t.CreatTaskActiveRewardEntity(t.Entity)
}

func (t *TaskActiveRewardModel) SaveModelToDB() {
	if t.Changed != nil && len(t.Changed) > 0 {
		easyDB.UpdatePlayerEntity(t.Entity, t.Changed, t.UserId)
	}
	t.Changed = make(map[string]interface{})
}

func (t *TaskActiveRewardModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	now := time.Now()

	// 今天零点
	todayZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayZeroTs := todayZero.UnixNano() / 1e6

	if t.Entity.DailyUpdateTime < todayZeroTs {
		t.UpdateDailyPoint(0)
		t.UpdateDailyUpdateTime(tool.UnixNowMilli())
		t.UpdateDailyBox(0)

		weekday := now.Weekday()
		daysSinceMonday := (int32(weekday) + 6) % 7
		monday := now.AddDate(0, 0, -int(daysSinceMonday))
		mondayZero := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
		mondayZeroTs := mondayZero.UnixNano() / 1e6
		if t.Entity.WeekUpdateTime < mondayZeroTs {
			t.UpdateWeekUpdateTime(tool.UnixNowMilli())
			t.UpdateWeekPoint(0)
			t.UpdateWeekBox(0)
		}
	}
}

func LoadTaskActiveRewardModel(userId int64) (*TaskActiveRewardModel, error) {
	entity := &TaskActiveRewardEntity{}
	row, err := easyDB.GetPlayerEntityByID[TaskActiveRewardEntity](userId)
	if err != nil {
		entity.UserId = userId
		err = easyDB.CreatePlayerEntity(entity)
		if err != nil {
			return NewTaskActiveRewardModel(entity, userId), err
		}
	} else {
		entity = row
	}
	return NewTaskActiveRewardModel(entity, userId), nil
}

func (t *TaskActiveRewardModel) UpdateDailyPoint(point int32) {
	t.Entity.DailyPoint = point
	t.Changed["daily_point"] = point
}

func (t *TaskActiveRewardModel) UpdateDailyUpdateTime(time int64) {
	t.Entity.DailyUpdateTime = time
	t.Changed["daily_update_time"] = time
}

func (t *TaskActiveRewardModel) UpdateDailyBox(box int32) {
	t.Entity.DailyBox = box
	t.Changed["daily_box"] = box
}

func (t *TaskActiveRewardModel) UpdateWeekPoint(point int32) {
	t.Entity.WeekPoint = point
	t.Changed["week_point"] = point
}

func (t *TaskActiveRewardModel) UpdateWeekUpdateTime(time int64) {
	t.Entity.WeekUpdateTime = time
	t.Changed["week_update_time"] = time
}

func (t *TaskActiveRewardModel) UpdateWeekBox(box int32) {
	t.Entity.WeekBox = box
	t.Changed["week_box"] = box
}

func (t *TaskActiveRewardModel) Reward() []*gameConfig.ItemConfig {
	res := make([]*gameConfig.ItemConfig, 0)

	cfg := gameConfig.GetDailyAwardsCfg(1)
	dailyBox := t.Entity.DailyBox
	weekBox := t.Entity.WeekBox

	for cfg != nil {
		if cfg.Type == 1 {
			if t.Entity.DailyPoint >= cfg.Point && dailyBox < cfg.Id {
				itemList := gameConfig.Drop(cfg.DropId)
				res = append(res, itemList...)
				dailyBox = cfg.Id
			}
		} else {
			if t.Entity.WeekPoint >= cfg.Point && weekBox < cfg.Id {
				itemList := gameConfig.Drop(cfg.DropId)
				res = append(res, itemList...)
				weekBox = cfg.Id
			}
		}
		cfg = gameConfig.GetDailyAwardsCfg(cfg.Id + 1)
	}

	t.UpdateDailyBox(dailyBox)
	t.UpdateWeekBox(weekBox)

	return res
}

type BountyEntity struct {
	UserId   int64               `gorm:"column:user_id;primaryKey" json:"user_id"`
	BountyId int32               `gorm:"column:bounty_id;primaryKey" json:"bounty_id"`
	EndTime  int64               `gorm:"column:end_time" json:"end_time"`
	Status   int32               `gorm:"column:status" json:"status"`
	SlotList tool.JSONInt32Slice `gorm:"column:slot_list;type:json" json:"slot_list"`
}

func (b *BountyEntity) TableName() string {
	return "bounty"
}

type BountyModel struct {
	UserId               int64
	Entities             map[int32]*BountyEntity
	Changed              map[int32]map[string]interface{}
	LastHeartBeatTime    int64
	playerModel          *PlayerModel
	CanUseBountySlotList *[]int32
	AlreadyUseSlot       int32
}

func NewBountyModel(entities map[int32]*BountyEntity, userid int64, player *PlayerModel, CanUseBountySlotList *[]int32, alreadyUseSlot int32) *BountyModel {
	return &BountyModel{
		UserId:               userid,
		Entities:             entities,
		Changed:              make(map[int32]map[string]interface{}),
		LastHeartBeatTime:    0,
		playerModel:          player,
		CanUseBountySlotList: CanUseBountySlotList,
		AlreadyUseSlot:       alreadyUseSlot,
	}
}

var _ logicCommon.PlayerModelInterface = (*BountyModel)(nil)

func (b *BountyModel) SaveModelToDB() {
	for _, v := range b.Entities {
		if b.Changed[v.BountyId] != nil && len(b.Changed[v.BountyId]) != 0 {
			easyDB.UpdatePlayerEntity(v, b.Changed[v.BountyId], b.UserId)
		}
	}
	b.Changed = make(map[int32]map[string]interface{})
}

func (b *BountyModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if currentTime-b.LastHeartBeatTime >= 1000 {
		if !unlockService.CheckSystemUnlock(enum.FUNCTION_ID_BOUNTY, b.playerModel) {
			return
		}
		num := 0
		for _, v := range b.Entities {
			if v.Status == enum.TaskStatusUnFinish || v.Status == enum.TaskStatusFinishUnReward {
				num++
			}
		}
		b.LastHeartBeatTime = currentTime
		for _, bountyCfg := range gameConfig.GetBountyList() {
			unlockOpen := true
			for _, v := range bountyCfg.UnlockId {
				if v != 0 && !unlockService.CheckUnlock(v, b.playerModel) {
					unlockOpen = false
					break
				}
			}
			for _, v := range bountyCfg.BanUnlockid {
				if v != 0 && unlockService.CheckUnlock(v, b.playerModel) {
					unlockOpen = false
					break
				}
			}
			if !unlockOpen {
				continue
			}
			if b.Entities[bountyCfg.Id] != nil && (b.Entities[bountyCfg.Id].Status == enum.TaskStatusFinishAndReward || b.Entities[bountyCfg.Id].Status == enum.TaskStatusTimeOverUnFinish) {
				continue
			}
			if num < 3 {
				if bountyCfg.ChainBounty != 0 {
					if b.Entities[bountyCfg.Id] == nil && b.Entities[bountyCfg.PreviousBounty] != nil && (b.Entities[bountyCfg.PreviousBounty].Status != enum.TaskStatusUnFinish && b.Entities[bountyCfg.PreviousBounty].Status != enum.TaskStatusFinishUnReward) {
						//todo 检测各种条件
						if bountyCfg.ChainBounty == 1 {
							if b.Entities[bountyCfg.PreviousBounty].Status == enum.TaskStatusFinishAndReward {
								b.editAddBounty(bountyCfg)
								num++
								continue
							}
						} else {
							b.editAddBounty(bountyCfg)
							num++
							continue
						}

					} else if bountyCfg.PreviousBounty == 0 && b.Entities[bountyCfg.Id] == nil {

						b.editAddBounty(bountyCfg)
						num++
						continue
					}
				} else {
					if b.Entities[bountyCfg.Id] == nil {

						b.editAddBounty(bountyCfg)
						num++
						continue
					}
				}
			}

			if b.Entities[bountyCfg.Id] != nil && b.Entities[bountyCfg.Id].Status == enum.TaskStatusUnFinish {
				taskFlag := true
				for _, v := range b.Entities[bountyCfg.Id].SlotList {
					taskDetail := b.playerModel.TaskModel.TaskEntityBySlot[v]
					if taskDetail.Status != enum.TaskStatusFinishAndReward {
						taskFlag = false
						break
					}
				}
				if taskFlag {
					b.UpdateStatus(bountyCfg.Id, enum.TaskStatusFinishUnReward)
				}
			}

			if b.Entities[bountyCfg.Id] != nil && b.Entities[bountyCfg.Id].EndTime < currentTime {
				if b.Entities[bountyCfg.Id].Status == enum.TaskStatusUnFinish || b.Entities[bountyCfg.Id].Status == enum.TaskStatusFinishUnReward {
					b.editEndBounty(bountyCfg)
				}
			}
		}
	}
}

func (b *BountyModel) editAddBounty(bountyCfg *gameConfig.BountyBaseCfg) {
	res := make([]*pb.TaskInfo, 0)
	if b.Entities[bountyCfg.Id] == nil {
		newSlot := b.AlreadyUseSlot
		slotIdList := b.TakeSlot(len(bountyCfg.TaskId))
		for id, value := range bountyCfg.TaskId {

			err := b.AddBountyTask(bountyCfg.Id, value, slotIdList[id], newSlot)
			if err != nil {
				logger.ErrorWithZapFields("add bounty task error", zap.Error(err), zap.Int64("user_id:", b.UserId))
			}
			res = append(res, &pb.TaskInfo{
				TaskId:    value,
				TaskState: enum.TaskStatusUnFinish,
				Progress:  0,
			})
		}
		err := b.AddBounty(bountyCfg.Id, int64(bountyCfg.Duration), slotIdList)
		if err != nil {
			logger.ErrorWithZapFields("add bounty error", zap.Error(err), zap.Int64("user_id:", b.UserId))
		}
	}
	messageSender.SendMessage(b.playerModel, pb.MESSAGE_ID_PUSH_BOUNTY_START, &pb.PushBountyStart{
		Info: &pb.BountyInfo{
			BountyId: bountyCfg.Id,
			TaskList: res,
			EndTime:  b.Entities[bountyCfg.Id].EndTime,
		},
	})
}

func (b *BountyModel) editEndBounty(bountyCfg *gameConfig.BountyBaseCfg) {
	*b.CanUseBountySlotList = append(*b.CanUseBountySlotList, bountyCfg.Id)
	if b.Entities[bountyCfg.Id].Status == enum.TaskStatusFinishUnReward {
		b.UpdateStatus(bountyCfg.Id, enum.TaskStatusFinishAndReward)

		// 将配置中的奖励转换为邮件附件格式
		items := make([]*logicCommon.MailAttachmentItem, 0, len(bountyCfg.Drop))
		for _, drop := range bountyCfg.Drop {
			if drop != nil {
				items = append(items, &logicCommon.MailAttachmentItem{
					Type: 1, // 1 表示道具
					ID:   drop.ID,
					Num:  int32(drop.Num),
				})
			}
		}

		// 如果没有奖励，不需要发邮件
		if len(items) == 0 {
			logger.ErrorWithZapFields("[Bounty] bounty has no reward items",
				zap.Int64("user_id", b.UserId),
				zap.Int32("bounty_id", bountyCfg.Id))
		} else {
			// 计算过期时间（7 天后）
			expireTime := time.Now().Unix() + 7*24*3600

			mailObj := &logicCommon.Mail{
				UserID:       b.UserId,
				MailType:     3, // 表示官方邮件
				Title:        "任务超时奖励",
				Content:      "您的悬赏令奖励超时未领取，现已通过邮件补发",
				SenderID:     0,
				SenderName:   "系统",
				Items:        items,
				ExpireTime:   expireTime, // 7 天后过期
				SendTime:     time.Now().Unix(),
				IsConvenient: true,
			}

			_, err := mailServer.SendMailToUserId(b.UserId, mailObj)
			if err != nil {
				logger.ErrorWithZapFields("[Bounty] send expire mail failed",
					zap.Int64("user_id", b.UserId),
					zap.Int32("bounty_id", bountyCfg.Id),
					zap.Error(err))
			} else {
				logger.InfoWithZapFields("[Bounty] sent expire reward mail success",
					zap.Int64("user_id", b.UserId),
					zap.Int32("bounty_id", bountyCfg.Id),
					zap.Int("items_count", len(items)))
			}
		}
	} else {
		b.UpdateStatus(bountyCfg.Id, enum.TaskStatusTimeOverUnFinish)
	}
	messageSender.SendMessage(b.playerModel, pb.MESSAGE_ID_PUSH_BOUNTY_END, &pb.PushBountyEnd{
		BountyId: bountyCfg.Id,
	})
}

func (b *BountyModel) UpdateStatus(bountyId, status int32) {
	b.Entities[bountyId].Status = status
	if b.Changed[bountyId] == nil {
		b.Changed[bountyId] = make(map[string]interface{})
	}
	b.Changed[bountyId]["status"] = status
}

func (b *BountyModel) UpdateSlotList(bountyId int32, slotList tool.JSONInt32Slice) {
	b.Entities[bountyId].SlotList = slotList
	if b.Changed[bountyId] == nil {
		b.Changed[bountyId] = make(map[string]interface{})
	}
	b.Changed[bountyId]["slot_list"] = slotList
}

func (b *BountyModel) CreatBounty(entity *BountyEntity) error {
	return easyDB.CreatePlayerEntity[BountyEntity](entity)
}

func (b *BountyModel) AddBounty(bountyId int32, duration int64, slotList tool.JSONInt32Slice) error {
	b.Entities[bountyId] = &BountyEntity{
		UserId:   b.UserId,
		BountyId: bountyId,
		EndTime:  tool.UnixNowMilli() + duration*1000,
		Status:   enum.TaskStatusUnFinish,
		SlotList: slotList,
	}
	return b.CreatBounty(b.Entities[bountyId])
}

func LoadBounty(userid int64, player *PlayerModel) (*BountyModel, error) {
	entities := make(map[int32]*BountyEntity)
	slots := make([]int32, 0)
	finishTaskSlotList := &slots
	num := 0

	rows, err := easyDB.GetPlayerEntitiesByWhere[BountyEntity](map[string]interface{}{"user_id": userid})
	if err != nil {
		return NewBountyModel(entities, userid, player, finishTaskSlotList, enum.TaskAffiliationBounty*10000), err
	}
	for _, v := range rows {
		if v != nil {
			entities[v.BountyId] = v
			if (v.Status == enum.TaskStatusTimeOverUnFinish || v.Status == enum.TaskStatusFinishAndReward) && len(v.SlotList) > 0 {
				*finishTaskSlotList = append(*finishTaskSlotList, v.BountyId)
			}
			num += len(v.SlotList)
		}
	}

	return NewBountyModel(entities, userid, player, finishTaskSlotList, enum.TaskAffiliationBounty*10000+int32(num)), nil
}

func (b *BountyModel) AddBountyTask(bountyId int32, taskId int32, slotId int32, newSlot int32) error {
	newEntity := NewTaskEntity(b.UserId, slotId, taskId, enum.TaskAffiliationBounty, 0, enum.TaskStatusUnFinish, tool.UnixNowMilli(), 0)
	if slotId > newSlot {
		err := b.playerModel.TaskModel.AddTask(newEntity)
		if err != nil {
			return err
		}
	} else {
		oldEntity := b.playerModel.TaskModel.TaskEntityBySlot[slotId]
		b.playerModel.TaskModel.UpdateChangedWithNewData(oldEntity, newEntity)
		b.playerModel.TaskModel.AddTaskEntityToMemory(newEntity)
		//b.playerModel.TaskModel.deteleTaskEntityFormMemory(oldEntity.TaskID, gameConfig.GetCoreCfg(gameConfig.GetCoreTaskId(oldEntity.TaskAttribution, oldEntity.TaskID)).TaskType, oldEntity.TaskAttribution)
	}
	b.playerModel.TaskModel.TaskEntityBySlot[slotId] = newEntity
	return nil
}

func (b *BountyModel) TakeSlot(n int) []int32 {
	res := make([]int32, 0)

	if n <= 0 {
		return res
	}
	if b.CanUseBountySlotList != nil && len(*b.CanUseBountySlotList) > 0 {

		for len(*b.CanUseBountySlotList) > 0 {
			if len(res) >= n {
				break
			}
			bountyId := (*b.CanUseBountySlotList)[0]                // 取第一个
			*b.CanUseBountySlotList = (*b.CanUseBountySlotList)[1:] // 移除第一个
			entity := b.Entities[bountyId]
			slotIdList := entity.SlotList
			need := n - len(res) // 还需要多少个

			// 从这个悬赏取槽位
			if need >= len(slotIdList) {
				// 要取的比有的多，全取
				res = append(res, slotIdList...)
				// 清空这个悬赏的槽位
				b.UpdateSlotList(bountyId, tool.JSONInt32Slice{})
			} else {
				// 要取的比有的少，只取一部分
				for i := 0; i < need; i++ {
					res = append(res, slotIdList[i])
				}
				// 更新剩余的槽位
				remaining := slotIdList[need:]
				b.UpdateSlotList(bountyId, remaining)
				*b.CanUseBountySlotList = append(*b.CanUseBountySlotList, bountyId)
			}
		}
		if len(res) < n {
			for i := len(res) + 1; i <= n; i++ {
				res = append(res, b.AlreadyUseSlot+1)
				b.AlreadyUseSlot++
			}
		}
	} else {
		for i := 0; i < n; i++ {
			res = append(res, b.AlreadyUseSlot+1)
			b.AlreadyUseSlot++
		}
	}

	return res
}

type PassCardTaskEntity struct {
	UserId          int64               `gorm:"column:user_id;primaryKey"`
	PassCardId      int32               `gorm:"column:pass_card_id;primaryKey"`
	Status          int32               `gorm:"column:status"`
	TaskSlotList    tool.JSONInt32Slice `gorm:"column:task_slot_list;type:json"`
	TaskFinishCount tool.JSONInt32Slice `gorm:"column:task_finish_count;type:json"`
}

func (p *PassCardTaskEntity) TableName() string {
	return "pass_card_task"
}

type PassCardTaskModel struct {
	UserId                 int64
	Entities               map[int32]*PassCardTaskEntity
	Changed                map[int32]map[string]interface{}
	playerModel            *PlayerModel
	MaxSlotId              int32
	CanUsePassCardSlotList *[]int32
}

func NewPassCardTaskModel(entities map[int32]*PassCardTaskEntity, playerModel *PlayerModel, userId int64, maxSlotId int32, canUsePassCardSlotList *[]int32) *PassCardTaskModel {
	return &PassCardTaskModel{
		UserId:                 userId,
		Entities:               entities,
		Changed:                make(map[int32]map[string]interface{}),
		playerModel:            playerModel,
		MaxSlotId:              maxSlotId,
		CanUsePassCardSlotList: canUsePassCardSlotList,
	}
}

var _ logicCommon.PlayerModelInterface = (*PassCardTaskModel)(nil)

func (p *PassCardTaskModel) SaveModelToDB() {
	for id, v := range p.Changed {
		if v != nil || len(v) == 0 {
			easyDB.UpdatePlayerEntity(p.Entities[id], v, p.UserId)
		}
	}
	p.Changed = make(map[int32]map[string]interface{})
}

func (p *PassCardTaskModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	taskModel := p.playerModel.TaskModel
	for _, v := range p.Entities {
		cfg := gameConfig.GetPassTaskMap()[v.PassCardId]
		if cfg == nil {
			logger.InfoWithZapFields("pass card  cfg is null ", zap.Int32("pass card id", v.PassCardId))
			continue
		}
		taskFinishNum := 0
		for id, slotId := range v.TaskSlotList {
			taskEntity := taskModel.TaskEntityBySlot[slotId]
			if taskEntity.Status == enum.TaskStatusFinishUnReward {
				cfg := gameConfig.GetPassTask(taskEntity.TaskID)
				if cfg == nil {
					logger.ErrorWithZapFields("pass card task cfg is null")
					continue
				}
				if cfg.Auto == 1 {
					err := itemService.AddItems(p.playerModel, cfg.TaskReward, enum.ITEM_CHANGE_REASON_PASS_CARD_TASK)
					if err != nil {
						logger.ErrorWithZapFields("add pass card task reward error", zap.Error(err))
						continue
					}
				} else {
					continue
				}
				taskFinishCount := v.TaskFinishCount
				if cfg.Repeat == 0 || cfg.Repeat > taskFinishCount[id]+1 {
					taskFinishCount[id]++
					p.UpdateTaskFinishCount(v.PassCardId, taskFinishCount)
					taskModel.ReFreshTask(taskEntity.TaskAttribution, taskEntity.TaskID)
				} else {
					taskCoreId, err := gameConfig.GetCoreTaskId(taskEntity.TaskAttribution, taskEntity.TaskID)
					if err != nil {
						logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", taskEntity.TaskAttribution), zap.Int32("task id", taskEntity.TaskID))
						return
					}
					if taskCoreId == 0 {
						logger.ErrorWithZapFields("task core id is error")
						continue
					}
					taskCoreCfg := gameConfig.GetCoreCfg(taskCoreId)
					if taskCoreCfg == nil {
						logger.ErrorWithZapFields("task core cfg is null")
						continue
					}
					taskModel.UpdateTaskStatus(taskEntity.TaskID, taskCoreCfg.TaskType, taskEntity.TaskAttribution, enum.TaskStatusFinishAndReward)
					taskFinishNum++
				}
			} else if taskEntity.Status == enum.TaskStatusFinishAndReward {
				taskFinishNum++
			}
		}
		if taskFinishNum == len(v.TaskSlotList) {
			p.UpdateStatus(v.PassCardId, enum.TaskStatusFinishAndReward)
			*p.CanUsePassCardSlotList = append(*p.CanUsePassCardSlotList, v.PassCardId)
		}
	}
}

func (p *PassCardTaskModel) AddPassCardTask(passCardId int32) error {
	cfgMap := gameConfig.GetPassTaskMap()[passCardId]
	if cfgMap == nil {
		return errors.New("cfg not found")
	}
	passCardTaskEntity := &PassCardTaskEntity{
		UserId:          p.UserId,
		PassCardId:      passCardId,
		TaskSlotList:    make(tool.JSONInt32Slice, 0),
		TaskFinishCount: make(tool.JSONInt32Slice, 0),
		Status:          enum.TaskStatusUnFinish,
	}
	oldSlotCount := p.MaxSlotId
	slotList := p.TakeSlots(len(cfgMap))
	slotNum := 0
	for _, v := range cfgMap {
		entity := &TaskEntity{
			UserID:          p.UserId,
			TaskID:          v.Id,
			SlotId:          slotList[slotNum],
			TaskAttribution: enum.TaskAffiliationPassCard,
			ProgressData:    0,
			Status:          enum.TaskStatusUnFinish,
			AddTime:         tool.UnixNowMilli(),
			UpdateTime:      0,
		}
		slotNum++
		if entity.SlotId > oldSlotCount {
			err := p.playerModel.TaskModel.AddTask(entity)
			if err != nil {
				return err
			}
		} else {
			oldEntity := p.playerModel.TaskModel.TaskEntityBySlot[entity.SlotId]
			p.playerModel.TaskModel.AddTaskEntityToMemory(entity)
			p.playerModel.TaskModel.UpdateChangedWithNewData(oldEntity, entity)
			oldEntity.Status = enum.TaskStatusDeleteInMemory
		}
		messageSender.SendMessage(p.playerModel, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
			Attribution: entity.TaskAttribution,
			TaskId:      entity.TaskID,
			TaskState:   entity.Status,
			Progress:    entity.ProgressData,
		})
		passCardTaskEntity.TaskSlotList = append(passCardTaskEntity.TaskSlotList, entity.SlotId)
		passCardTaskEntity.TaskFinishCount = append(passCardTaskEntity.TaskFinishCount, 0)
	}
	err := easyDB.CreatePlayerEntity(passCardTaskEntity)
	p.Entities[passCardId] = passCardTaskEntity
	if err != nil {
		return err
	}
	return nil
}

func (p *PassCardTaskModel) ClosePassCardTask(passCardId int32) error {
	passCardTaskEntity := p.Entities[passCardId]
	if passCardTaskEntity.Status == enum.TaskStatusFinishAndReward {
		return nil
	}
	taskModel := p.playerModel.TaskModel
	for _, slotId := range passCardTaskEntity.TaskSlotList {
		taskEntity := taskModel.TaskEntityBySlot[slotId]
		taskCoreId, err := gameConfig.GetCoreTaskId(taskEntity.TaskAttribution, taskEntity.TaskID)
		if err != nil {
			logger.ErrorWithZapFields("get core task id error", zap.Int32("attribution", taskEntity.TaskAttribution), zap.Int32("task id", taskEntity.TaskID))
			return errors.New("get core task id error")
		}
		if taskCoreId == 0 {
			return errors.New("task cfg not found")
		}
		taskCfg := gameConfig.GetCoreCfg(taskCoreId)
		if taskCfg == nil {
			return errors.New("task core not found")
		}
		taskModel.UpdateTaskStatus(taskEntity.TaskID, taskCfg.TaskType, taskEntity.TaskAttribution, enum.TaskStatusFinishAndReward)
	}
	return nil
}

func (p *PassCardTaskModel) UpdateSlotList(passCardId int32, slotList tool.JSONInt32Slice) {
	p.Entities[passCardId].TaskSlotList = slotList
	if p.Changed[passCardId] == nil {
		p.Changed[passCardId] = make(map[string]interface{})
	}
	p.Changed[passCardId]["slots"] = slotList
}

func (p *PassCardTaskModel) UpdateStatus(passCardId int32, status int32) {
	p.Entities[passCardId].Status = status
	if p.Changed[passCardId] == nil {
		p.Changed[passCardId] = make(map[string]interface{})
	}
	p.Changed[passCardId]["status"] = status
}

func (p *PassCardTaskModel) UpdateTaskFinishCount(passCardId int32, finishCount tool.JSONInt32Slice) {
	p.Entities[passCardId].TaskFinishCount = finishCount
	if p.Changed[passCardId] == nil {
		p.Changed[passCardId] = make(map[string]interface{})
	}
	p.Changed[passCardId]["task_finish_count"] = finishCount
}

func (p *PassCardTaskModel) TakeSlots(n int) []int32 {
	res := make([]int32, 0)

	if n <= 0 {
		return res
	}
	if p.CanUsePassCardSlotList != nil && len(*p.CanUsePassCardSlotList) > 0 {

		for len(*p.CanUsePassCardSlotList) > 0 {
			if len(res) >= n {
				break
			}
			passCardId := (*p.CanUsePassCardSlotList)[0]                // 取第一个
			*p.CanUsePassCardSlotList = (*p.CanUsePassCardSlotList)[1:] // 移除第一个
			entity := p.Entities[passCardId]
			slotIdList := entity.TaskSlotList
			need := n - len(res) // 还需要多少个

			// 从这个悬赏取槽位
			if need >= len(slotIdList) {
				// 要取的比有的多，全取
				res = append(res, slotIdList...)
				// 清空这个悬赏的槽位
				p.UpdateSlotList(passCardId, tool.JSONInt32Slice{})
			} else {
				// 要取的比有的少，只取一部分
				for i := 0; i < need; i++ {
					res = append(res, slotIdList[i])
				}
				// 更新剩余的槽位
				remaining := slotIdList[need:]
				p.UpdateSlotList(passCardId, remaining)
				*p.CanUsePassCardSlotList = append(*p.CanUsePassCardSlotList, passCardId)
			}
		}
		if len(res) < n {
			for i := len(res) + 1; i <= n; i++ {
				res = append(res, p.MaxSlotId+1)
				p.MaxSlotId++
			}
		}
	} else {
		for i := len(res) + 1; i <= n; i++ {
			res = append(res, p.MaxSlotId+1)
			p.MaxSlotId++
		}
	}

	return res
}

func LoadPassCardTask(userid int64, player *PlayerModel) (*PassCardTaskModel, error) {
	entities := make(map[int32]*PassCardTaskEntity)
	slots := make([]int32, 0)
	finishTaskSlotList := &slots
	num := 0

	rows, err := easyDB.GetPlayerEntitiesByWhere[PassCardTaskEntity](map[string]interface{}{"user_id": userid})
	if err != nil {
		return NewPassCardTaskModel(entities, player, userid, enum.TaskAffiliationPassCard*10000, finishTaskSlotList), err
	}
	for _, v := range rows {
		if v != nil {
			entities[v.PassCardId] = v
			if (v.Status == enum.TaskStatusTimeOverUnFinish || v.Status == enum.TaskStatusFinishAndReward) && len(v.TaskSlotList) > 0 {
				*finishTaskSlotList = append(*finishTaskSlotList, v.PassCardId)
			}
			num += len(v.TaskSlotList)
		}
	}

	return NewPassCardTaskModel(entities, player, userid, int32(enum.TaskAffiliationPassCard*10000+num), finishTaskSlotList), err
}
