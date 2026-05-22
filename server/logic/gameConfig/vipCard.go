// File: vipCard.go
// Description: 特权卡配置加载
// Author: 木村凉太
// Create Time: 2026.02

package gameConfig

import (
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("vipCard", &VipCardCfgLoader{})
}

type VipCardCfgLoader struct {
	temp1 map[int32]*VipCardCfg
}

var _ configLoaderInterface = (*VipCardCfgLoader)(nil)

func (s *VipCardCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/privileges.json`, &rawData); err != nil {
		// 如果配置文件不存在，返回空配置
		s.temp1 = make(map[int32]*VipCardCfg)
		return nil
	}

	s.temp1 = make(map[int32]*VipCardCfg)
	for _, row := range rawData["privileges"] {
		privId := ParseInt(row["privId"])
		if privId <= 0 {
			continue
		}
		privType := ParseInt(row["privType"])
		if privType <= 0 {
			continue
		}
		// 验证特权类型是否有效
		if !enum.IsValidVipPrivilegeType(privType) {
			continue
		}
		data := ParseInt64(row["data"])

		cfg := s.temp1[privId]
		if cfg == nil {
			cfg = &VipCardCfg{
				ItemId:    privId,
				Functions: make(map[enum.VipPrivilegeType]int64),
			}
			s.temp1[privId] = cfg
		}

		// 同一张卡允许同一 privType 多行配置，累加即可
		cfg.Functions[enum.VipPrivilegeType(privType)] += data
	}

	return nil
}

func (s *VipCardCfgLoader) checkData() error {
	for itemId, v := range s.temp1 {
		if v.ItemId <= 0 {
			return fmt.Errorf("[gameConfig] load vipCard error invalid ItemId:%d", itemId)
		}
		// 验证对应的item配置存在且为特权卡类型
		//itemCfg := GetItemCfg(v.ItemId)
		//if itemCfg == nil {
		//    return fmt.Errorf("[gameConfig] load vipCard error item config not found:%d", v.ItemId)
		//}
		//if itemCfg.ShowGroup != int32(enum.ITEM_TYPE_VIP_CARD) {
		//    return fmt.Errorf("[gameConfig] load vipCard error item is not vip card type:%d", v.ItemId)
		//}
	}
	return nil
}

func (s *VipCardCfgLoader) apply() {
	vipCard.Store(s.temp1)
}

var vipCard atomic.Value

// VipCardCfg 特权卡配置
type VipCardCfg struct {
	ItemId    int32                           `json:"itemId"`    // 特权卡物品ID
	Functions map[enum.VipPrivilegeType]int64 `json:"functions"` // 特权功能类型 -> 数值映射
}

// GetVipCardCfg 获取特权卡配置
func GetVipCardCfg(itemId int32) *VipCardCfg {
	cfgMap := vipCard.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*VipCardCfg)[itemId]
}

// GetAllVipCardCfg 获取所有特权卡配置
func GetAllVipCardCfg() map[int32]*VipCardCfg {
	cfgMap := vipCard.Load()
	if cfgMap == nil {
		return make(map[int32]*VipCardCfg)
	}
	return cfgMap.(map[int32]*VipCardCfg)
}
