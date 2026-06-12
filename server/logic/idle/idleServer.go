// File: idleServer.go
// Description: 挂机奖励系统服务实现
// Author: 木村凉太
// Create Time: 2026.02

package idle

import (
	"encoding/json"
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
)

type IdleServer struct {
	unlockService logicCommon.UnlockServiceInterface
}

func NewIdleServer(unlockService logicCommon.UnlockServiceInterface) *IdleServer {
	return &IdleServer{
		unlockService: unlockService,
	}
}

// GetIdleInfo 获取挂机信息
func (s *IdleServer) GetIdleInfo(player *model.PlayerModel) (*pb.IdleInfo, error) {
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.IdleModel == nil {
		return nil, errors.New("idle model not loaded")
	}

	// 结算挂机奖励
	s.settleIdleReward(player)

	entity := player.IdleModel.Entity
	levelCfg := gameConfig.GetIdleLevelCfg(entity.IdleLevel)
	if levelCfg == nil {
		return nil, errors.New("idle level config not found")
	}

	// 可领取的奖励（落库 pending_rewards，预览后不可变更）
	rewards := s.getPendingRewardsPB(player)

	info := &pb.IdleInfo{
		IdleLevel:       entity.IdleLevel,
		AccumulatedTime: entity.AccumulatedTime,
		LastSettleTime:  entity.LastSettleTime,
		Rewards:         rewards,
		CanUpgrade:      s.canUpgrade(player),
		QuickClaimInfo:  s.getQuickClaimInfo(player),
	}

	return info, nil
}

// ClaimReward 领取挂机奖励
func (s *IdleServer) ClaimReward(player *model.PlayerModel) ([]*pb.ItemBasicInfo, error) {
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.IdleModel == nil {
		return nil, errors.New("idle model not loaded")
	}

	// 结算挂机奖励
	s.settleIdleReward(player)

	entity := player.IdleModel.Entity
	if entity.AccumulatedTime <= 0 {
		return nil, errors.New("no reward to claim")
	}

	// 读取待领取奖励（落库 pending_rewards，预览后不可变更）
	rewards := s.getPendingRewardsPB(player)
	if len(rewards) == 0 {
		return nil, errors.New("no reward to claim")
	}

	// 发放奖励
	items := make([]*gameConfig.ItemConfig, 0)
	for _, reward := range rewards {
		items = append(items, &gameConfig.ItemConfig{
			ID:  reward.ItemId,
			Num: reward.Count,
		})
	}

	err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_IDLE_REWARD)
	if err != nil {
		return nil, err
	}

	// 重置累计时间
	player.IdleModel.UpdateAccumulatedTime(0)
	player.IdleModel.UpdateLastClaimTime(tool.UnixNow())
	player.IdleModel.ClearPendingRewards()

	return rewards, nil
}

