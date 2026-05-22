// Package trial 实现七日试炼玩法：按活动解锁每日任务组、任务领奖、进度里程碑领奖，
// 活动结束时未领进度奖励邮件补发，以及登录时同步清理/补建任务。
package trial

import (
	"errors"
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

// attachmentItemTypeItem 邮件附件条目类型：与 logicCommon / mail 约定一致，1 表示道具。
const (
	attachmentItemTypeItem int32 = mail.AttachmentItemTypeItem
)

// TrialService 七日试炼服务，依赖活动、消息、道具数量查询及邮件（过期补发）。
type TrialService struct {
	activityService logicCommon.GameActivityServiceInterface                     // 判断活动是否开启及开放时间
	messageSender   logicCommon.MessageSenderInterface                           // 向客户端推送消息（本模块部分错误经 controller 下发）
	getItemCount    func(player logicCommon.PlayerInterface, itemId int32) int64 // 查询指定道具数量（进度里程碑用）
	mailService     logicCommon.MailServiceInterface                             // 活动结束补发未领进度奖励
}

// Service 全局单例，由 gameController.InitGameController 中 InitTrialService 注入。
var Service *TrialService

// InitTrialService 初始化全局 TrialService，须在 InitMailService、RegisterItemService 之后调用。
func InitTrialService(activity logicCommon.GameActivityServiceInterface, sender logicCommon.MessageSenderInterface, getItemCount func(logicCommon.PlayerInterface, int32) int64, mailSvc logicCommon.MailServiceInterface) {
	Service = &TrialService{
		activityService: activity,
		messageSender:   sender,
		getItemCount:    getItemCount,
		mailService:     mailSvc,
	}
}

// OnPlayerLoad 登录加载 TrialModel 后：过期进度补邮件、已结束活动清理。
func (s *TrialService) OnPlayerLoad(player *model.PlayerModel) {
	if player == nil || player.User == nil {
		return
	}
	s.ProcessExpiredTrialMailsForPlayer(player)
	allActIDs := gameConfig.GetTrialAllActIDs()
	for _, actID := range allActIDs {
		act := s.activityService.IsActivityOpen(player.User.GetServerId(), actID)
		if act == nil {
			if player.TrialModel.GetInitializedDay(actID) > 0 {
				s.CleanupTrial(player, actID)
			}
		}
	}
}

// mergeTrialMailItems 合并邮件附件中相同道具 ID 的数量，便于一封邮件携带汇总奖励。
func mergeTrialMailItems(items []*logicCommon.MailAttachmentItem) []*logicCommon.MailAttachmentItem {
	if len(items) == 0 {
		return nil
	}
	m := make(map[int32]int32)
	for _, it := range items {
		if it == nil || it.Type != attachmentItemTypeItem || it.ID <= 0 || it.Num <= 0 {
			continue
		}
		m[it.ID] += it.Num
	}
	out := make([]*logicCommon.MailAttachmentItem, 0, len(m))
	for id, num := range m {
		out = append(out, &logicCommon.MailAttachmentItem{Type: attachmentItemTypeItem, ID: id, Num: num})
	}
	return out
}

// GetTrialInfo 在活动开启时拉取试炼面板：补建已解锁天任务，并返回各天任务进度与状态。
func (s *TrialService) GetTrialInfo(player *model.PlayerModel, actID int32) (*pb.TrialInfoResp, error) {
	act := s.activityService.IsActivityOpen(player.User.GetServerId(), actID)
	if act == nil {
		return nil, errors.New("activity not open")
	}

	foremosts := gameConfig.GetTrialForemostsByActID(actID)
	if len(foremosts) == 0 {
		return nil, errors.New("trial config not found")
	}

	taskGroups := getTrialTaskGroups(foremosts)
	unlockedDays := s.calcUnlockedDays(act.GetOpenTime(), int32(len(taskGroups)))

	s.EnsureTasksCreated(player, actID, foremosts, unlockedDays)

	dayInfos := make([]*pb.TrialDayInfo, 0, unlockedDays)
	for i := int32(0); i < unlockedDays; i++ {
		dayInfos = append(dayInfos, s.buildDayInfo(player, i+1, taskGroups[i]))
	}

	return &pb.TrialInfoResp{
		ActId:    actID,
		DayInfos: dayInfos,
		ClaimId:  int32(player.TrialModel.GetClaimId(actID)),
	}, nil
}

// EnsureTasksCreated 为 [initializedDay+1, unlockedDays] 区间内尚未创建的试炼任务写入 TaskModel，
// 并更新 TrialModel 的 InitializedDay（试炼任务归属 TaskAffiliationTrial）。
func (s *TrialService) EnsureTasksCreated(player *model.PlayerModel, actID int32, foremosts []*gameConfig.TrialForemostCfg, unlockedDays int32) {
	initializedDay := player.TrialModel.GetInitializedDay(actID)
	if initializedDay >= unlockedDays {
		return
	}
	taskGroups := getTrialTaskGroups(foremosts)
	totalDays := int32(len(taskGroups))
	for day := initializedDay + 1; day <= unlockedDays && day <= totalDays; day++ {
		taskGroup := taskGroups[day-1]
		tasks := gameConfig.GetTrialTasksByGroup(taskGroup)
		for _, tc := range tasks {
			existing := s.findTrialTaskEntity(player, tc.Id)
			if existing != nil {
				continue
			}
			entity := model.NewTaskEntity(
				player.GetUserId(),
				tc.Id,
				tc.Id,
				enum.TaskAffiliationTrial,
				0,
				enum.TaskStatusUnFinish,
				tool.UnixNowMilli(),
				0,
			)
			if err := player.TaskModel.AddTask(entity); err != nil {
				logger.ErrorBySprintf("[trial] EnsureTasksCreated add task error actID=%d taskGroup=%d trialTaskId=%d: %v", actID, taskGroup, tc.Id, err)
			}
		}
	}
	player.TrialModel.SetInitializedDay(actID, unlockedDays)
}

// ClaimTaskReward 领取单条试炼任务奖励：校验活动、天数解锁、任务完成态后发奖并标记已领奖。
func (s *TrialService) ClaimTaskReward(player *model.PlayerModel, actID int32, trialTaskId int32) (*pb.PushTaskUpdate, error) {
	if actID <= 0 {
		actID = gameConfig.GetTrialActIDByTaskID(trialTaskId)
	}
	if actID <= 0 {
		return nil, errors.New("trial task not in activity")
	}
	if err := s.grantTaskReward(player, actID, trialTaskId); err != nil {
		return nil, err
	}
	entity := s.findTrialTaskEntity(player, trialTaskId)
	if entity == nil {
		return nil, errors.New("task not found")
	}
	return &pb.PushTaskUpdate{
		Attribution: enum.TaskAffiliationTrial,
		TaskId:      trialTaskId,
		TaskState:   entity.Status,
		Progress:    entity.ProgressData,
	}, nil
}

func (s *TrialService) grantTaskReward(player *model.PlayerModel, actID int32, trialTaskId int32) error {
	act := s.activityService.IsActivityOpen(player.User.GetServerId(), actID)
	if act == nil {
		return errors.New("activity not open")
	}

	tc := gameConfig.GetTrialTaskCfg(trialTaskId)
	if tc == nil {
		return errors.New("trial task config not found")
	}
	if gameConfig.GetTrialActIDByTaskID(trialTaskId) != actID {
		return errors.New("trial task not in activity")
	}

	if !s.isTaskGroupUnlocked(actID, tc.TaskGroup, act.GetOpenTime()) {
		return errors.New("day not unlocked")
	}

	taskCore := gameConfig.GetCoreCfg(tc.TaskId)
	if taskCore == nil {
		return errors.New("task core is nil")
	}

	taskEntity := s.findTrialTaskEntity(player, trialTaskId)
	if taskEntity == nil {
		return errors.New("task not found")
	}
	if taskEntity.Status == enum.TaskStatusFinishAndReward {
		return errors.New("task already rewarded")
	}
	if taskEntity.Status != enum.TaskStatusFinishUnReward && taskEntity.ProgressData < taskCore.TaskNum {
		return errors.New("task not finished")
	}

	if err := itemService.AddItems(player, tc.TaskReward, enum.ITEM_CHANGE_REASON_TRIAL_TASK_REWARD); err != nil {
		return fmt.Errorf("add task reward error: %w", err)
	}

	player.TaskModel.UpdateTaskStatus(trialTaskId, taskCore.TaskType, enum.TaskAffiliationTrial, enum.TaskStatusFinishAndReward)
	player.TaskModel.UpdateUpdateTime(trialTaskId, taskCore.TaskType, enum.TaskAffiliationTrial, tool.UnixNowMilli())
	operationLogService.OnUserTrialFinishTask(player.GetUserId(), trialTaskId)

	return nil
}

// ClaimProgressReward 按配置顺序领取进度里程碑：进度道具数量达阈值且未领过则发奖并更新 ClaimId。
func (s *TrialService) ClaimProgressReward(player *model.PlayerModel, actID int32) error {
	act := s.activityService.IsActivityOpen(player.User.GetServerId(), actID)
	if act == nil {
		return errors.New("activity not open")
	}

	rewardCfgs := gameConfig.GetTrialRewardsByActID(actID)
	if len(rewardCfgs) == 0 {
		return errors.New("reward config not found")
	}

	claimId := player.TrialModel.GetClaimId(actID)
	addItems := make([]*gameConfig.ItemConfig, 0)
	nextClaimId := claimId

	for _, rc := range rewardCfgs {
		threshold := rc.Value.Num
		if int64(rc.Id) <= claimId {
			continue
		}
		currentNum := s.getItemCount(player, rc.Value.ID)
		if currentNum < threshold {
			continue
		}
		addItems = append(addItems, rc.Reward...)
		nextClaimId = int64(rc.Id)
	}

	if len(addItems) == 0 {
		return errors.New("no reward available")
	}
	if err := itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_TRIAL_PROGRESS_REWARD); err != nil {
		return fmt.Errorf("add progress reward error: %w", err)
	}
	player.TrialModel.SetClaimId(actID, nextClaimId)

	return nil
}

