// File: passInterface.go
// Description: 通行证服务接口定义
// Author: 木村凉太
// Create Time: 2026.02

package logicCommon

import (
	"github.com/drop/GoServer/server/logic/pb"
)

// PassServiceInterface 通行证服务接口
type PassServiceInterface interface {
	// AddPassProgressFromItem 从道具添加通行证进度（ShowGroup 21）
	AddPassProgressFromItem(player PlayerInterface, passId int32, num int64) error

	// SetPassVipLevelFromItem 从道具添加通行证VIP档位（ShowGroup 22，使用位运算）
	// level: 档位编号（1=档位1, 2=档位2, 3=档位3, ...）
	SetPassVipLevelFromItem(player PlayerInterface, passId int32, level int32) error

	// GetPassInfo 获取通行证信息（用于下发给客户端）
	GetPassInfo(player PlayerInterface, passId int32) (*pb.PassInfo, error)

	// GetAllPassInfo 获取所有通行证信息
	GetAllPassInfo(player PlayerInterface) ([]*pb.PassInfo, error)

	// GetPassRewardOptions 获取通行证奖励选项（当drop有多个道具时）
	GetPassRewardOptions(player PlayerInterface, passId int32, level int32, rewardLevel int32) ([]*pb.PassRewardOption, error)

	// ClaimAllPassReward 一次性领取所有可领取的奖励（从当前等级到最大可领取等级）
	// choices: 奖励选择信息映射，key为 "level_rewardLevel"，value为选择信息
	ClaimAllPassReward(player PlayerInterface, passId int32, choices map[string]*pb.PassRewardChoice) ([]*pb.ItemBasicInfo, int32, error)

	// BuyPassProgress 用钻石购买通行证积分，返回需要扣除的道具信息（不修改数据，由调用方负责扣除道具和添加进度）
	BuyPassProgress(player PlayerInterface, passId int32, points int32) ([]*pb.ItemBasicInfo, error)

	// AddPassProgress 添加通行证进度（用于购买积分等场景，由调用方负责扣除道具）
	AddPassProgress(player PlayerInterface, passId int32, points int32) error

	// ClaimLoopReward 领取循环奖励（当通行证满级后，多出的积分可以领取循环奖励）
	ClaimLoopReward(player PlayerInterface, passId int32) ([]*pb.ItemBasicInfo, error)

	// UpdatePassProgressBySystem 根据系统进度更新通行证进度（passType=2时使用）
	UpdatePassProgressBySystem(player PlayerInterface, passId int32) error

	// UpdateAllPassProgressBySystem 更新所有相关通行证的进度（根据param类型）
	UpdateAllPassProgressBySystem(player PlayerInterface, param int32)

	// ProcessExpiredPassMailsForPlayer 玩家登录时检测：该玩家所在服已结束的通行证活动中，若有未领取奖励则补发邮件
	ProcessExpiredPassMailsForPlayer(player PlayerInterface)
}