// QuickClaim 快速领取
// idleType 区分普通领取/视频领取，领取奖励逻辑一致，仅计数与消耗不同
func (s *IdleServer) QuickClaim(player *model.PlayerModel, idleType int32) ([]*pb.ItemBasicInfo, []*pb.ItemBasicInfo, error) {
	if player == nil {
		return nil, nil, errors.New("player not found")
	}

	if player.IdleModel == nil {
		return nil, nil, errors.New("idle model not loaded")
	}

	entity := player.IdleModel.Entity

	// 领取类型：0=普通领取(消耗钻石)，1=视频领取(消耗广告次数)
	const (
		idleTypeNormal = int32(0)
		idleTypeAd     = int32(1)
	)

	// 检查快速领取次数重置
	s.checkQuickClaimReset(player)

	// 今日最大可领取次数（考虑解锁），所有类型共享总次数上限
	maxCount := s.getMaxQuickClaimCount(player)
	nextClaimId := int32(0)
	var quickClaimCfg *gameConfig.IdleQuickClaimCfg
	// 当前领取序号：统计今日已快速领取总次数（普通+广告）
	if idleType == idleTypeNormal {
		nextClaimId = entity.QuickClaimCount + 1
		if maxCount > 0 && nextClaimId > maxCount {
			return nil, nil, errors.New("quick claim config not found")
		}
		quickClaimCfg = gameConfig.GetIdleQuickClaimCfg(nextClaimId)
		if quickClaimCfg == nil {
			return nil, nil, errors.New("quick claim config not found")
		}

		// 检查解锁条件
		if quickClaimCfg.UnlockId > 0 {
			if !s.unlockService.CheckUnlock(quickClaimCfg.UnlockId, player) {
				return nil, nil, errors.New("quick claim not unlocked")
			}
		}
	}

	// 检查领取类型相关的前置条件与消耗
	// 普通领取：按原逻辑消耗钻石（首次免费）
	// 视频领取：不消耗钻石，但需要有剩余广告次数
	if idleType == idleTypeAd {
		// 检查广告剩余次数
		if entity.QuickADClaimCount <= 0 {
			return nil, nil, errors.New("ad quick claim not enough")
		}
	} else {
		// 默认走普通快速领取逻辑
		cost := int32(0)
		if nextClaimId > 0 {
			cost = quickClaimCfg.Cost
			if cost > 0 {
				// 检查钻石是否足够（假设钻石道具ID为1，需要根据实际配置调整）
				diamondItemId := int32(1) // TODO: 从配置中获取钻石道具ID
				hasDiamond, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{
					ID:  diamondItemId,
					Num: int64(cost),
				})
				if err != nil || !hasDiamond {
					return nil, nil, errors.New("diamond not enough")
				}

				// 扣除钻石
				err = itemService.RemoveItem(player, &gameConfig.ItemConfig{
					ID:  diamondItemId,
					Num: int64(cost),
				}, enum.ITEM_CHANGE_REASON_IDLE_QUICK_CLAIM)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// 当前领取奖励：优先使用“预览已锁定”的奖励
	quickClaimTime := gameConfig.GetQuickClaimTime()
	currentRewards, err := s.getOrCreateQuickClaimPreviewRewards(player, quickClaimTime)
	if err != nil {
		// 解析失败兜底：重新生成并覆盖（避免阻塞领取）
		currentRewards = s.calculateRewards(player, quickClaimTime)
		_ = s.saveQuickClaimPreviewRewards(player, currentRewards)
	}

	// 发放奖励
	items := make([]*gameConfig.ItemConfig, 0)
	for _, reward := range currentRewards {
		items = append(items, &gameConfig.ItemConfig{
			ID:  reward.ItemId,
			Num: reward.Count,
		})
	}

	err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_IDLE_QUICK_CLAIM)
	if err != nil {
		return nil, nil, err
	}

	// 如果是广告领取，消耗一条广告剩余次数
	if idleType == idleTypeAd {
		player.IdleModel.UpdateQuickADClaimCount(entity.QuickADClaimCount - 1)
	} else {
		// 更新快速领取次数（总次数），广告不计入今日总次数
		player.IdleModel.UpdateQuickClaimCount(nextClaimId)
	}
	// 消耗掉本次预览
	player.IdleModel.ClearQuickClaimPreview()

	// 计算并保存下次快速领取奖励（回包 nextRewards，同时落库用于下次预览）
	nextClaimIdAfter := nextClaimId + 1
	var nextRewards []*pb.ItemBasicInfo
	if maxCount > 0 && nextClaimIdAfter <= maxCount {
		if nextCfg := gameConfig.GetIdleQuickClaimCfg(nextClaimIdAfter); nextCfg != nil {
			nextRewards = s.calculateRewards(player, quickClaimTime)
			_ = s.saveQuickClaimPreviewRewards(player, nextRewards)
		}
	}

	return currentRewards, nextRewards, nil
}

// UpgradeIdleLevel 升级挂机等级
func (s *IdleServer) UpgradeIdleLevel(player *model.PlayerModel) error {
	if player == nil {
		return errors.New("player not found")
	}

	if player.IdleModel == nil {
		return errors.New("idle model not loaded")
	}

	entity := player.IdleModel.Entity
	nextLevel := entity.IdleLevel + 1
	nextLevelCfg := gameConfig.GetIdleLevelCfg(nextLevel)
	if nextLevelCfg == nil {
		return errors.New("next level config not found")
	}

	// 检查解锁条件
	if nextLevelCfg.UnlockId > 0 {
		if !s.unlockService.CheckUnlock(nextLevelCfg.UnlockId, player) {
			return errors.New("upgrade condition not met")
		}
	}

	// 升级
	player.IdleModel.UpdateIdleLevel(nextLevel)

	// 如果升级时正在挂机，需要重新结算奖励
	if entity.AccumulatedTime > 0 {
		s.settleIdleReward(player)
	}

	// 快速领取预览按当前挂机等级计算；升级后必须重算，否则仍沿用旧等级遗留的锁定预览
	quickClaimTime := gameConfig.GetQuickClaimTime()
	_ = s.saveQuickClaimPreviewRewards(player, s.calculateRewards(player, quickClaimTime))

	return nil
}

// settleIdleReward 结算挂机奖励
func (s *IdleServer) settleIdleReward(player *model.PlayerModel) {
	if player.IdleModel == nil || player.IdleModel.Entity == nil {
		return
	}

	entity := player.IdleModel.Entity
	currentTimeSeconds := tool.UnixNow()
	settlementTimeSeconds := int64(gameConfig.GetIdleSettlementTime())
	maxIdleTimeSeconds := int64(gameConfig.GetMaxIdleTime())

	// 获取VIP叠加的挂机时长（秒）
	vipMaxIdleTime, _ := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_IDLE_TIME)
	// 计算最终最大挂机时间 = 基础时间 + VIP叠加时间
	maxIdleTimeSeconds += vipMaxIdleTime

	// 如果上次结算时间为0，说明是首次，设置为当前时间
	if entity.LastSettleTime == 0 {
		player.IdleModel.UpdateLastSettleTime(currentTimeSeconds)
		return
	}

	// 计算经过的时间（秒）
	elapsedTime := currentTimeSeconds - entity.LastSettleTime
	if elapsedTime <= 0 {
		return
	}

	// 检查是否超过最大挂机时间
	maxTimeSeconds := int32(maxIdleTimeSeconds)
	if entity.AccumulatedTime >= maxTimeSeconds {
		// 已经达到最大时间，不再累计，更新结算时间
		player.IdleModel.UpdateLastSettleTime(currentTimeSeconds)
		return
	}

	// 每5分钟结算一次
	settlementCount := elapsedTime / settlementTimeSeconds
	if settlementCount <= 0 {
		return
	}

	// 受最大挂机时间限制：最多还能结算多少次
	remainSeconds := int64(maxTimeSeconds - entity.AccumulatedTime)
	maxSettleCountByCap := remainSeconds / settlementTimeSeconds
	if maxSettleCountByCap <= 0 {
		player.IdleModel.UpdateLastSettleTime(currentTimeSeconds)
		return
	}
	if settlementCount > maxSettleCountByCap {
		settlementCount = maxSettleCountByCap
	}

	// 把随机掉落累加进 pending_rewards（预览后不可变更）
	pendingMap, _ := s.parseItemBasicInfoJSONToMap(entity.PendingRewards)

	levelCfg := gameConfig.GetIdleLevelCfg(entity.IdleLevel)
	if levelCfg != nil {
		for i := int64(0); i < settlementCount; i++ {
			if levelCfg.DropGroupId1 > 0 {
				s.addDropToMap(pendingMap, levelCfg.DropGroupId1)
			}
			if levelCfg.DropGroupId2 > 0 {
				s.addDropGroupToMap(pendingMap, levelCfg.DropGroupId2)
			}
		}
	}

	// 累计挂机时间（秒）
	newAccumulatedTime := entity.AccumulatedTime + int32(settlementCount*settlementTimeSeconds)
	if newAccumulatedTime > maxTimeSeconds {
		newAccumulatedTime = maxTimeSeconds
	}
	player.IdleModel.UpdateAccumulatedTime(newAccumulatedTime)

	// 保存 pending_rewards
	if pendingJSON, err := s.mapToItemBasicInfoJSON(pendingMap); err == nil {
		player.IdleModel.UpdatePendingRewards(pendingJSON)
	}

	// 更新时间：按已结算次数推进，保留余数（避免丢时间）
	player.IdleModel.UpdateLastSettleTime(entity.LastSettleTime + settlementCount*settlementTimeSeconds)
}

