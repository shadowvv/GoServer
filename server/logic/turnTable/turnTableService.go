package turnTable

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

type TurnTableService struct {
	activityService logicCommon.GameActivityServiceInterface
	messageSender   logicCommon.MessageSenderInterface
}

var Service *TurnTableService

const (
	turnTableUsuallyTypeDraw    int32 = 1
	turnTableUsuallyTypeConsume int32 = 2
)

func InitTurnTableService(activity logicCommon.GameActivityServiceInterface, sender logicCommon.MessageSenderInterface) {
	Service = &TurnTableService{
		activityService: activity,
		messageSender:   sender,
	}
}

func (s *TurnTableService) GetDetail(player *model.PlayerModel, modId int32) (*pb.TurnTableInfo, error) {
	_, mainCfg, err := s.prepare(player, modId)
	if err != nil {
		return nil, err
	}

	typeList := getUsuallyUseTypeList(modId)
	progressList := make([]*pb.TurnTableUsuallyProgress, 0, len(typeList))
	drawProgressEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallyProgress, turnTableUsuallyTypeDraw)
	if err != nil {
		return nil, err
	}
	drawCount := int32(drawProgressEntity.Progress)
	for typeIndex, typ := range typeList {
		progressEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallyProgress, typ)
		if err != nil {
			return nil, err
		}
		claimedMaxId := int32(0)
		loopClaimCount := int32(0)
		cfgs := gameConfig.GetUsuallyUseCfgsByModIdAndType(modId, typ)
		if progressEntity.Count > 0 && len(cfgs) > 0 {
			claimedIndex := int(progressEntity.Count) - 1
			if claimedIndex >= len(cfgs) {
				claimedIndex = len(cfgs) - 1
			}
			claimedMaxId = cfgs[claimedIndex].Id
		}
		if len(typeList) == 1 && typeIndex == 0 {
			if stateEntity := player.TurnTableModel.GetState(modId, model.TurnTableStateTypeUsuallySingleReward, typ); stateEntity != nil {
				loopClaimCount = stateEntity.Count
			}
		} else if len(cfgs) > 0 && cfgs[len(cfgs)-1].LoopLayer > 0 && progressEntity.Count >= int32(len(cfgs)) {
			loopClaimCount = progressEntity.Count - int32(len(cfgs)) + 1
		}
		progressList = append(progressList, &pb.TurnTableUsuallyProgress{
			Type:           typ,
			Progress:       progressEntity.Progress,
			ClaimedMaxId:   claimedMaxId,
			LoopClaimCount: loopClaimCount,
		})
	}

	taskList := make([]*pb.TaskInfo, 0)
	for _, taskCfg := range gameConfig.GetActTaskCfgsByActID(mainCfg.ActId) {
		taskEntity := player.TaskModel.TaskEntityBySlot[model.TurnTableTaskSlot(taskCfg.Id)]
		if taskEntity == nil {
			taskList = append(taskList, &pb.TaskInfo{TaskId: taskCfg.Id, TaskState: enum.TaskStatusUnFinish, Progress: 0})
			continue
		}
		taskList = append(taskList, &pb.TaskInfo{TaskId: taskCfg.Id, TaskState: taskEntity.Status, Progress: taskEntity.ProgressData})
	}

	limitList := make([]*pb.TurnTableRewardLimitInfo, 0)
	guaranteeCountMap := make(map[int32]int32)
	for _, cfg := range gameConfig.GetTurnTableCfgsByModId(modId) {
		if cfg.Limit > 0 {
			hitCount := int32(0)
			if stateEntity := player.TurnTableModel.GetState(modId, model.TurnTableStateTypeRewardLimit, cfg.Id); stateEntity != nil {
				hitCount = stateEntity.Count
			}
			limitList = append(limitList, &pb.TurnTableRewardLimitInfo{RewardId: cfg.Id, HitCount: hitCount})
		}
		if len(cfg.MinimumGuarantee) == 2 {
			if stateEntity := player.TurnTableModel.GetState(modId, model.TurnTableStateTypeGuarantee, cfg.Id); stateEntity != nil {
				guaranteeCountMap[cfg.Id] = stateEntity.Count
			}
		}
	}

	return &pb.TurnTableInfo{
		ModId:               modId,
		ActId:               mainCfg.ActId,
		DrawCount:           drawCount,
		UsuallyProgressList: progressList,
		TaskList:            taskList,
		RewardLimitList:     limitList,
		GuaranteeCountMap:   guaranteeCountMap,
	}, nil
}