// CleanupTrial 活动已结束或未开启时清理：扣除进度相关道具、移除试炼任务内存与 DB 记录、删除 trial 表行。
func (s *TrialService) CleanupTrial(player *model.PlayerModel, actID int32) {
	clearedItems := make(map[int32]bool)
	rewardCfgs := gameConfig.GetTrialRewardsByActID(actID)
	for _, rc := range rewardCfgs {
		if clearedItems[rc.Value.ID] {
			continue
		}
		clearedItems[rc.Value.ID] = true
		count := s.getItemCount(player, rc.Value.ID)
		if count > 0 {
			if err := itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: rc.Value.ID, Num: count}, enum.ITEM_CHANGE_REASON_TRIAL_PROGRESS_REWARD); err != nil {
				logger.ErrorBySprintf("[trial] CleanupTrial remove progress item error actID=%d itemID=%d: %v", actID, rc.Value.ID, err)
			}
		}
	}

	taskIDs := gameConfig.GetTrialTaskIDsByActID(actID)
	for _, taskID := range taskIDs {
		entity := s.findTrialTaskEntity(player, taskID)
		if entity != nil {
			player.TaskModel.DeteleTaskEntityFormMemory(entity)
		}
	}
	deleteTrialTasks(player.GetUserId(), taskIDs)
	player.TrialModel.RemoveAct(actID)
}