// calculateRewards 计算奖励
func (s *IdleServer) calculateRewards(player *model.PlayerModel, timeSeconds int32) []*pb.ItemBasicInfo {
	if player.IdleModel == nil || player.IdleModel.Entity == nil {
		return nil
	}

	entity := player.IdleModel.Entity
	levelCfg := gameConfig.GetIdleLevelCfg(entity.IdleLevel)
	if levelCfg == nil {
		return nil
	}

	rewards := make([]*pb.ItemBasicInfo, 0)
	rewardsMap := make(map[int32]int64) // itemId -> count

	// 获取结算周期（秒）
	settlementTime := gameConfig.GetIdleSettlementTime()
	settlementCount := timeSeconds / settlementTime

	// 计算固定奖励（每5分钟结算一次）
	if levelCfg.DropGroupId1 > 0 {
		dropCfg := gameConfig.GetDropCfg(levelCfg.DropGroupId1)
		if dropCfg != nil {
			// 每5分钟结算一次
			for i := int32(0); i < settlementCount; i++ {
				items := gameConfig.Drop(levelCfg.DropGroupId1)
				for _, item := range items {
					if item != nil && item.ID > 0 {
						rewardsMap[item.ID] += item.Num
					}
				}
			}
		}
	}

	// 计算随机奖励（每5分钟结算一次，使用 DropGroupItems）
	if levelCfg.DropGroupId2 > 0 {
		// 每5分钟结算一次
		for i := int32(0); i < settlementCount; i++ {
			items := gameConfig.DropGroupItems(levelCfg.DropGroupId2, nil)
			for _, item := range items {
				if item != nil && item.ID > 0 {
					rewardsMap[item.ID] += item.Num
				}
			}
		}
	}

	// 转换为PB格式
	for itemId, count := range rewardsMap {
		if count > 0 {
			rewards = append(rewards, &pb.ItemBasicInfo{
				ItemId: itemId,
				Count:  count,
			})
		}
	}

	return rewards
}