func (s *TurnTableService) Draw(player *model.PlayerModel, modId int32, count int32) (map[int32]int32, error) {
	if count != 1 && count != 10 && count != 50 {
		return nil, errors.New("turn table count error")
	}
	_, mainCfg, err := s.prepare(player, modId)
	if err != nil {
		return nil, err
	}
	act := s.activityService.IsActivityOpen(player.GetUserServerId(), mainCfg.ActId)
	if act == nil {
		return nil, errors.New("activity not open")
	}
	if tool.UnixNowMilli() >= act.GetSettleTime() {
		return nil, errors.New("activity settled")
	}

	cost := &gameConfig.ItemConfig{ID: mainCfg.Use.ID, Num: mainCfg.Use.Num * int64(count)}
	if ok, err := itemService.CheckItemCount(player, cost); !ok || err != nil {
		return nil, errors.New("item not enough")
	}

	drawProgress, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallyProgress, turnTableUsuallyTypeDraw)
	if err != nil {
		return nil, err
	}
	consumeProgress, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallyProgress, turnTableUsuallyTypeConsume)
	if err != nil {
		return nil, err
	}
	addItems := make([]*gameConfig.ItemConfig, 0)
	pool := gameConfig.GetTurnTableCfgsByModId(modId)
	limitMap := make(map[int32]int32)
	guaranteeMap := make(map[int32]int32)
	for _, cfg := range pool {
		if cfg.Limit > 0 {
			if stateEntity := player.TurnTableModel.GetState(modId, model.TurnTableStateTypeRewardLimit, cfg.Id); stateEntity != nil {
				limitMap[cfg.Id] = stateEntity.Count
			}
		}
		if len(cfg.MinimumGuarantee) == 2 {
			if stateEntity := player.TurnTableModel.GetState(modId, model.TurnTableStateTypeGuarantee, cfg.Id); stateEntity != nil {
				guaranteeMap[cfg.Id] = stateEntity.Count
			}
		}
	}
	limitChanged := make(map[int32]int32)
	guaranteeChanged := make(map[int32]int32)
	drawCountMap := make(map[int32]int32)
	for i := range count {
		var rewardCfg *gameConfig.TurnTableCfg
		nextDraw := int32(drawProgress.Progress) + i + 1
		for _, cfg := range pool {
			if len(cfg.MinimumGuarantee) != 2 {
				continue
			}
			guaranteeCount := cfg.MinimumGuarantee[0]
			guaranteeTimes := cfg.MinimumGuarantee[1]
			usedTimes := guaranteeMap[cfg.Id]
			leftDraw := guaranteeCount - nextDraw + 1
			leftTimes := guaranteeTimes - usedTimes
			if nextDraw <= guaranteeCount && leftTimes > 0 && leftDraw <= leftTimes && (cfg.Limit <= 0 || limitMap[cfg.Id] < cfg.Limit) {
				rewardCfg = cfg
				guaranteeMap[cfg.Id]++
				guaranteeChanged[cfg.Id] = guaranteeMap[cfg.Id]
				break
			}
		}
		if rewardCfg == nil {
			ids := make([]int32, 0)
			weights := make([]int32, 0)
			cfgMap := make(map[int32]*gameConfig.TurnTableCfg)
			for _, cfg := range pool {
				if cfg.Limit > 0 && limitMap[cfg.Id] >= cfg.Limit {
					continue
				}
				ids = append(ids, cfg.Id)
				weights = append(weights, cfg.Weight)
				cfgMap[cfg.Id] = cfg
			}
			if len(ids) == 0 {
				return nil, errors.New("pool empty")
			}
			rewardCfg = cfgMap[gameConfig.WeightedRandomChoice(ids, weights)]
		}
		items := gameConfig.Drop(rewardCfg.Drop)
		if len(items) == 0 {
			return nil, errors.New("cfg not found")
		}
		addItems = append(addItems, items...)
		drawCountMap[rewardCfg.Id]++
		if rewardCfg.Limit > 0 {
			limitMap[rewardCfg.Id]++
			limitChanged[rewardCfg.Id] = limitMap[rewardCfg.Id]
		}
	}

	if err := itemService.RemoveItem(player, cost, enum.ITEM_CHANGE_REASON_TURN_TABLE_REWARD); err != nil {
		return nil, err
	}
	if err := itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_TURN_TABLE_REWARD); err != nil {
		return nil, err
	}

	player.TurnTableModel.AddStateProgress(drawProgress, int64(count))
	player.TurnTableModel.AddStateProgress(consumeProgress, cost.Num)
	for rewardId, hitCount := range limitChanged {
		stateEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeRewardLimit, rewardId)
		if err != nil {
			return nil, err
		}
		player.TurnTableModel.SetStateCount(stateEntity, hitCount)
	}
	for rewardId, guaranteeCount := range guaranteeChanged {
		stateEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeGuarantee, rewardId)
		if err != nil {
			return nil, err
		}
		player.TurnTableModel.SetStateCount(stateEntity, guaranteeCount)
	}
	return drawCountMap, nil
}

