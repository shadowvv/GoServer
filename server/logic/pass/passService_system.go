// File: passService_system.go
// Description: 通行证系统进度更新相关方法
// Author: 木村凉太
// Create Time: 2026.02

package pass

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/tool"
)

// getPassProgressBySystem 根据系统获取通行证进度值（passType=2时使用）
func (p *PassService) getPassProgressBySystem(player logicCommon.PlayerInterface, basePassCfg *gameConfig.BasePassCfg) int32 {
	if basePassCfg.PassType != 2 {
		return 0
	}

	playerModel, ok := player.(*model.PlayerModel)
	if !ok {
		return 0
	}

	switch basePassCfg.Param {
	case 201: // 登录天数
		// 需要从活动开启时间开始计算
		if basePassCfg.ActId <= 0 {
			return 0
		}
		// 通过 PlayerActivityModel 获取活动信息
		if playerModel.PlayerActivityModel == nil {
			return 0
		}
		// 先检查活动是否开启
		//if !playerModel.PlayerActivityModel.IsActivityOpen(basePassCfg.ActId) {
		//    return 0
		//}
		// 通过 PlayerActivityModel 获取活动的开启时间
		//activityOpenTime := playerModel.PlayerActivityModel.GetActivityOpenTime(basePassCfg.ActId)
		//if activityOpenTime == 0 {
		//    return 0
		//}
		// 计算今天是活动开启后的第几天（从活动开启当天算起是第1天）
		// GetNatureDayDistance 返回的是自然日间隔，活动开启当天是第1天
		days := tool.GetNatureDayDistance(tool.UnixNowMilli(), playerModel.User.GetLastLoginTime())
		if days <= 0 {
			return 0
		}
		return 1
	case 202: // 等级
		return playerModel.GetLevel()
	case 203: // 主线关卡
		if playerModel.PlayerInstanceModel == nil {
			return 0
		}
		instanceEntity := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
		if instanceEntity == nil {
			return 0
		}
		return instanceEntity.MaxStageId // todo: 后续如果有其他系统进度需求，继续在这里添加case
	case 204: // 金币副本
		if playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.COIN_INSTANCE_ID)] != nil {
			playerInfo := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.COIN_INSTANCE_ID)]
			cfg := gameConfig.GetDungeonAdventureCfg(playerInfo.CommitLevelReward)
			if cfg == nil {
				return 0
			}
			return cfg.Level
		} else {
			return 0
		}
	case 205: // 胶囊副本
		if playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.CAPSULE_INSTANCE_ID)] != nil {
			playerInfo := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.CAPSULE_INSTANCE_ID)]
			cfg := gameConfig.GetDungeonAdventureCfg(playerInfo.CommitLevelReward)
			if cfg == nil {
				return 0
			}
			return cfg.Level
		} else {
			return 0
		}
	case 206: // 英雄副本
		if playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.HERO_INSTANCE_ID)] != nil {
			playerInfo := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.HERO_INSTANCE_ID)]
			cfg := gameConfig.GetDungeonAdventureCfg(playerInfo.CommitLevelReward)
			if cfg == nil {
				return 0
			}
			return cfg.Level
		} else {
			return 0
		}
	case 207: // 宠物副本
		if playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.PET_INSTANCE_ID)] != nil {
			playerInfo := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.PET_INSTANCE_ID)]
			cfg := gameConfig.GetDungeonAdventureCfg(playerInfo.CommitLevelReward)
			if cfg == nil {
				return 0
			}
			return cfg.Level
		} else {
			return 0
		}
	case 208: // 爬塔副本
		if playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID)] != nil {
			playerTowerInfo := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID)]
			towerCfg := gameConfig.GetTowerCfg(playerTowerInfo.MaxStageId)
			if towerCfg == nil {
				return 0
			}
			return towerCfg.Level
		} else {
			return 0
		}
	default:
		return 0
	}
}

// UpdatePassProgressBySystem 根据系统进度更新通行证进度（passType=2时使用）
func (p *PassService) UpdatePassProgressBySystem(player logicCommon.PlayerInterface, passId int32) error {
	if player == nil {
		return errors.New("player is nil")
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return errors.New("failed to load pass model")
	}

	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return errors.New("pass config not found")
	}

	// 只处理 passType=2 的通行证
	if basePassCfg.PassType != 2 {
		return nil
	}

	// 获取当前系统进度值
	currentProgress := p.getPassProgressBySystem(player, basePassCfg)

	// 获取通行证当前进度
	progressEntity, _ := passModel.GetOrCreateProgress(passId)

	if basePassCfg.Param == 201 {
		currentProgress += progressEntity.Progress
	}

	// 如果系统进度大于通行证进度，更新通行证进度
	if currentProgress > progressEntity.Progress {
		// 计算增加的进度
		addedProgress := currentProgress - progressEntity.Progress
		passModel.AddProgress(passId, addedProgress)

		// 计算新等级
		p.updatePassLevel(passModel, passId)

		// 保存到数据库
		passModel.SaveModelToDB()
	}

	return nil
}

// UpdateAllPassProgressBySystem 更新所有相关通行证的进度（根据param类型）
func (p *PassService) UpdateAllPassProgressBySystem(player logicCommon.PlayerInterface, param int32) {
	if player == nil {
		return
	}

	// 获取所有通行证配置
	allPassCfg := gameConfig.GetAllBasePassCfg()
	for passId, basePassCfg := range allPassCfg {
		if basePassCfg.PassType == 2 && basePassCfg.Param == param {
			_ = p.UpdatePassProgressBySystem(player, passId)
		}
	}
}
