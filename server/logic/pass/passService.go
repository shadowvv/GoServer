// File: passService.go
// Description: 通行证服务实现
// Author: 木村凉太
// Create Time: 2026.02

package pass

import (
	"errors"
	"fmt"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

type PassService struct {
	ActivityService logicCommon.GameActivityServiceInterface
	MailService     logicCommon.MailServiceInterface // 用于通行证过期时补发未领取奖励
}

var _ logicCommon.PassServiceInterface = (*PassService)(nil)

// getOrLoadPassModel 获取或加载通行证模型
func (p *PassService) getOrLoadPassModel(player logicCommon.PlayerInterface) *model.PassModel {
	if playerModel, ok := player.(*model.PlayerModel); ok {
		if playerModel.PassModel != nil {
			return playerModel.PassModel
		}
		userId := player.GetUserId()
		passModel, err := model.LoadPassModel(userId)
		if err != nil {
			return nil
		}
		playerModel.PassModel = passModel
		playerModel.AppendPlayerModel(passModel)
		return passModel
	}
	return nil
}

// AddPassProgressFromItem 从道具添加通行证进度（ShowGroup 21）
func (p *PassService) AddPassProgressFromItem(player logicCommon.PlayerInterface, passId int32, num int64) error {
	if player == nil {
		return errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return errors.New("failed to load pass model")
	}
	if info, _ := p.GetPassInfo(player, passId); info == nil {
		return errors.New("act is not active")
	}

	// 验证通行证配置
	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return errors.New("pass config not found")
	}

	// 添加进度
	passModel.AddProgress(passId, int32(num))

	// 计算新等级
	p.updatePassLevel(passModel, passId)

	// 保存到数据库
	passModel.SaveModelToDB()
	return nil
}

// SetPassVipLevelFromItem 从道具添加通行证VIP档位（ShowGroup 22）
// level: 档位编号（1=档位1, 2=档位2, 3=档位3, ...）
func (p *PassService) SetPassVipLevelFromItem(player logicCommon.PlayerInterface, passId int32, level int32) error {
	if player == nil {
		return errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return errors.New("failed to load pass model")
	}

	// 验证档位编号（必须大于0）
	if level <= 0 {
		return errors.New("invalid vip level, must be greater than 0")
	}

	// 添加VIP档位（位运算）
	passModel.AddVipLevel(passId, level)

	// 保存到数据库
	passModel.SaveModelToDB()
	return nil
}

// updatePassLevel 更新通行证等级（根据进度计算）
func (p *PassService) updatePassLevel(passModel *model.PassModel, passId int32) {
	progress, _ := passModel.GetOrCreateProgress(passId)
	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return
	}

	// 获取所有奖励配置，按等级排序
	allRewards := gameConfig.GetAllPassRewardCfgByPassId(passId)
	if len(allRewards) == 0 {
		return
	}

	// 计算最大等级和总积分
	maxLevel := int32(0)
	totalRequiredPoints := int32(0)
	for _, reward := range allRewards {
		if reward.Level > maxLevel {
			maxLevel = reward.Level
		}
		totalRequiredPoints += reward.PointsPer
	}

	// 计算当前等级（根据进度和每级所需积分）
	currentLevel := int32(0)
	totalPoints := int32(0)

	for _, reward := range allRewards {
		if reward.Level <= currentLevel {
			continue
		}
		if totalPoints+reward.PointsPer > progress.Progress {
			break
		}
		totalPoints += reward.PointsPer
		currentLevel = reward.Level
	}

	// 更新等级
	if currentLevel > progress.Level {
		passModel.UpdateLevel(passId, currentLevel)
	}

	// 如果已达到最大等级，计算循环积分
	if currentLevel >= maxLevel && basePassCfg.Num3 > 0 {
		// 计算超出最大等级所需的积分
		excessProgress := progress.Progress - totalRequiredPoints
		if excessProgress > 0 {
			// 将超出部分转换为循环积分
			//loopPoints := excessProgress / basePassCfg.Num3
			//if loopPoints > 0 {
			// 更新进度，减去已转换为循环积分的部分
			//progress.Progress = totalRequiredPoints + (excessProgress % basePassCfg.Num3)
			//passModel.AddProgress(passId, 0) // 触发进度变更标记
			// 添加循环积分
			//}
			passModel.AddLoopProgress(passId, excessProgress)

		}
	}
}