func (s *TurnTableService) ClaimUsuallyReward(player *model.PlayerModel, modId int32, typ int32, rewardId int32) error {
	_, _, err := s.prepare(player, modId)
	if err != nil {
		return err
	}

	typeList := getUsuallyUseTypeList(modId)
	if len(typeList) == 0 {
		return errors.New("cfg not found")
	}

	typeIndex := -1
	for i, v := range typeList {
		if v == typ {
			typeIndex = i
			break
		}
	}
	if typeIndex < 0 ||
		(typeIndex > 0 && rewardId == 0) {
		return errors.New("invalid request")
	}

	cfgs := gameConfig.GetUsuallyUseCfgsByModIdAndType(modId, typ)
	if len(cfgs) == 0 {
		return errors.New("cfg not found")
	}
	progressEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallyProgress, typ)
	if err != nil {
		return err
	}
	progress := progressEntity.Progress
	claimedCount := progressEntity.Count
	claimedEntity := progressEntity
	items := make([]*gameConfig.ItemConfig, 0)

	if rewardId == 0 {
		if typeIndex > 0 {
			return errors.New("invalid request")
		}
		autoClaimCount := len(cfgs)
		if len(typeList) == 1 {
			autoClaimCount = len(cfgs) - 1
		}
		for int(claimedCount) < autoClaimCount {
			cfg := cfgs[int(claimedCount)]
			if progress < int64(cfg.Param) {
				break
			}
			items = append(items, cfg.Drop...)
			claimedCount++
		}
	} else if len(typeList) == 1 && typeIndex == 0 {
		claimCfg := cfgs[len(cfgs)-1]
		if claimCfg.Id != rewardId {
			return errors.New("reward order error")
		}
		singleRewardEntity, err := player.TurnTableModel.GetOrCreateState(modId, model.TurnTableStateTypeUsuallySingleReward, typ)
		if err != nil {
			return err
		}
		if claimCfg.LoopLayer <= 0 && singleRewardEntity.Count > 0 {
			return errors.New("reward already claimed")
		}
		needProgress := int64(claimCfg.Param) * int64(singleRewardEntity.Count+1)
		if progress < needProgress {
			return errors.New("reward not reach")
		}
		items = append(items, claimCfg.Drop...)
		claimedCount = singleRewardEntity.Count + 1
		claimedEntity = singleRewardEntity
	} else {
		var claimCfg *gameConfig.UsuallyUseCfg
		if int(claimedCount) < len(cfgs) {
			claimCfg = cfgs[int(claimedCount)]
		} else {
			lastCfg := cfgs[len(cfgs)-1]
			if lastCfg.LoopLayer > 0 {
				claimCfg = lastCfg
			}
		}
		if claimCfg == nil {
			return errors.New("reward already claimed")
		}
		if claimCfg.Id != rewardId {
			return errors.New("reward order error")
		}
		needProgress := int64(claimCfg.Param)
		if int(claimedCount) >= len(cfgs) {
			needProgress = int64(claimCfg.Param) * int64(claimedCount-int32(len(cfgs))+2)
		}
		if progress < needProgress {
			return errors.New("reward not reach")
		}
		items = append(items, claimCfg.Drop...)
		claimedCount++
	}

	if len(items) == 0 {
		return errors.New("reward not reach")
	}
	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_LOTTERY); err != nil {
		return err
	}
	player.TurnTableModel.SetStateCount(claimedEntity, claimedCount)
	return nil
}