// getPendingRewardsPB 读取待领取奖励（落库 JSON）并转换为 PB
func (s *IdleServer) getPendingRewardsPB(player *model.PlayerModel) []*pb.ItemBasicInfo {
	if player == nil || player.IdleModel == nil || player.IdleModel.Entity == nil {
		return nil
	}
	arr, err := model.DecodeItemBasicInfoJSON(player.IdleModel.Entity.PendingRewards)
	if err != nil {
		return nil
	}
	res := make([]*pb.ItemBasicInfo, 0, len(arr))
	for _, it := range arr {
		if it.ItemId <= 0 || it.Count <= 0 {
			continue
		}
		res = append(res, &pb.ItemBasicInfo{ItemId: it.ItemId, Count: it.Count})
	}
	return res
}

func (s *IdleServer) addDropToMap(m map[int32]int64, dropId int32) {
	if dropId <= 0 {
		return
	}
	items := gameConfig.Drop(dropId)
	for _, item := range items {
		if item == nil || item.ID <= 0 || item.Num <= 0 {
			continue
		}
		m[item.ID] += item.Num
	}
}

func (s *IdleServer) addDropGroupToMap(m map[int32]int64, dropGroupId int32) {
	if dropGroupId <= 0 {
		return
	}
	items := gameConfig.DropGroupItems(dropGroupId, nil)
	for _, item := range items {
		if item == nil || item.ID <= 0 || item.Num <= 0 {
			continue
		}
		m[item.ID] += item.Num
	}
}

