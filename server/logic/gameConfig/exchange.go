package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("exchange", &ExchangeCfgLoader{})
}

type ExchangeCfgLoader struct {
	temp1 map[int32]*ExchangeCfg
}

var _ configLoaderInterface = (*ExchangeCfgLoader)(nil)

func (s *ExchangeCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/exchange.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*ExchangeCfg)
	for _, row := range rawData["exchange"] {
		var v ExchangeCfg
		v.Id = ParseInt(row["id"])
		v.ExchangeId = ParseItemArray(row["exchangeId"])
		v.TargetId = ParseItemArray(row["targetId"])
		v.SystemId = ParseInt(row["systemId"])
		v.ActivityId = ParseInt(row["activityId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load exchange error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *ExchangeCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load exchange error invalid ID:%d", id))
		}
		if v.SystemId != 0 && GetSystemUnlockCfg(v.SystemId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load exchange error invalid systemId:%d,configId:%d", v.SystemId, id))
		}
		for _, item := range v.ExchangeId {
			if GetItemCfg(item.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load exchange error invalid exchangeId:%d,configId:%d", item.ID, id))
			}
		}
		for _, item := range v.TargetId {
			if GetItemCfg(item.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load exchange error invalid targetId:%d,configId:%d", item.ID, id))
			}
		}
	}
	return nil
}

func (s *ExchangeCfgLoader) apply() {
	exchange.Store(s.temp1)
}

var exchange atomic.Value

type ExchangeCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 兑换道具
	ExchangeId []*ItemConfig `json:"exchangeId"`
	// 目标道具
	TargetId []*ItemConfig `json:"targetId"`
	// 功能id
	SystemId int32 `json:"systemId"`
	// 活动id
	ActivityId int32 `json:"activityId"`
}

func GetExchangeCfg(id int32) *ExchangeCfg {
	cfgMap := exchange.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ExchangeCfg)[id]
}
