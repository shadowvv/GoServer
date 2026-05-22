// File: idle.go
// Description: 挂机奖励系统配置加载器
// Author: 木村凉太
// Create Time: 2026.02

package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("idle", &IdleCfgLoader{})
}

type IdleCfgLoader struct {
	temp1 map[int32]*IdleLevelCfg      // 挂机等级配置
	temp2 map[int32]*IdleQuickClaimCfg // 快速领取配置
}

var _ configLoaderInterface = (*IdleCfgLoader)(nil)

func (s *IdleCfgLoader) loadData() error {
	// 加载 idle.json
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/idle.json`, &rawData); err != nil {
		return err
	}

	// 加载挂机等级配置 (idle_level)
	s.temp1 = make(map[int32]*IdleLevelCfg)
	if idleLevel, ok := rawData["level"]; ok {
		for _, row := range idleLevel {
			var v IdleLevelCfg
			v.Id = ParseInt(row["id"])
			v.UnlockId = ParseInt(row["unlockId"])
			v.DropGroupId1 = ParseInt(row["dropId"])
			v.DropGroupId2 = ParseInt(row["dropGroupId"])
			if v.Id <= 0 {
				continue
			}
			if s.temp1[v.Id] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load idle_level error duplicate ID:%d", v.Id))
			}
			s.temp1[v.Id] = &v
		}
	}

	// 加载快速领取配置 (idle_quickClaim)
	s.temp2 = make(map[int32]*IdleQuickClaimCfg)
	if idleQuickClaim, ok := rawData["quickClaim"]; ok {
		for _, row := range idleQuickClaim {
			var v IdleQuickClaimCfg
			v.Id = ParseInt(row["id"])
			v.Cost = ParseInt(row["cost"])
			v.UnlockId = ParseInt(row["unlockId"])
			if v.Id <= 0 {
				continue
			}
			if s.temp2[v.Id] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load idle_quickClaim error duplicate ID:%d", v.Id))
			}
			s.temp2[v.Id] = &v
		}
	}

	return nil
}

func (s *IdleCfgLoader) checkData() error {
	// 检查挂机等级配置
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load idle_level error invalid ID:%d", id))
		}
		// 检查掉落组配置是否存在
		if v.DropGroupId1 > 0 && GetDropCfg(v.DropGroupId1) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load idle_level error invalid dropGroupId1:%d,configId:%d", v.DropGroupId1, id))
		}
		if v.DropGroupId2 > 0 && GetDropGroupCfg(v.DropGroupId2) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load idle_level error invalid dropGroupId2:%d,configId:%d", v.DropGroupId2, id))
		}
	}

	// 检查快速领取配置
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load idle_quickClaim error invalid ID:%d", id))
		}
		if v.Cost < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load idle_quickClaim error invalid cost:%d,configId:%d", v.Cost, id))
		}
	}

	return nil
}

func (s *IdleCfgLoader) apply() {
	idleLevel.Store(s.temp1)
	idleQuickClaim.Store(s.temp2)
}

var idleLevel atomic.Value
var idleQuickClaim atomic.Value

// IdleLevelCfg 挂机等级配置
type IdleLevelCfg struct {
	Id           int32 `json:"id"`          // 等级
	UnlockId     int32 `json:"unlockId"`    // 升级条件（unlock配置ID）
	DropGroupId1 int32 `json:"dropId"`      // 固定奖励（drop ID）
	DropGroupId2 int32 `json:"dropGroupId"` // 随机奖励（掉落组ID）
}

// IdleQuickClaimCfg 快速领取配置
type IdleQuickClaimCfg struct {
	Id       int32 `json:"id"`       // 领取次数
	Cost     int32 `json:"cost"`     // 消耗钻石
	UnlockId int32 `json:"unlockId"` // 解锁条件（unlock配置ID，0表示无条件）
}

// GetIdleLevelCfg 获取挂机等级配置
func GetIdleLevelCfg(level int32) *IdleLevelCfg {
	cfgMap := idleLevel.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*IdleLevelCfg)[level]
}

// GetAllIdleLevelCfgs 获取所有挂机等级配置
func GetAllIdleLevelCfgs() map[int32]*IdleLevelCfg {
	cfgMap := idleLevel.Load()
	if cfgMap == nil {
		return make(map[int32]*IdleLevelCfg)
	}
	return cfgMap.(map[int32]*IdleLevelCfg)
}

// GetIdleQuickClaimCfg 获取快速领取配置
func GetIdleQuickClaimCfg(id int32) *IdleQuickClaimCfg {
	cfgMap := idleQuickClaim.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*IdleQuickClaimCfg)[id]
}

// GetAllIdleQuickClaimCfgs 获取所有快速领取配置
func GetAllIdleQuickClaimCfgs() map[int32]*IdleQuickClaimCfg {
	cfgMap := idleQuickClaim.Load()
	if cfgMap == nil {
		return make(map[int32]*IdleQuickClaimCfg)
	}
	return cfgMap.(map[int32]*IdleQuickClaimCfg)
}

const (
	CONSTANT_IDLE_SETTLEMENT_TIME = "idleSettlementTime"
	CONSTANT_MAX_IDLE_TIME        = "maxIdleTime"
	CONSTANT_QUICK_CLAIM_TIME     = "quickClaimTime"
)

// GetIdleSettlementTime 获取挂机结算时间（秒）
func GetIdleSettlementTime() int32 {
	cfg := GetConstantCfg(CONSTANT_IDLE_SETTLEMENT_TIME)
	if cfg == nil || len(cfg.Value) == 0 {
		return 300 // 默认5分钟
	}
	return cfg.Value[0]
}

// GetMaxIdleTime 获取最长挂机时间（秒）
func GetMaxIdleTime() int32 {
	cfg := GetConstantCfg(CONSTANT_MAX_IDLE_TIME)
	if cfg == nil || len(cfg.Value) == 0 {
		return 21600 // 默认6小时
	}
	return cfg.Value[0]
}

// GetQuickClaimTime 获取快速领取收益时间（秒）
func GetQuickClaimTime() int32 {
	cfg := GetConstantCfg(CONSTANT_QUICK_CLAIM_TIME)
	if cfg == nil || len(cfg.Value) == 0 {
		return 10800 // 默认180分钟
	}
	return cfg.Value[0]
}