func (s *IdleServer) parseItemBasicInfoJSONToMap(jsonStr string) (map[int32]int64, error) {
	arr, err := model.DecodeItemBasicInfoJSON(jsonStr)
	if err != nil {
		return make(map[int32]int64), err
	}
	m := make(map[int32]int64, len(arr))
	for _, it := range arr {
		if it.ItemId <= 0 || it.Count <= 0 {
			continue
		}
		m[it.ItemId] += it.Count
	}
	return m, nil
}

func (s *IdleServer) mapToItemBasicInfoJSON(m map[int32]int64) (string, error) {
	type item struct {
		ItemId int32 `json:"itemId"`
		Count  int64 `json:"count"`
	}
	arr := make([]item, 0, len(m))
	for id, cnt := range m {
		if id <= 0 || cnt <= 0 {
			continue
		}
		arr = append(arr, item{ItemId: id, Count: cnt})
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// canUpgrade 检查是否可以升级
func (s *IdleServer) canUpgrade(player *model.PlayerModel) bool {
	if player.IdleModel == nil || player.IdleModel.Entity == nil {
		return false
	}

	entity := player.IdleModel.Entity
	nextLevel := entity.IdleLevel + 1
	nextLevelCfg := gameConfig.GetIdleLevelCfg(nextLevel)
	if nextLevelCfg == nil {
		return false
	}

	// 检查解锁条件
	if nextLevelCfg.UnlockId > 0 {
		return s.unlockService.CheckUnlock(nextLevelCfg.UnlockId, player)
	}

	return true
}

// getQuickClaimInfo 获取快速领取信息
func (s *IdleServer) getQuickClaimInfo(player *model.PlayerModel) *pb.QuickClaimInfo {
	if player.IdleModel == nil || player.IdleModel.Entity == nil {
		return nil
	}

	entity := player.IdleModel.Entity

	// 检查快速领取次数重置（会顺便重置广告次数）
	s.checkQuickClaimReset(player)

	// 所有类型共享总次数配置：按今日已领取总次数计算下一次配置ID
	nextClaimId := entity.QuickClaimCount + 1
	quickClaimCfg := gameConfig.GetIdleQuickClaimCfg(nextClaimId)

	// 获取最大领取次数（配置表中的最大ID）
	maxCount := s.getMaxQuickClaimCount(player)

	cost := int32(200)
	if nextClaimId > 1 {
		if quickClaimCfg != nil {
			cost = quickClaimCfg.Cost
		}
	}

	// 提前生成快速领取奖励（并落库锁定，界面看到后不可变更）
	quickClaimTime := gameConfig.GetQuickClaimTime()
	rewards, err := s.getOrCreateQuickClaimPreviewRewards(player, quickClaimTime)
	if err != nil {
		// 解析失败兜底：重新生成并覆盖
		rewards = s.calculateRewards(player, quickClaimTime)
		_ = s.saveQuickClaimPreviewRewards(player, rewards)
	}

	return &pb.QuickClaimInfo{
		TodayCount: int32(entity.QuickClaimCount),   // 今日已快速领取总次数（普通+广告）
		AdCount:    int32(entity.QuickADClaimCount), // 今日广告剩余次数
		MaxCount:   maxCount,
		Cost:       cost,
		Rewards:    rewards,
	}
}

// getMaxQuickClaimCount 获取今日最大快速领取次数（考虑解锁）
func (s *IdleServer) getMaxQuickClaimCount(player *model.PlayerModel) int32 {
	allCfgs := gameConfig.GetAllIdleQuickClaimCfgs()
	maxCount := int32(0)
	for id, cfg := range allCfgs {
		if cfg == nil {
			continue
		}
		//if cfg.UnlockId > 0 && player != nil && !s.unlockService.CheckUnlock(cfg.UnlockId, player) {
		//	continue
		//}
		if cfg.UnlockId > 0 {
			continue
		}
		if id > maxCount {
			maxCount = id
		}
	}

	vipMaxIdleCount, _ := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_IDLE_REWARD)
	maxCount += int32(vipMaxIdleCount)
	return maxCount
}

func (s *IdleServer) getOrCreateQuickClaimPreviewRewards(player *model.PlayerModel, quickClaimTime int32) ([]*pb.ItemBasicInfo, error) {
	if player == nil || player.IdleModel == nil || player.IdleModel.Entity == nil {
		return nil, errors.New("idle model not loaded")
	}
	entity := player.IdleModel.Entity

	// 如果已有预览奖励，直接复用
	if entity.QuickClaimPreviewRewards != "" && entity.QuickClaimPreviewRewards != "[]" {
		arr, err := model.DecodeItemBasicInfoJSON(entity.QuickClaimPreviewRewards)
		if err != nil {
			return nil, err
		}
		res := make([]*pb.ItemBasicInfo, 0, len(arr))
		for _, it := range arr {
			if it.ItemId <= 0 || it.Count <= 0 {
				continue
			}
			res = append(res, &pb.ItemBasicInfo{ItemId: it.ItemId, Count: it.Count})
		}
		return res, nil
	}

	// 否则生成新的并保存
	rewards := s.calculateRewards(player, quickClaimTime)
	if err := s.saveQuickClaimPreviewRewards(player, rewards); err != nil {
		return rewards, err
	}
	return rewards, nil
}

func (s *IdleServer) saveQuickClaimPreviewRewards(player *model.PlayerModel, rewards []*pb.ItemBasicInfo) error {
	if player == nil || player.IdleModel == nil {
		return errors.New("idle model not loaded")
	}
	type item struct {
		ItemId int32 `json:"itemId"`
		Count  int64 `json:"count"`
	}
	arr := make([]item, 0, len(rewards))
	for _, r := range rewards {
		if r == nil || r.ItemId <= 0 || r.Count <= 0 {
			continue
		}
		arr = append(arr, item{ItemId: r.ItemId, Count: r.Count})
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	player.IdleModel.UpdateQuickClaimPreview(string(b))
	return nil
}

// checkQuickClaimReset 检查快速领取次数重置
func (s *IdleServer) checkQuickClaimReset(player *model.PlayerModel) {
	if player.IdleModel == nil || player.IdleModel.Entity == nil {
		return
	}

	entity := player.IdleModel.Entity
	currentTimeSeconds := tool.UnixNow()

	// 如果重置时间为0，视为首次初始化，设置为当前时间并补充默认广告次数
	if entity.QuickClaimResetTime == 0 {
		player.IdleModel.UpdateQuickClaimResetTime(currentTimeSeconds)
		if entity.QuickADClaimCount <= 0 {
			player.IdleModel.UpdateQuickADClaimCount(1)
		}
		return
	}

	// 检查是否跨天（每天0点重置）
	// 将秒时间戳转换为毫秒用于GetTodayZeroByTimeStamp函数
	currentTimeMilli := currentTimeSeconds * 1000
	resetTimeMilli := entity.QuickClaimResetTime * 1000
	todayZero := tool.GetTodayZeroByTimeStamp(currentTimeMilli)
	resetTimeZero := tool.GetTodayZeroByTimeStamp(resetTimeMilli)
	// 仅在新的一天才重置；同一天使用 >= 会错误重置（如重启后首次进入挂机界面）
	if todayZero > resetTimeZero {
		player.IdleModel.UpdateQuickClaimCount(0)
		player.IdleModel.UpdateQuickADClaimCount(1)
		player.IdleModel.UpdateQuickClaimResetTime(currentTimeSeconds)
		player.IdleModel.ClearQuickClaimPreview()
	}
}