// GetPassInfo 获取通行证信息
func (p *PassService) GetPassInfo(player logicCommon.PlayerInterface, passId int32) (*pb.PassInfo, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return nil, errors.New("failed to load pass model")
	}

	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return nil, errors.New("pass config not found")
	}

	// 检查活动是否开启
	if basePassCfg.ActId > 0 {
		if playerModel, ok := player.(*model.PlayerModel); ok {
			if playerModel.PlayerActivityModel == nil {
				return nil, errors.New("player activity model not loaded")
			}
			if open, _ := playerModel.PlayerActivityModel.CheckActivityOpen(basePassCfg.ActId); !open {
				// 没开就是关
				playerModel.PassTaskModel.ClosePassCardTask(passId)
				return nil, errors.New("activity not open")
			}
		}
	}

	progress, isFirst := passModel.GetOrCreateProgress(passId)
	// 首次添加通行证任务
	if isFirst {
		if playerModel, ok := player.(*model.PlayerModel); ok {
			playerModel.PassTaskModel.AddPassCardTask(passId)
		}
	}

	vip := passModel.GetOrCreateVip(passId)

	// 获取所有奖励配置
	allRewards := gameConfig.GetAllPassRewardCfgByPassId(passId)
	rewardList := make([]*pb.PassRewardInfo, 0, len(allRewards))

	for _, rewardCfg := range allRewards {
		// 检查是否已领取各档位奖励
		receivedFree := passModel.HasReceivedReward(passId, rewardCfg.Level, 0)
		receivedVip1 := passModel.HasReceivedReward(passId, rewardCfg.Level, 1)
		receivedVip2 := passModel.HasReceivedReward(passId, rewardCfg.Level, 2)

		rewardList = append(rewardList, &pb.PassRewardInfo{
			Level:        rewardCfg.Level,
			PointsPer:    rewardCfg.PointsPer,
			ReceivedFree: receivedFree,
			ReceivedVip1: receivedVip1,
			ReceivedVip2: receivedVip2,
		})
	}

	// 获取循环积分
	loopProgress := passModel.GetLoopProgress(passId)

	return &pb.PassInfo{
		PassId:       passId,
		Progress:     progress.Progress,
		Level:        progress.Level,
		VipLevel:     vip.VipLevel,
		Rewards:      rewardList,
		LoopProgress: loopProgress,
	}, nil
}

// GetAllPassInfo 获取所有通行证信息
func (p *PassService) GetAllPassInfo(player logicCommon.PlayerInterface) ([]*pb.PassInfo, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	// 获取所有通行证配置
	allPassCfg := gameConfig.GetAllBasePassCfg()
	result := make([]*pb.PassInfo, 0)

	// 检查玩家活动模型
	var playerActivityModel *model.PlayerActivityModel
	if playerModel, ok := player.(*model.PlayerModel); ok {
		playerActivityModel = playerModel.PlayerActivityModel
	}

	// 遍历所有通行证配置
	for passId := range allPassCfg {
		basePassCfg := allPassCfg[passId]
		if basePassCfg == nil {
			continue
		}

		// 检查活动是否开启
		if basePassCfg.ActId > 0 {
			if playerActivityModel == nil {
				continue
			}
			if open, _ := playerActivityModel.CheckActivityOpen(basePassCfg.ActId); !open {
				continue
			}
		}

		// 获取通行证信息
		passInfo, err := p.GetPassInfo(player, passId)
		if err != nil {
			continue
		}
		if passInfo != nil {
			result = append(result, passInfo)
		}
	}

	return result, nil
}

