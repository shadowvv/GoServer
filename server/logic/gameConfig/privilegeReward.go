// File: privilegeReward.go
// Description: 特权奖励配置加载（来源 privileges.json recruitment 页签）
// Author: Auto
// Create Time: 2026.02

package gameConfig

import (
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("privilegeReward", &PrivilegeRewardCfgLoader{})
}

// PrivilegeRewardCfg 特权奖励配置（用于 ClaimPrivilegeReward）
type PrivilegeRewardCfg struct {
	RewardType  int32         // 奖励类型（与 pb/逻辑的 rewardType 对应）
	PrivType    int32         // 对应的特权功能类型（VipPrivilegeType）
	Items       []*ItemConfig // 奖励物品列表
	RefreshTime int32         // 刷新周期（当前逻辑按“每日”处理；预留字段）
}

type PrivilegeRewardCfgLoader struct {
	temp map[int32]*PrivilegeRewardCfg
}

var _ configLoaderInterface = (*PrivilegeRewardCfgLoader)(nil)

func (s *PrivilegeRewardCfgLoader) loadData() error {
	// 与 vipCard.go 一致：配置文件不存在则返回空配置
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/privileges.json`, &rawData); err != nil {
		s.temp = make(map[int32]*PrivilegeRewardCfg)
		return nil
	}

	s.temp = make(map[int32]*PrivilegeRewardCfg)
	rows := rawData["recruitment"]
	if rows == nil {
		// 没有 recruitment 页签，视为无配置
		return nil
	}

	for _, row := range rows {
		rewardType := ParseInt(row["id"])
		if rewardType <= 0 {
			continue
		}
		privType := ParseInt(row["privType"])
		if privType <= 0 || !enum.IsValidVipPrivilegeType(privType) {
			continue
		}

		// item 字段为 int[]，格式通常为 "itemId|count|itemId|count..."
		// 兼容 2 个值（单个奖励）和多对（多个奖励）
		intArr := ParseIntArray(row["data"])
		items := make([]*ItemConfig, 0)
		for i := 0; i+1 < len(intArr); i += 2 {
			itemId := intArr[i]
			cnt := intArr[i+1]
			if itemId <= 0 || cnt <= 0 {
				continue
			}
			items = append(items, &ItemConfig{ID: itemId, Num: int64(cnt)})
		}

		s.temp[privType] = &PrivilegeRewardCfg{
			RewardType:  rewardType,
			PrivType:    privType,
			Items:       items,
			RefreshTime: ParseInt(row["refreshTime"]),
		}
	}

	return nil
}

func (s *PrivilegeRewardCfgLoader) checkData() error {
	for rewardType, cfg := range s.temp {
		if cfg == nil {
			continue
		}
		if cfg.PrivType <= 0 || !enum.IsValidVipPrivilegeType(cfg.PrivType) {
			return fmt.Errorf("[gameConfig] privilegeReward invalid privType:%d rewardType:%d", cfg.PrivType, rewardType)
		}
	}
	return nil
}

func (s *PrivilegeRewardCfgLoader) apply() {
	privilegeRewardCfg.Store(s.temp)
}

var privilegeRewardCfg atomic.Value // map[int32]*PrivilegeRewardCfg

// GetPrivilegeRewardCfg 获取特权奖励配置
func GetPrivilegeRewardCfg(rewardType int32) *PrivilegeRewardCfg {
	val := privilegeRewardCfg.Load()
	if val == nil {
		return nil
	}
	return val.(map[int32]*PrivilegeRewardCfg)[rewardType]
}

// GetAllPrivilegeRewardCfg 获取所有特权奖励配置
func GetAllPrivilegeRewardCfg() map[int32]*PrivilegeRewardCfg {
	val := privilegeRewardCfg.Load()
	if val == nil {
		return make(map[int32]*PrivilegeRewardCfg)
	}
	return val.(map[int32]*PrivilegeRewardCfg)
}
