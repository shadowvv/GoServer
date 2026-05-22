// File: vipCardInterface.go
// Description: 特权卡服务接口定义
// Author: 木村凉太
// Create Time: 2026.02

package vipCard

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

// VipCardServiceInterface 特权卡服务接口
type VipCardServiceInterface interface {
	// AddVipCardFromItem 从item添加特权卡（由itemService调用）
	AddVipCardFromItem(player logicCommon.PlayerInterface, itemId int32, hours int64) error

	// GetFunctionValue 获取指定功能的所有叠加数值
	GetFunctionValue(player logicCommon.PlayerInterface, privilegeType enum.VipPrivilegeType) (int64, error)

	// GetAllFunctionValues 获取所有功能的叠加数值
	GetAllFunctionValues(player logicCommon.PlayerInterface) (map[enum.VipPrivilegeType]int64, error)

	// GetActiveVipCards 获取所有有效的特权卡
	GetActiveVipCards(player logicCommon.PlayerInterface) ([]*model.VipCardEntity, error)

	// GetVipCardInfoList 获取用于下发给客户端的特权卡信息列表
	GetVipCardInfoList(player logicCommon.PlayerInterface) ([]*pb.VipCardInfo, error)

	// ClaimPrivilegeReward 领取特权奖励
	ClaimPrivilegeReward(player logicCommon.PlayerInterface, rewardType int32) ([]*gameConfig.ItemConfig, error)
}
