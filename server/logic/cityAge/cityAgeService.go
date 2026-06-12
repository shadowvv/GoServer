package cityAge

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type CityAgeService struct{}

var Service = NewCityAgeService()
var messageSender logicCommon.MessageSenderInterface

const (
	cityAgeDailyRewardStatusUnclaimed int32 = 0
	cityAgeDailyRewardStatusClaimed   int32 = 1
)

func NewCityAgeService() *CityAgeService {
	return &CityAgeService{}
}

func InitCityAgeService(sender logicCommon.MessageSenderInterface) {
	messageSender = sender
}

func cityAgeTaskSlot(groupIndex, taskIndex int32) int32 {
	return enum.TaskAffiliationCityAge*10000 + groupIndex*100 + taskIndex
}

func (s *CityAgeService) GetOrLoadModel(player *model.PlayerModel, forceReset bool) (*model.CityAgeModel, bool) {
	if player == nil || player.User == nil || player.TaskModel == nil {
		return nil, false
	}

	if player.CityAgeModel == nil {
		cityAgeModel, err := model.LoadCityAgeModel(player.GetUserId())
		if err != nil {
			platformLogger.ErrorWithUser("load city age model error", player, err)
			return nil, false
		}
		player.CityAgeModel = cityAgeModel
		player.AppendPlayerModel(cityAgeModel)
	}

	if err := player.CityAgeModel.EnsureInit(); err != nil {
		platformLogger.ErrorWithUser("init city age model error", player, err)
		return nil, false
	}
	if err := s.syncCurrentAgeTasks(player, player.CityAgeModel, forceReset); err != nil {
		platformLogger.ErrorWithUser("sync city age tasks error", player, err)
		return nil, false
	}
	return player.CityAgeModel, true
}

func (s *CityAgeService) GetDetail(player *model.PlayerModel) (*pb.CityAgeInfo, bool) {
	cityAgeModel, ok := s.GetOrLoadModel(player, false)
	if !ok {
		return nil, false
	}
	cfg := cityAgeModel.GetCurrentCfg()
	if cfg == nil {
		return nil, false
	}
	firstReachList, err := s.loadFirstReachInfoList(player.GetUserServerId())
	if err != nil {
		logger.ErrorBySprintf("load city age first reach list error serverID:%d err:%v", player.GetUserServerId(), err)
	}
	return s.buildInfoWithCfg(player, cityAgeModel, cfg, firstReachList), true
}

func (s *CityAgeService) ClaimGroupReward(player *model.PlayerModel, groupIndex int32) ([]*pb.PushTaskUpdate, error) {
	cityAgeModel, ok := s.GetOrLoadModel(player, false)
	if !ok {
		return nil, errors.New("system error")
	}
	cfg := cityAgeModel.GetCurrentCfg()
	if cfg == nil {
		return nil, errors.New("city age cfg not found")
	}
	if groupIndex < 0 || groupIndex > model.CityAgeGroupCount {
		return nil, errors.New("group index error")
	}

	if groupIndex > 0 {
		if cityAgeModel.GetGroupRewardStatus(groupIndex) == 1 {
			return nil, errors.New("group reward already claimed")
		}

		dropID := cfg.GetGroupDrop(groupIndex)
		if dropID <= 0 {
			return nil, errors.New("group reward not found")
		}
		if !s.isGroupFinished(player, cfg, groupIndex) {
			return nil, errors.New("task not finish")
		}

		items := gameConfig.Drop(dropID)
		if len(items) == 0 {
			return nil, errors.New("group reward not found")
		}
		if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_CITY_AGE); err != nil {
			return nil, err
		}

		statuses := cityAgeModel.GetGroupRewardStatuses()
		statuses[groupIndex-1] = 1
		cityAgeModel.SetGroupRewardStatus(statuses)

		return s.markGroupTasksRewarded(player, cfg, groupIndex), nil
	}

	groupIndexes := make([]int32, 0, model.CityAgeGroupCount)
	items := make([]*gameConfig.ItemConfig, 0)
	hasUnclaimed := false
	for idx := int32(1); idx <= model.CityAgeGroupCount; idx++ {
		if len(cfg.GetGroupTasks(idx)) == 0 && cfg.GetGroupDrop(idx) == 0 {
			continue
		}
		if cityAgeModel.GetGroupRewardStatus(idx) == 1 {
			continue
		}
		hasUnclaimed = true
		if !s.isGroupFinished(player, cfg, idx) {
			continue
		}
		dropID := cfg.GetGroupDrop(idx)
		if dropID <= 0 {
			return nil, errors.New("group reward not found")
		}
		dropItems := gameConfig.Drop(dropID)
		if len(dropItems) == 0 {
			return nil, errors.New("group reward not found")
		}
		groupIndexes = append(groupIndexes, idx)
		items = append(items, dropItems...)
	}
	if len(groupIndexes) == 0 {
		if hasUnclaimed {
			return nil, errors.New("task not finish")
		}
		return nil, errors.New("group reward already claimed")
	}
	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_CITY_AGE); err != nil {
		return nil, err
	}

	statuses := cityAgeModel.GetGroupRewardStatuses()
	pushes := make([]*pb.PushTaskUpdate, 0)
	for _, idx := range groupIndexes {
		statuses[idx-1] = 1
		pushes = append(pushes, s.markGroupTasksRewarded(player, cfg, idx)...)
	}
	cityAgeModel.SetGroupRewardStatus(statuses)
	return pushes, nil
}