// GetPassRewardOptions 获取通行证奖励选项（当drop有多个道具时）
func (p *PassService) GetPassRewardOptions(player logicCommon.PlayerInterface, passId int32, level int32, rewardLevel int32) ([]*pb.PassRewardOption, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	rewardCfg := gameConfig.GetPassRewardCfgByPassIdAndLevel(passId, level)
	if rewardCfg == nil {
		return nil, errors.New("reward config not found")
	}

	// 根据rewardLevel获取对应的dropId或dropId数组
	var dropIds []int32
	switch rewardLevel {
	case 0: // 免费档位
		if rewardCfg.DropId1 > 0 {
			dropIds = []int32{rewardCfg.DropId1}
		}
	case 1: // 付费档位1
		dropIds = rewardCfg.DropId2
	case 2: // 付费档位2
		dropIds = rewardCfg.DropId3
	default:
		return nil, errors.New("invalid reward level")
	}

	if len(dropIds) == 0 {
		return nil, errors.New("drop id is invalid")
	}

	// 如果只有一个dropId，直接返回该drop的道具选项
	if len(dropIds) == 1 {
		dropCfg := gameConfig.GetDropCfg(dropIds[0])
		if dropCfg == nil {
			return nil, errors.New("drop config not found")
		}

		// 收集所有可能的道具
		options := make([]*pb.PassRewardOption, 0)

		// 固定掉落
		for _, item := range dropCfg.FixedItem {
			options = append(options, &pb.PassRewardOption{
				ItemId: item.ID,
				Count:  item.Num,
			})
		}

		// 概率掉落（每组的所有道具）
		for _, group := range dropCfg.Groups {
			for _, item := range group.Items {
				options = append(options, &pb.PassRewardOption{
					ItemId: item.ID,
					Count:  item.Num,
				})
			}
		}

		return options, nil
	}

	// 如果有多个dropId，返回所有dropId的道具选项（客户端需要先选择dropId，再选择道具）
	options := make([]*pb.PassRewardOption, 0)
	for _, dropId := range dropIds {
		dropCfg := gameConfig.GetDropCfg(dropId)
		if dropCfg == nil {
			continue
		}

		// 固定掉落
		for _, item := range dropCfg.FixedItem {
			options = append(options, &pb.PassRewardOption{
				ItemId: item.ID,
				Count:  item.Num,
			})
		}

		// 概率掉落（每组的所有道具）
		for _, group := range dropCfg.Groups {
			for _, item := range group.Items {
				options = append(options, &pb.PassRewardOption{
					ItemId: item.ID,
					Count:  item.Num,
				})
			}
		}
	}

	return options, nil
}

