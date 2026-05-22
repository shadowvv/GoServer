// File: vipCardService.go
// Description: 特权卡服务实现
// Author: 木村凉太
// Create Time: 2026.02

package vipCard

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

var Service = &VipCardService{}

type VipCardService struct {
}

// GetFunctionValue 获取指定功能的所有叠加数值
func (v *VipCardService) GetFunctionValue(player logicCommon.PlayerInterface, privilegeType enum.VipPrivilegeType) (int64, error) {
	if player == nil {
		return 0, errors.New("player is nil")
	}

	vipCardModel := v.getOrLoadVipCardModel(player)
	if vipCardModel == nil {
		return 0, errors.New("failed to load vip card model")
	}

	currentTime := tool.UnixNowMilli()
	activeCards := vipCardModel.GetActiveVipCards(currentTime)

	var totalValue int64 = 0
	for _, card := range activeCards {
		// 获取特权卡配置
		vipCardCfg := gameConfig.GetVipCardCfg(card.ItemId)
		if vipCardCfg == nil {
			continue
		}

		// 累加该功能的数值
		if value, ok := vipCardCfg.Functions[privilegeType]; ok {
			totalValue += value
		}
	}

	return totalValue, nil
}

// GetAllFunctionValues 获取所有功能的叠加数值
func (v *VipCardService) GetAllFunctionValues(player logicCommon.PlayerInterface) (map[enum.VipPrivilegeType]int64, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	vipCardModel := v.getOrLoadVipCardModel(player)
	if vipCardModel == nil {
		return nil, errors.New("failed to load vip card model")
	}

	currentTime := tool.UnixNowMilli()
	activeCards := vipCardModel.GetActiveVipCards(currentTime)

	result := make(map[enum.VipPrivilegeType]int64)
	for _, card := range activeCards {
		// 获取特权卡配置
		vipCardCfg := gameConfig.GetVipCardCfg(card.ItemId)
		if vipCardCfg == nil {
			continue
		}

		// 累加所有功能的数值
		for privType, value := range vipCardCfg.Functions {
			result[privType] += value
		}
	}

	return result, nil
}

// GetActiveVipCards 获取所有有效的特权卡
func (v *VipCardService) GetActiveVipCards(player logicCommon.PlayerInterface) ([]*model.VipCardEntity, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	vipCardModel := v.getOrLoadVipCardModel(player)
	if vipCardModel == nil {
		return nil, errors.New("failed to load vip card model")
	}

	currentTime := tool.UnixNowMilli()
	return vipCardModel.GetActiveVipCards(currentTime), nil
}

// GetVipCardInfoList 获取用于下发给客户端的特权卡信息列表
func (v *VipCardService) GetVipCardInfoList(player logicCommon.PlayerInterface) ([]*pb.VipCardInfo, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	vipCardModel := v.getOrLoadVipCardModel(player)
	if vipCardModel == nil {
		return nil, errors.New("failed to load vip card model")
	}

	currentTime := tool.UnixNowMilli()
	activeCards := vipCardModel.GetActiveVipCards(currentTime)

	result := make([]*pb.VipCardInfo, 0, len(activeCards))
	for _, card := range activeCards {
		if card == nil {
			continue
		}
		cfg := gameConfig.GetVipCardCfg(card.ItemId)
		if cfg == nil {
			continue
		}
		privs := make([]*pb.VipPrivilegeData, 0, len(cfg.Functions))
		for privType, value := range cfg.Functions {
			privs = append(privs, &pb.VipPrivilegeData{
				Type:  pb.VipPrivilegeType(privType),
				Value: value,
			})
		}
		result = append(result, &pb.VipCardInfo{
			ItemId:     card.ItemId,
			ExpireTime: card.ExpireTime,
			Privs:      privs,
		})
	}
	return result, nil
}

// getOrLoadVipCardModel 获取或加载特权卡模型
func (v *VipCardService) getOrLoadVipCardModel(player logicCommon.PlayerInterface) *model.VipCardModel {
	// 尝试从PlayerModel获取
	if playerModel, ok := player.(*model.PlayerModel); ok {
		if playerModel.VipCardModel != nil {
			return playerModel.VipCardModel
		}
		// 如果不存在，加载它
		userId := player.GetUserId()
		vipCardModel, err := model.LoadVipCardModel(userId)
		if err != nil {
			return nil
		}
		playerModel.VipCardModel = vipCardModel
		playerModel.AppendPlayerModel(vipCardModel)
		return vipCardModel
	}
	return nil
}

// AddVipCardFromItem 从item添加特权卡（由itemService调用）
// hours：小时数；当 hours >= VIP_CARD_PERMANENT_HOURS 时视为永久（ExpireTime=-1）
func (v *VipCardService) AddVipCardFromItem(player logicCommon.PlayerInterface, itemId int32, hours int64) error {
	if player == nil {
		return errors.New("player is nil")
	}

	vipCardModel := v.getOrLoadVipCardModel(player)
	if vipCardModel == nil {
		return errors.New("failed to load vip card model")
	}

	// 添加/续期特权卡（hours 从 item.Num 获取）
	vipCardModel.AddVipCardHours(itemId, hours)

	// 保存到数据库
	vipCardModel.SaveModelToDB()
	return nil
}

// ClaimPrivilegeReward 领取特权奖励
func (v *VipCardService) ClaimPrivilegeReward(player logicCommon.PlayerInterface, rewardType int32) ([]*gameConfig.ItemConfig, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	// 从 privileges.json (recruitment 页签) 获取奖励配置
	cfg := gameConfig.GetPrivilegeRewardCfg(rewardType)
	if cfg == nil {
		return nil, errors.New("privilege reward config not found")
	}

	// 检查是否拥有对应的特权功能（由配置的 privType 决定）
	hasPriv, err := v.GetFunctionValue(player, enum.VipPrivilegeType(cfg.PrivType))
	if err != nil {
		return nil, err
	}
	if hasPriv <= 0 {
		return nil, errors.New("no required privilege")
	}

	// 获取或加载特权奖励模型
	rewardModel := v.getOrLoadPrivilegeRewardModel(player)
	if rewardModel == nil {
		return nil, errors.New("failed to load privilege reward model")
	}

	// 特权奖励模型按毫秒判断是否同一天
	currentTimeMs := tool.UnixNowMilli()
	if !rewardModel.CanClaimReward(rewardType, currentTimeMs) {
		return nil, errors.New("already claimed today")
	}

	// 获取奖励配置（直接来自 privileges.json recruitment）
	items := cfg.Items

	// 更新领取时间
	rewardModel.ClaimReward(rewardType, currentTimeMs)
	rewardModel.SaveModelToDB()

	// 注意：物品发放由 controller 层处理，避免循环导入
	return items, nil
}

// getOrLoadPrivilegeRewardModel 获取或加载特权奖励模型
func (v *VipCardService) getOrLoadPrivilegeRewardModel(player logicCommon.PlayerInterface) *model.PrivilegeRewardModel {
	if playerModel, ok := player.(*model.PlayerModel); ok {
		if playerModel.PrivilegeRewardModel != nil {
			return playerModel.PrivilegeRewardModel
		}
		userId := player.GetUserId()
		rewardModel, err := model.LoadPrivilegeRewardModel(userId)
		if err != nil {
			return nil
		}
		playerModel.PrivilegeRewardModel = rewardModel
		playerModel.AppendPlayerModel(rewardModel)
		return rewardModel
	}
	return nil
}