func (s *CityAgeService) Upgrade(player *model.PlayerModel) (*pb.CityAgeInfo, []*pb.PushTaskUpdate, error) {
	cityAgeModel, ok := s.GetOrLoadModel(player, false)
	if !ok {
		return nil, nil, errors.New("system error")
	}
	cfg := cityAgeModel.GetCurrentCfg()
	if cfg == nil {
		return nil, nil, errors.New("city age cfg not found")
	}
	for groupIndex := int32(1); groupIndex <= model.CityAgeGroupCount; groupIndex++ {
		if len(cfg.GetGroupTasks(groupIndex)) == 0 && cfg.GetGroupDrop(groupIndex) == 0 {
			continue
		}
		if cityAgeModel.GetGroupRewardStatus(groupIndex) != 1 {
			return nil, nil, errors.New("group reward not all claimed")
		}
	}

	nextCfg := gameConfig.GetNextCityAgeUpCfg(cfg.Id)
	if nextCfg == nil {
		return nil, nil, errors.New("city age max")
	}
	if cfg.DropAge <= 0 {
		return nil, nil, errors.New("upgrade reward not found")
	}

	items := gameConfig.Drop(cfg.DropAge)
	if len(items) == 0 {
		return nil, nil, errors.New("upgrade reward not found")
	}
	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_CITY_AGE); err != nil {
		return nil, nil, err
	}

	cityAgeModel.SetAgeId(nextCfg.Id)
	cityAgeModel.ResetGroupRewardStatus()
	if err := s.syncCurrentAgeTasks(player, cityAgeModel, true); err != nil {
		return nil, nil, errors.New("system error")
	}
	reachEntity := &model.CityAgeFirstReachEntity{
		ServerId:  player.GetUserServerId(),
		CityAgeId: nextCfg.Id,
		UserId:    player.GetUserId(),
		ReachTime: tool.UnixNowMilli(),
	}
	if ok, err := model.TryCreateCityAgeFirstReach(reachEntity); err != nil {
		platformLogger.ErrorWithUser("record city age first reach error", player, err)
	} else if ok && messageSender != nil {
		messageSender.Broadcast(pb.MESSAGE_ID_PUSH_CITY_AGE_FIRST_REACH, &pb.PushCityAgeFirstReach{
			Info: s.buildFirstReachInfo(reachEntity, player.User.Entity),
		}, enum.BROADCAST_TYPE_SERVER_ID, int64(player.GetUserServerId()))
	}

	firstReachList, err := s.loadFirstReachInfoList(player.GetUserServerId())
	if err != nil {
		logger.ErrorBySprintf("load city age first reach list error serverID:%d err:%v", player.GetUserServerId(), err)
	}

	pushes := make([]*pb.PushTaskUpdate, 0)
	for groupIndex := int32(1); groupIndex <= model.CityAgeGroupCount; groupIndex++ {
		for taskIndex, taskID := range nextCfg.GetGroupTasks(groupIndex) {
			taskInfo := s.buildTaskInfo(player, taskID, groupIndex, int32(taskIndex+1))
			pushes = append(pushes, &pb.PushTaskUpdate{
				Attribution: enum.TaskAffiliationCityAge,
				TaskId:      taskInfo.TaskId,
				TaskState:   taskInfo.TaskState,
				Progress:    taskInfo.Progress,
			})
		}
	}
	return s.buildInfoWithCfg(player, cityAgeModel, nextCfg, firstReachList), pushes, nil
}

func (s *CityAgeService) ClaimDailyReward(player *model.PlayerModel) error {
	cityAgeModel, ok := s.GetOrLoadModel(player, false)
	if !ok {
		return errors.New("system error")
	}
	cfg := cityAgeModel.GetCurrentCfg()
	if cfg == nil {
		return errors.New("city age cfg not found")
	}
	if cfg.Drop2 <= 0 {
		return errors.New("daily reward not found")
	}
	if cityAgeModel.Entity.DailyRewardTime > 0 &&
		tool.IsSameDayByMilli(cityAgeModel.Entity.DailyRewardTime, tool.UnixNowMilli()) {
		return errors.New("daily reward already claimed")
	}

	items := gameConfig.Drop(cfg.Drop2)
	if len(items) == 0 {
		return errors.New("daily reward not found")
	}
	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_CITY_AGE); err != nil {
		return err
	}

	cityAgeModel.SetDailyRewardTime(tool.UnixNowMilli())
	return nil
}