// calcUnlockedDays 根据活动开放时间与当前时间计算已解锁天数（自然日，上限为配置总天数）。
func (s *TrialService) calcUnlockedDays(openTime int64, totalDays int32) int32 {
	now := tool.UnixNowMilli()
	if now < openTime {
		return 0
	}
	days := tool.GetNatureDayDistance(now, openTime) + 1
	if days > totalDays {
		days = totalDays
	}
	return days
}

// isTaskGroupUnlocked 判断某任务组是否已随「第几天」解锁（foremost 顺序对应天序号）。
func (s *TrialService) isTaskGroupUnlocked(actID int32, taskGroup int32, openTime int64) bool {
	foremosts := gameConfig.GetTrialForemostsByActID(actID)
	taskGroups := getTrialTaskGroups(foremosts)
	unlockedDays := s.calcUnlockedDays(openTime, int32(len(taskGroups)))
	for i := int32(0); i < unlockedDays; i++ {
		if taskGroups[i] == taskGroup {
			return true
		}
	}
	return false
}

func (s *TrialService) getTrialProgress(player logicCommon.PlayerInterface, actID int32) int32 {
	if s == nil || s.getItemCount == nil {
		return 0
	}
	rewardCfgs := gameConfig.GetTrialRewardsByActID(actID)
	if len(rewardCfgs) == 0 {
		return 0
	}
	return int32(s.getItemCount(player, rewardCfgs[0].Value.ID))
}

