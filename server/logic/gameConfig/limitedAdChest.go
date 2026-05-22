package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("limitedAdChest", &LimitedAdChestCfgLoader{})
}

type LimitedAdChestCfgLoader struct {
	temp1 map[int32]*LimitedAdChestCfg
}

var _ configLoaderInterface = (*LimitedAdChestCfgLoader)(nil)

func (s *LimitedAdChestCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/limitedAdChest.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*LimitedAdChestCfg)
	for _, row := range rawData["reward"] {
		var v LimitedAdChestCfg
		v.Id = ParseInt(row["id"])
		v.DropId = ParseInt(row["reward1"])
		v.AdDropId = ParseInt(row["reward2"])
		v.Duration = ParseInt(row["existenceTime"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load limitedAdChest error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *LimitedAdChestCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load limitedAdChest error invalid ID:%d", id))
		}
		if GetDropCfg(v.DropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load limitedAdChest error invalid dropId:%d,configId:%d", v.DropId, id))
		}
		if GetDropCfg(v.AdDropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load limitedAdChest error invalid adDropId:%d,configId:%d", v.AdDropId, id))
		}
		if v.Duration <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load limitedAdChest error invalid duration:%d,configId:%d", v.Duration, id))
		}
	}
	return nil
}

func (s *LimitedAdChestCfgLoader) apply() {
	limitedAdChest.Store(s.temp1)
}

var limitedAdChest atomic.Value

// LimitedAdChestCfg 广告宝箱配置：普通掉落ID、看广告掉落ID、持续时间(秒)
type LimitedAdChestCfg struct {
	Id       int32 `json:"id"`
	DropId   int32 `json:"dropId"`   // 普通开启掉落ID
	AdDropId int32 `json:"adDropId"` // 看广告开启掉落ID
	Duration int32 `json:"duration"` // 宝箱持续时间(分钟)
}

// GetLimitedAdChestCfg 获取广告宝箱配置
func GetLimitedAdChestCfg(id int32) *LimitedAdChestCfg {
	cfgMap := limitedAdChest.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*LimitedAdChestCfg)[id]
}