func (s *CityAgeService) syncCurrentAgeTasks(player *model.PlayerModel, cityAgeModel *model.CityAgeModel, forceReset bool) error {
	cfg := cityAgeModel.GetCurrentCfg()
	if cfg == nil {
		return errors.New("city age cfg not found")
	}

	now := tool.UnixNowMilli()
	expectedSlots := make(map[int32]struct{})
	for groupIndex := int32(1); groupIndex <= model.CityAgeGroupCount; groupIndex++ {
		for taskIndex, taskID := range cfg.GetGroupTasks(groupIndex) {
			taskEntity := model.NewTaskEntity(
				player.GetUserId(),
				cityAgeTaskSlot(groupIndex, int32(taskIndex+1)),
				taskID,
				enum.TaskAffiliationCityAge,
				0,
				enum.TaskStatusUnFinish,
				now,
				0,
			)
			expectedSlots[taskEntity.SlotId] = struct{}{}

			oldEntity := player.TaskModel.TaskEntityBySlot[taskEntity.SlotId]
			if !forceReset && oldEntity != nil && oldEntity.TaskAttribution == enum.TaskAffiliationCityAge && oldEntity.TaskID == taskEntity.TaskID {
				player.TaskModel.NeedCheckTaskList = append(player.TaskModel.NeedCheckTaskList, oldEntity)
				continue
			}
			if oldEntity != nil {
				player.TaskModel.UpdateChangedWithNewData(oldEntity, taskEntity)
				player.TaskModel.DeteleTaskEntityFormMemory(oldEntity)
				player.TaskModel.AddTaskEntityToMemory(taskEntity)
				continue
			}
			player.TaskModel.AddTaskEntityToMemory(taskEntity)
			if err := easyDB.CreatePlayerEntity(taskEntity); err != nil {
				return err
			}
		}
	}

	for slotID, entity := range player.TaskModel.TaskEntityBySlot {
		if entity == nil || entity.TaskAttribution != enum.TaskAffiliationCityAge {
			continue
		}
		if _, ok := expectedSlots[slotID]; ok {
			continue
		}
		player.TaskModel.DeteleTaskEntityFormMemory(entity)
		delete(player.TaskModel.TaskEntityBySlot, slotID)
		if err := easyDB.DeletePlayerEntityByWhere[model.TaskEntity](map[string]interface{}{
			"user_id":          player.GetUserId(),
			"slot_id":          slotID,
			"task_attribution": entity.TaskAttribution,
		}, player.GetUserId()); err != nil {
			return err
		}
	}
	return nil
}

func (s *CityAgeService) buildInfoWithCfg(player *model.PlayerModel, cityAgeModel *model.CityAgeModel, cfg *gameConfig.CityAgeUpCfg, firstReachList []*pb.CityAgeFirstReachInfo) *pb.CityAgeInfo {
	groupList := make([]*pb.CityAgeTaskGroup, 0, model.CityAgeGroupCount)
	for groupIndex := int32(1); groupIndex <= model.CityAgeGroupCount; groupIndex++ {
		taskIDs := cfg.GetGroupTasks(groupIndex)
		taskList := make([]*pb.TaskInfo, 0, len(taskIDs))
		for taskIndex, taskID := range taskIDs {
			taskList = append(taskList, s.buildTaskInfo(player, taskID, groupIndex, int32(taskIndex+1)))
		}
		groupList = append(groupList, &pb.CityAgeTaskGroup{
			GroupIndex:   groupIndex,
			RewardStatus: cityAgeModel.GetGroupRewardStatus(groupIndex),
			TaskList:     taskList,
		})
	}

	dailyRewardStatus := cityAgeDailyRewardStatusUnclaimed
	if cfg.Drop2 <= 0 {
		dailyRewardStatus = cityAgeDailyRewardStatusClaimed
	} else if cityAgeModel.Entity.DailyRewardTime > 0 &&
		tool.IsSameDayByMilli(cityAgeModel.Entity.DailyRewardTime, tool.UnixNowMilli()) {
		dailyRewardStatus = cityAgeDailyRewardStatusClaimed
	}

	return &pb.CityAgeInfo{
		CityAgeId:         cfg.Id,
		GroupList:         groupList,
		FirstReachList:    firstReachList,
		DailyRewardStatus: dailyRewardStatus,
	}
}

