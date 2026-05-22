package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("adventure", &AdventureCfgLoader{})
}

type AdventureCfgLoader struct {
	temp1 map[int32]*AdventureCfg
}

var _ configLoaderInterface = (*AdventureCfgLoader)(nil)

func (s *AdventureCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/adventure.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*AdventureCfg)
	for _, row := range rawData["adventure"] {
		var v AdventureCfg
		v.Id = ParseInt(row["id"])
		v.Weight = ParseInt(row["weight"])
		v.Limit = ParseInt(row["limit"])
		v.TimeLimit = ParseInt(row["timeLimit"])
		v.Unlock = ParseIntArray(row["unlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load adventure error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *AdventureCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventure error invalid ID:%d", id))
		}
		if v.Weight < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventure error invalid Weight:%d,configId:%d", v.Weight, id))
		}
		if v.Limit <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventure error invalid Limit:%d,configId:%d", v.Limit, id))
		}
		if v.TimeLimit <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventure error invalid TimeLimit:%d,configId:%d", v.TimeLimit, id))
		}
		for _, unlock := range v.Unlock {
			if unlock != 0 && GetUnlockCfg(unlock) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load adventure error invalid Unlock:%d,configId:%d", unlock, id))
			}
		}
	}
	return nil
}

func (s *AdventureCfgLoader) apply() {
	adventure.Store(s.temp1)
}

var adventure atomic.Value

type AdventureCfg struct {
	// 副本
	Id int32 `json:"id"`
	// 权重
	Weight int32 `json:"weight"`
	// 每日结算次数
	Limit int32 `json:"limit"`
	// 挑战时间s
	TimeLimit int32   `json:"timeLimit"`
	Unlock    []int32 `json:"unlock"`
}

func GetAdventureCfg(id int32) *AdventureCfg {
	cfgMap := adventure.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AdventureCfg)[id]
}

func GetAllAdventureCfg() map[int32]*AdventureCfg {
	cfgMap := adventure.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AdventureCfg)
}