// buildDayInfo 组装某一天（dayIndex）下某任务组内各试炼任务的进度与状态，供协议下发。
func (s *TrialService) buildDayInfo(player *model.PlayerModel, dayIndex int32, taskGroup int32) *pb.TrialDayInfo {
	tasks := gameConfig.GetTrialTasksByGroup(taskGroup)
	pbTasks := make([]*pb.TaskInfo, 0, len(tasks))
	for _, tc := range tasks {
		var progress int32
		status := int32(enum.TaskStatusUnFinish)
		entity := s.findTrialTaskEntity(player, tc.Id)
		if entity != nil {
			progress = entity.ProgressData
			status = entity.Status
		}

		pbTasks = append(pbTasks, &pb.TaskInfo{
			TaskId:    tc.Id,
			TaskState: status,
			Progress:  progress,
		})
	}
	return &pb.TrialDayInfo{
		DayIndex: dayIndex,
		Tasks:    pbTasks,
	}
}

func getTrialTaskGroups(foremosts []*gameConfig.TrialForemostCfg) []int32 {
	taskGroups := make([]int32, 0)
	for _, fc := range foremosts {
		if fc == nil {
			continue
		}
		taskGroups = append(taskGroups, fc.TaskGroup...)
	}
	return taskGroups
}

// findTrialTaskEntity 在 TaskModel 中按试炼配置子 id（trialTaskId）查找任务实体。
func (s *TrialService) findTrialTaskEntity(player *model.PlayerModel, trialTaskId int32) *model.TaskEntity {
	attrMap := player.TaskModel.TaskEntity[enum.TaskAffiliationTrial]
	if attrMap == nil {
		return nil
	}
	for _, typeMap := range attrMap {
		if entity, ok := typeMap[trialTaskId]; ok {
			return entity
		}
	}
	return nil
}

// deleteTrialTasks 从数据库删除该玩家指定归属为试炼的任务行（task_attribution = TaskAffiliationTrial）。
func deleteTrialTasks(userId int64, taskIDs []int32) {
	for _, taskID := range taskIDs {
		_ = easyDB.DeletePlayerEntityByWhere[model.TaskEntity](map[string]interface{}{
			"user_id":          userId,
			"task_attribution": enum.TaskAffiliationTrial,
			"task_id":          taskID,
		}, userId)
	}
}

// ================== 邮件相关（活动结束未领进度奖励补发） ==================

