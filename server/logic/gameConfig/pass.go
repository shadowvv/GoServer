// File: pass.go
// Description: 通行证配置加载
// Author: 木村凉太
// Create Time: 2026.02

package gameConfig

import (
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("pass", &PassCfgLoader{})
}

type PassCfgLoader struct {
	temp1 map[int32]*BasePassCfg
	temp2 map[int32]*PassRewardCfg
}

var _ configLoaderInterface = (*PassCfgLoader)(nil)

func (s *PassCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/pass.json`, &rawData); err != nil {
		// 如果配置文件不存在，返回空配置
		s.temp1 = make(map[int32]*BasePassCfg)
		s.temp2 = make(map[int32]*PassRewardCfg)
		return nil
	}

	// 加载 basePass 配置
	s.temp1 = make(map[int32]*BasePassCfg)
	if basePassData, ok := rawData["basePass"]; ok {
		for _, row := range basePassData {
			var v BasePassCfg
			v.Id = ParseInt(row["id"])
			if v.Id <= 0 {
				continue
			}
			// 只读取服务器字段（cs 和 s）
			v.PassType = ParseInt(row["passType"])
			v.TaskSwitch = ParseInt(row["taskSwitch"])
			v.ActId = ParseInt(row["actId"])               // 活动ID（cs字段）
			v.DiamondValue = ParseInt(row["diamondValue"]) // 1积分的钻石价值（cs字段）
			v.Num3 = ParseInt(row["num3"])                 // 循环积分（cs字段）
			v.DropId = ParseInt(row["dropId"])             // 循环奖励（cs字段）
			v.Param = ParseInt(row["param"])               // 参数（cs字段）：201=登录天数, 202=等级, 203=主线关卡
			// 其他字段为客户端字段，不读取
			if s.temp1[v.Id] != nil {
				return fmt.Errorf("[gameConfig] load basePass error duplicate ID:%d", v.Id)
			}
			s.temp1[v.Id] = &v
		}
	}

	// 加载 passReward 配置
	s.temp2 = make(map[int32]*PassRewardCfg)
	if passRewardData, ok := rawData["passReward"]; ok {
		for _, row := range passRewardData {
			var v PassRewardCfg
			v.Id = ParseInt(row["id"])
			if v.Id <= 0 {
				continue
			}
			v.PassId = ParseInt(row["passId"])
			v.Level = ParseInt(row["level"])
			v.PointsPer = ParseInt(row["pointsPer"])
			v.DropId1 = ParseInt(row["dropId1"])
			// dropId2 和 dropId3 是数组
			v.DropId2 = ParseIntArray(row["dropId2"])
			v.DropId3 = ParseIntArray(row["dropId3"])
			if s.temp2[v.Id] != nil {
				return fmt.Errorf("[gameConfig] load passReward error duplicate ID:%d", v.Id)
			}
			s.temp2[v.Id] = &v
		}
	}

	return nil
}

func (s *PassCfgLoader) checkData() error {
	// 检查 basePass
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return fmt.Errorf("[gameConfig] load basePass error invalid ID:%d", id)
		}
		// passType: 1=道具数量进度, 2=其他系统进度
		if v.PassType != 1 && v.PassType != 2 {
			return fmt.Errorf("[gameConfig] load basePass error invalid passType:%d,configId:%d", v.PassType, id)
		}
	}

	// 检查 passReward
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return fmt.Errorf("[gameConfig] load passReward error invalid ID:%d", id)
		}
		if v.PassId <= 0 {
			return fmt.Errorf("[gameConfig] load passReward error invalid PassId:%d,configId:%d", v.PassId, id)
		}
		if s.temp1[v.PassId] == nil {
			return fmt.Errorf("[gameConfig] load passReward error basePass not found:%d,configId:%d", v.PassId, id)
		}
		if v.Level <= 0 {
			return fmt.Errorf("[gameConfig] load passReward error invalid Level:%d,configId:%d", v.Level, id)
		}
		// 检查 dropId 配置
		if v.DropId1 > 0 && GetDropCfg(v.DropId1) == nil {
			return fmt.Errorf("[gameConfig] load passReward error invalid DropId1:%d,configId:%d", v.DropId1, id)
		}
		// 检查 dropId2 数组
		for _, dropId := range v.DropId2 {
			if dropId > 0 && GetDropCfg(dropId) == nil {
				return fmt.Errorf("[gameConfig] load passReward error invalid DropId2:%d,configId:%d", dropId, id)
			}
		}
		// 检查 dropId3 数组
		for _, dropId := range v.DropId3 {
			if dropId > 0 && GetDropCfg(dropId) == nil {
				return fmt.Errorf("[gameConfig] load passReward error invalid DropId3:%d,configId:%d", dropId, id)
			}
		}
	}

	return nil
}

func (s *PassCfgLoader) apply() {
	basePass.Store(s.temp1)
	passReward.Store(s.temp2)
}

var basePass atomic.Value
var passReward atomic.Value

// BasePassCfg 通行证基础配置
type BasePassCfg struct {
	Id           int32 `json:"id"`           // 通行证ID
	PassType     int32 `json:"passType"`     // 通行证类型：1=道具数量进度, 2=其他系统进度
	TaskSwitch   int32 `json:"taskSwitch"`   // 任务开关（暂时不做）
	ActId        int32 `json:"actId"`        // 活动ID（需要检查活动是否开启）
	DiamondValue int32 `json:"diamondValue"` // 1积分的钻石价值（可以用钻石购买积分）
	Num3         int32 `json:"num3"`         // 循环积分（当通行证满级后，多出的积分可以领取循环奖励）
	DropId       int32 `json:"dropId"`       // 循环奖励掉落ID
	Param        int32 `json:"param"`        // 参数（passType=2时使用）：201=登录天数, 202=等级, 203=主线关卡
}

// PassRewardCfg 通行证奖励配置
type PassRewardCfg struct {
	Id        int32   `json:"id"`        // 奖励ID
	PassId    int32   `json:"passId"`    // 通行证ID
	Level     int32   `json:"level"`     // 等级
	PointsPer int32   `json:"pointsPer"` // 每级所需积分
	DropId1   int32   `json:"dropId1"`   // 免费档位掉落ID
	DropId2   []int32 `json:"dropId2"`   // 付费档位1掉落ID数组
	DropId3   []int32 `json:"dropId3"`   // 付费档位2掉落ID数组
}

// GetBasePassCfg 获取通行证基础配置
func GetBasePassCfg(passId int32) *BasePassCfg {
	cfgMap := basePass.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BasePassCfg)[passId]
}

// GetAllBasePassCfg 获取所有通行证基础配置
func GetAllBasePassCfg() map[int32]*BasePassCfg {
	cfgMap := basePass.Load()
	if cfgMap == nil {
		return make(map[int32]*BasePassCfg)
	}
	return cfgMap.(map[int32]*BasePassCfg)
}

// GetPassRewardCfg 获取通行证奖励配置
func GetPassRewardCfg(rewardId int32) *PassRewardCfg {
	cfgMap := passReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PassRewardCfg)[rewardId]
}

// GetPassRewardCfgByPassIdAndLevel 根据通行证ID和等级获取奖励配置
func GetPassRewardCfgByPassIdAndLevel(passId int32, level int32) *PassRewardCfg {
	cfgMap := passReward.Load()
	if cfgMap == nil {
		return nil
	}
	for _, cfg := range cfgMap.(map[int32]*PassRewardCfg) {
		if cfg.PassId == passId && cfg.Level == level {
			return cfg
		}
	}
	return nil
}

// GetAllPassRewardCfgByPassId 获取指定通行证的所有奖励配置（按等级排序）
func GetAllPassRewardCfgByPassId(passId int32) []*PassRewardCfg {
	cfgMap := passReward.Load()
	if cfgMap == nil {
		return nil
	}
	result := make([]*PassRewardCfg, 0)
	for _, cfg := range cfgMap.(map[int32]*PassRewardCfg) {
		if cfg.PassId == passId {
			result = append(result, cfg)
		}
	}
	// 按等级排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Level < result[j].Level
	})
	return result
}

// GetDropItemCount 获取掉落配置中的道具数量
// 返回掉落配置中所有可能的道具数量（用于判断是否需要客户端选择）
func GetDropItemCount(dropId int32) int {
	if dropId <= 0 {
		return 0
	}
	dropCfg := GetDropCfg(dropId)
	if dropCfg == nil {
		return 0
	}
	// 计算固定掉落数量
	count := len(dropCfg.FixedItem)
	// 计算概率掉落组数（每组只能掉落一个）
	count += len(dropCfg.Groups)
	return count
}

// GetPassRewardDropIds 获取通行证奖励的掉落ID列表（用于判断是否需要选择dropId）
func GetPassRewardDropIds(rewardCfg *PassRewardCfg, rewardLevel int32) []int32 {
	switch rewardLevel {
	case 0: // 免费档位
		if rewardCfg.DropId1 > 0 {
			return []int32{rewardCfg.DropId1}
		}
	case 1: // 付费档位1
		return rewardCfg.DropId2
	case 2: // 付费档位2
		return rewardCfg.DropId3
	}
	return nil
}