func (s *CityAgeService) isGroupFinished(player *model.PlayerModel, cfg *gameConfig.CityAgeUpCfg, groupIndex int32) bool {
	taskIDs := cfg.GetGroupTasks(groupIndex)
	if len(taskIDs) == 0 {
		return false
	}

	for taskIndex, taskID := range taskIDs {
		slotID := cityAgeTaskSlot(groupIndex, int32(taskIndex+1))
		entity := player.TaskModel.TaskEntityBySlot[slotID]
		if entity == nil || entity.TaskAttribution != enum.TaskAffiliationCityAge || entity.TaskID != taskID {
			return false
		}
		coreCfg := gameConfig.GetCoreCfg(taskID)
		if coreCfg == nil {
			return false
		}
		if entity.Status == enum.TaskStatusFinishUnReward || entity.Status == enum.TaskStatusFinishAndReward {
			continue
		}
		if entity.ProgressData < coreCfg.TaskNum {
			return false
		}
	}
	return true
}

func (s *CityAgeService) markGroupTasksRewarded(player *model.PlayerModel, cfg *gameConfig.CityAgeUpCfg, groupIndex int32) []*pb.PushTaskUpdate {
	pushes := make([]*pb.PushTaskUpdate, 0)
	now := tool.UnixNowMilli()
	for taskIndex, taskID := range cfg.GetGroupTasks(groupIndex) {
		slotID := cityAgeTaskSlot(groupIndex, int32(taskIndex+1))
		entity := player.TaskModel.TaskEntityBySlot[slotID]
		if entity == nil || entity.TaskAttribution != enum.TaskAffiliationCityAge || entity.TaskID != taskID {
			continue
		}
		entity.Status = enum.TaskStatusFinishAndReward
		entity.UpdateTime = now
		changed := player.TaskModel.Changed[enum.TaskAffiliationCityAge]
		if changed == nil {
			changed = make(map[int32]map[string]interface{})
			player.TaskModel.Changed[enum.TaskAffiliationCityAge] = changed
		}
		if changed[entity.SlotId] == nil {
			changed[entity.SlotId] = make(map[string]interface{})
		}
		changed[entity.SlotId]["status"] = entity.Status
		changed[entity.SlotId]["update_time"] = entity.UpdateTime
		pushes = append(pushes, &pb.PushTaskUpdate{
			Attribution: enum.TaskAffiliationCityAge,
			TaskId:      taskID,
			TaskState:   enum.TaskStatusFinishAndReward,
			Progress:    entity.ProgressData,
		})
	}
	return pushes
}

func (s *CityAgeService) loadFirstReachInfoList(serverID int32) ([]*pb.CityAgeFirstReachInfo, error) {
	entities, err := model.LoadCityAgeFirstReachList(serverID)
	if err != nil {
		return nil, err
	}
	res := make([]*pb.CityAgeFirstReachInfo, 0, len(entities))
	for _, entity := range entities {
		userEntity, _ := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{
			"user_id":   entity.UserId,
			"server_id": entity.ServerId,
		})
		res = append(res, s.buildFirstReachInfo(entity, userEntity))
	}
	return res, nil
}

func (s *CityAgeService) buildFirstReachInfo(entity *model.CityAgeFirstReachEntity, userEntity *model.UserEntity) *pb.CityAgeFirstReachInfo {
	playerInfo := &pb.PlayerBasicInfo{
		UserId:   entity.UserId,
		ServerId: entity.ServerId,
	}
	if userEntity != nil {
		playerInfo.NickName = userEntity.Nickname
		playerInfo.HeadId = userEntity.HeadId
		playerInfo.HeadFrameId = userEntity.HeadFrameId
		playerInfo.TitleId = userEntity.TitleId
		playerInfo.Level = userEntity.Level
	}
	return &pb.CityAgeFirstReachInfo{
		CityAgeId:  entity.CityAgeId,
		PlayerInfo: playerInfo,
		ReachTime:  entity.ReachTime,
	}
}

func (s *CityAgeService) buildTaskInfo(player *model.PlayerModel, taskID int32, groupIndex int32, taskIndex int32) *pb.TaskInfo {
	slotID := cityAgeTaskSlot(groupIndex, taskIndex)
	entity := player.TaskModel.TaskEntityBySlot[slotID]
	if entity == nil || entity.TaskAttribution != enum.TaskAffiliationCityAge || entity.TaskID != taskID {
		return &pb.TaskInfo{
			TaskId:    taskID,
			TaskState: enum.TaskStatusUnFinish,
			Progress:  0,
		}
	}
	return &pb.TaskInfo{
		TaskId:    entity.TaskID,
		TaskState: entity.Status,
		Progress:  entity.ProgressData,
	}
}