// ClaimAllPassReward 一次性领取所有可领取的奖励（从当前等级到最大可领取等级）
func (p *PassService) ClaimAllPassReward(player logicCommon.PlayerInterface, passId int32, choices map[string]*pb.PassRewardChoice) ([]*pb.ItemBasicInfo, int32, error) {
	if player == nil {
		return nil, 0, errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return nil, 0, errors.New("failed to load pass model")
	}

	// 验证通行证配置
	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return nil, 0, errors.New("pass config not found")
	}

	// 获取当前进度
	progress, _ := passModel.GetOrCreateProgress(passId)
	currentLevel := progress.Level

	// 获取所有奖励配置（按等级排序）
	allRewardCfgs := gameConfig.GetAllPassRewardCfgByPassId(passId)
	if len(allRewardCfgs) == 0 {
		return nil, 0, errors.New("no reward config found")
	}

	// 获取VIP等级（用于检查）
	_ = passModel.GetOrCreateVip(passId)

	// 收集所有奖励
	allRewards := make([]*pb.ItemBasicInfo, 0)
	maxClaimedLevel := int32(0)

	// 从当前等级开始，遍历所有可领取的奖励
	for _, rewardCfg := range allRewardCfgs {
		// 只处理达到等级的奖励
		if rewardCfg.Level > currentLevel {
			continue
		}

		// 检查每个档位的奖励
		for rewardLevel := int32(0); rewardLevel <= 2; rewardLevel++ {
			// 检查是否已领取
			if passModel.HasReceivedReward(passId, rewardCfg.Level, rewardLevel) {
				continue
			}

			// 检查VIP等级（位运算）
			if rewardLevel > 0 && !passModel.HasVipLevel(passId, rewardLevel) {
				continue
			}

			// 根据rewardLevel获取对应的dropId或dropId数组
			var dropId int32
			switch rewardLevel {
			case 0: // 免费档位
				dropId = rewardCfg.DropId1
			case 1: // 付费档位1
				if len(rewardCfg.DropId2) == 0 {
					continue
				}
				if len(rewardCfg.DropId2) == 1 {
					dropId = rewardCfg.DropId2[0]
				} else {
					// 查找选择信息
					choiceKey := fmt.Sprintf("%d_%d", rewardCfg.Level, rewardLevel)
					choice, hasChoice := choices[choiceKey]
					if !hasChoice || choice.ChosenDropId <= 0 {
						return nil, 0, fmt.Errorf("chosen drop id is required for level %d reward level %d", rewardCfg.Level, rewardLevel)
					}
					// 验证选择的dropId是否在数组中
					validDropId := false
					for _, id := range rewardCfg.DropId2 {
						if id == choice.ChosenDropId {
							validDropId = true
							dropId = choice.ChosenDropId
							break
						}
					}
					if !validDropId {
						return nil, 0, fmt.Errorf("chosen drop id %d is invalid for level %d reward level %d", choice.ChosenDropId, rewardCfg.Level, rewardLevel)
					}
				}
			case 2: // 付费档位2
				if len(rewardCfg.DropId3) == 0 {
					continue
				}
				if len(rewardCfg.DropId3) == 1 {
					dropId = rewardCfg.DropId3[0]
				} else {
					// 查找选择信息
					choiceKey := fmt.Sprintf("%d_%d", rewardCfg.Level, rewardLevel)
					choice, hasChoice := choices[choiceKey]
					if !hasChoice || choice.ChosenDropId <= 0 {
						return nil, 0, fmt.Errorf("chosen drop id is required for level %d reward level %d", rewardCfg.Level, rewardLevel)
					}
					// 验证选择的dropId是否在数组中
					validDropId := false
					for _, id := range rewardCfg.DropId3 {
						if id == choice.ChosenDropId {
							validDropId = true
							dropId = choice.ChosenDropId
							break
						}
					}
					if !validDropId {
						return nil, 0, fmt.Errorf("chosen drop id %d is invalid for level %d reward level %d", choice.ChosenDropId, rewardCfg.Level, rewardLevel)
					}
				}
			default:
				continue
			}

			if dropId <= 0 {
				continue
			}

			// 获取掉落配置
			dropCfg := gameConfig.GetDropCfg(dropId)
			if dropCfg == nil {
				continue
			}

			// 计算掉落道具数量
			dropItemCount := gameConfig.GetDropItemCount(dropId)

			var items []*gameConfig.ItemConfig

			if dropItemCount == 1 {
				// 只有一个道具，直接掉落
				items = gameConfig.Drop(dropId)
			} else if dropItemCount > 1 {
				// 多个道具，需要客户端选择
				// 检查是否已选择过
				dropChoice := passModel.GetDropChoice(passId, rewardCfg.Level, rewardLevel, dropId)
				if dropChoice != nil {
					// 使用之前的选择
					items = []*gameConfig.ItemConfig{
						{ID: dropChoice.ChosenItemId, Num: 1},
					}
				} else {
					// 需要从请求中获取选择信息
					choiceKey := fmt.Sprintf("%d_%d", rewardCfg.Level, rewardLevel)
					choice, hasChoice := choices[choiceKey]
					if !hasChoice || choice.ChosenItemId <= 0 {
						return nil, 0, fmt.Errorf("chosen item id is required for level %d reward level %d drop %d", rewardCfg.Level, rewardLevel, dropId)
					}
					// 验证选择的道具是否在掉落列表中
					validItem := false
					for _, item := range dropCfg.FixedItem {
						if item.ID == choice.ChosenItemId {
							validItem = true
							items = []*gameConfig.ItemConfig{item}
							break
						}
					}
					if !validItem {
						for _, group := range dropCfg.Groups {
							for _, item := range group.Items {
								if item.ID == choice.ChosenItemId {
									validItem = true
									items = []*gameConfig.ItemConfig{item}
									break
								}
							}
							if validItem {
								break
							}
						}
					}
					if !validItem {
						return nil, 0, fmt.Errorf("chosen item id %d is invalid for level %d reward level %d drop %d", choice.ChosenItemId, rewardCfg.Level, rewardLevel, dropId)
					}
					// 保存选择
					passModel.SetDropChoice(passId, rewardCfg.Level, rewardLevel, dropId, choice.ChosenItemId)
				}
			} else {
				continue
			}

			// 添加奖励领取记录
			passModel.AddRewardRecord(passId, rewardCfg.Level, rewardLevel)

			// 转换为PB格式并添加到总奖励列表
			for _, item := range items {
				if item != nil && item.ID > 0 {
					allRewards = append(allRewards, &pb.ItemBasicInfo{
						ItemId: item.ID,
						Count:  item.Num,
					})
				}
			}

			// 更新最大领取等级
			if rewardCfg.Level > maxClaimedLevel {
				maxClaimedLevel = rewardCfg.Level
			}
		}
	}

	// 保存到数据库
	if len(allRewards) > 0 {
		passModel.SaveModelToDB()
	}

	return allRewards, maxClaimedLevel, nil
}