// mailAttachmentItemsToItemConfigs 将邮件附件条目转为道具配置切片，供 SendRewardMailByTemplateID 使用。
func mailAttachmentItemsToItemConfigs(items []*logicCommon.MailAttachmentItem) []*gameConfig.ItemConfig {
	if len(items) == 0 {
		return nil
	}
	out := make([]*gameConfig.ItemConfig, 0, len(items))
	for _, it := range items {
		if it == nil || it.ID <= 0 || it.Num <= 0 {
			continue
		}
		out = append(out, &gameConfig.ItemConfig{ID: it.ID, Num: int64(it.Num)})
	}
	return out
}

func (s *TrialService) collectExpiredProgressRewardMailItems(player *model.PlayerModel, actID int32, claimId int64) ([]*logicCommon.MailAttachmentItem, int64) {
	if s == nil || player == nil || player.InventoryModel == nil {
		return nil, claimId
	}
	rewardCfgs := gameConfig.GetTrialRewardsByActID(actID)
	var items []*logicCommon.MailAttachmentItem
	for _, rc := range rewardCfgs {
		threshold := rc.Value.Num
		if int64(rc.Id) <= claimId {
			continue
		}
		currentNum := player.InventoryModel.GetItemCount(rc.Value.ID)
		if currentNum < threshold {
			continue
		}
		for _, item := range rc.Reward {
			items = append(items, &logicCommon.MailAttachmentItem{
				Type: attachmentItemTypeItem,
				ID:   item.ID,
				Num:  int32(item.Num),
			})
		}
		claimId = int64(rc.Id)
	}
	return items, claimId
}

// ProcessExpiredTrialMailsForPlayer 登录时调用：对本服已结束的开服活动，若存在试炼配置且玩家参与过，
// 将未领进度奖励通过 constant「trialMail」指向的邮件模板补发（需在 CleanupTrial 之前执行）。
func (s *TrialService) ProcessExpiredTrialMailsForPlayer(player *model.PlayerModel) {
	if s.mailService == nil || player == nil || player.User == nil {
		return
	}
	templateID := gameConfig.GetTrialExpireMailTemplateID()
	if templateID <= 0 || gameConfig.GetMailContentCfg(templateID) == nil {
		logger.ErrorBySprintf("[trial] mail template invalid templateID=%d", templateID)
		return
	}
	userId := player.GetUserId()
	serverId := player.User.GetServerId()
	now := tool.UnixNowMilli()
	allOpen, err := easyDB.GetServerAllEntities[model.ServerOpenActivityEntity]()
	if err != nil {
		logger.ErrorBySprintf("[trial] get server open activity failed: %v", err)
		return
	}

	seenAct := make(map[int32]struct{})
	for _, open := range allOpen {
		if open == nil || open.OpenServerId != serverId || open.EndTime >= now {
			continue
		}
		actID := open.ActivityId
		if _, dup := seenAct[actID]; dup {
			continue
		}
		seenAct[actID] = struct{}{}
		if len(gameConfig.GetTrialForemostsByActID(actID)) == 0 {
			continue
		}
		if player.TrialModel.GetInitializedDay(actID) == 0 {
			continue
		}
		claimId := player.TrialModel.GetClaimId(actID)
		items, nextClaimId := s.collectExpiredProgressRewardMailItems(player, actID, claimId)
		items = mergeTrialMailItems(items)
		if len(items) == 0 {
			continue
		}
		reward := mailAttachmentItemsToItemConfigs(items)
		_, err = s.mailService.SendRewardMailByTemplateID(userId, templateID, reward, nil, nil)
		if err != nil {
			logger.ErrorBySprintf("[trial] send expire mail failed userId=%d actID=%d: %v", userId, actID, err)
			continue
		}
		player.TrialModel.SetClaimId(actID, nextClaimId)
		player.TrialModel.SaveModelToDB()
	}
}