func getUsuallyUseTypeList(modId int32) []int32 {
	typeList := make([]int32, 0)
	typeExist := make(map[int32]struct{})
	for _, cfg := range gameConfig.GetUsuallyUseCfgsByModId(modId) {
		if _, ok := typeExist[cfg.Type]; ok {
			continue
		}
		typeExist[cfg.Type] = struct{}{}
		typeList = append(typeList, cfg.Type)
	}
	return typeList
}

func (s *TurnTableService) ClaimTaskReward(player *model.PlayerModel, actTaskId int32) (*pb.PushTaskUpdate, error) {
	cfg := gameConfig.GetActTaskCfg(actTaskId)
	if cfg == nil {
		return nil, errors.New("cfg not found")
	}
	mainCfgs := gameConfig.GetTurnTableMainCfgsByActId(cfg.ActId)
	if len(mainCfgs) == 0 {
		return nil, errors.New("cfg not found")
	}
	if _, _, err := s.prepare(player, mainCfgs[0].Id); err != nil {
		return nil, err
	}
	coreCfg := gameConfig.GetCoreCfg(cfg.TaskId)
	if coreCfg == nil {
		return nil, errors.New("cfg not found")
	}
	entity := player.TaskModel.TaskEntityBySlot[model.TurnTableTaskSlot(actTaskId)]
	if entity == nil {
		return nil, errors.New("task not found")
	}
	if entity.Status == enum.TaskStatusFinishAndReward {
		return nil, errors.New("task already rewarded")
	}
	if entity.Status != enum.TaskStatusFinishUnReward && entity.ProgressData < coreCfg.TaskNum {
		return nil, errors.New("task not finish")
	}
	if err := itemService.AddItems(player, cfg.TaskReward, enum.ITEM_CHANGE_REASON_TASK); err != nil {
		return nil, err
	}
	player.TaskModel.UpdateTaskStatus(actTaskId, coreCfg.TaskType, enum.TaskAffiliationAct, enum.TaskStatusFinishAndReward)
	player.TaskModel.UpdateUpdateTime(actTaskId, coreCfg.TaskType, enum.TaskAffiliationAct, tool.UnixNowMilli())
	return &pb.PushTaskUpdate{
		Attribution: enum.TaskAffiliationAct,
		TaskId:      actTaskId,
		TaskState:   enum.TaskStatusFinishAndReward,
		Progress:    entity.ProgressData,
	}, nil
}

func (s *TurnTableService) prepare(player *model.PlayerModel, modId int32) (*model.TurnTableEntity, *gameConfig.TurnTableMainCfg, error) {
	mainCfg := gameConfig.GetTurnTableMainCfg(modId)
	if mainCfg == nil {
		return nil, nil, errors.New("cfg not found")
	}
	act := s.activityService.IsActivityOpen(player.GetUserServerId(), mainCfg.ActId)
	if act == nil {
		return nil, nil, errors.New("activity not open")
	}
	if player.TurnTableModel == nil {
		turnTableModel, err := model.LoadTurnTableModel(player)
		if err != nil {
			return nil, nil, err
		}
		player.TurnTableModel = turnTableModel
		player.AppendPlayerModel(turnTableModel)
	}
	entity, created, err := player.TurnTableModel.GetOrCreate(modId)
	if err != nil {
		return nil, nil, err
	}
	needReset := !created && entity.TaskRefreshTime < act.GetOpenTime()
	if needReset {
		player.TurnTableModel.Reset(modId)
	}
	pushes, err := player.TurnTableModel.SyncActTasks(modId, created || needReset)
	if err != nil {
		return nil, nil, err
	}
	for _, push := range pushes {
		s.messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
	}
	return entity, mainCfg, nil
}