// BuyPassProgress 用钻石购买通行证积分，返回需要扣除的道具信息（不修改数据，由调用方负责扣除道具和添加进度）
func (p *PassService) BuyPassProgress(player logicCommon.PlayerInterface, passId int32, points int32) ([]*pb.ItemBasicInfo, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return nil, errors.New("failed to load pass model")
	}

	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return nil, errors.New("pass config not found")
	}

	// 检查活动是否开启
	if basePassCfg.ActId > 0 {
		if playerModel, ok := player.(*model.PlayerModel); ok {
			if playerModel.PlayerActivityModel == nil {
				return nil, errors.New("player activity model not loaded")
			}
			if settled, _ := playerModel.PlayerActivityModel.CheckActivitySettled(basePassCfg.ActId); settled {
				return nil, errors.New("activity not open")
			}
		}
	}

	// 检查钻石价值配置
	if basePassCfg.DiamondValue <= 0 {
		return nil, errors.New("diamond value not configured")
	}

	// 计算需要的钻石数量
	diamondNeeded := points * basePassCfg.DiamondValue

	// 计算需要扣除的道具信息（假设钻石道具ID为1，需要根据实际配置调整）
	diamondItemId := int32(1) // TODO: 从配置中获取钻石道具ID
	costItems := []*pb.ItemBasicInfo{
		{
			ItemId: diamondItemId,
			Count:  int64(diamondNeeded),
		},
	}

	return costItems, nil
}

// AddPassProgress 添加通行证进度（用于购买积分等场景，由调用方负责扣除道具）
func (p *PassService) AddPassProgress(player logicCommon.PlayerInterface, passId int32, points int32) error {
	if player == nil {
		return errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return errors.New("failed to load pass model")
	}

	// 添加通行证进度
	passModel.AddProgress(passId, points)

	// 计算新等级
	p.updatePassLevel(passModel, passId)

	// 保存到数据库
	passModel.SaveModelToDB()

	return nil
}

// ClaimLoopReward 领取循环奖励（当通行证满级后，多出的积分可以领取循环奖励）
func (p *PassService) ClaimLoopReward(player logicCommon.PlayerInterface, passId int32) ([]*pb.ItemBasicInfo, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return nil, errors.New("failed to load pass model")
	}

	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return nil, errors.New("pass config not found")
	}

	// 检查活动是否开启
	if basePassCfg.ActId > 0 {
		if playerModel, ok := player.(*model.PlayerModel); ok {
			if playerModel.PlayerActivityModel == nil {
				return nil, errors.New("player activity model not loaded")
			}
			if settled, _ := playerModel.PlayerActivityModel.CheckActivitySettled(basePassCfg.ActId); settled {
				return nil, errors.New("activity not open")
			}
		}
	}

	// 检查循环奖励配置
	if basePassCfg.DropId <= 0 {
		return nil, errors.New("loop reward not configured")
	}

	// 检查循环积分是否足够
	if basePassCfg.Num3 <= 0 {
		return nil, errors.New("loop progress requirement not configured")
	}

	loopProgress := passModel.GetLoopProgress(passId)
	if loopProgress < basePassCfg.Num3 {
		return nil, errors.New("loop progress not enough")
	}

	// 计算可以领取的最大次数（向下取整）
	claimCount := loopProgress / basePassCfg.Num3
	if claimCount <= 0 {
		return nil, errors.New("loop progress not enough")
	}

	// 直接掉落，不需要选择逻辑
	items := gameConfig.Drop(basePassCfg.DropId)
	if len(items) == 0 {
		return nil, errors.New("no items in drop")
	}

	// 将掉落物品数量乘以领取次数
	for _, item := range items {
		if item != nil {
			item.Num *= int64(claimCount)
		}
	}

	// 消耗循环积分（消耗所有可领取的次数）
	consumedProgress := claimCount * basePassCfg.Num3
	if !passModel.ConsumeLoopProgress(passId, consumedProgress) {
		return nil, errors.New("failed to consume loop progress")
	}

	// 保存到数据库
	passModel.SaveModelToDB()

	// 转换为PB格式
	result := make([]*pb.ItemBasicInfo, 0, len(items))
	for _, item := range items {
		if item != nil && item.ID > 0 {
			result = append(result, &pb.ItemBasicInfo{
				ItemId: item.ID,
				Count:  item.Num,
			})
		}
	}

	return result, nil
}
