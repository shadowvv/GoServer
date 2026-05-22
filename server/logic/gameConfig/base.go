package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("base", &BaseCfgLoader{})
}

type BaseCfgLoader struct {
	temp1 map[int32]*BaseCfg
}

var _ configLoaderInterface = (*BaseCfgLoader)(nil)

func (s *BaseCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/base.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*BaseCfg)
	for _, row := range rawData["base"] {
		var v BaseCfg
		v.Id = ParseInt(row["id"])
		v.Level = ParseInt(row["level"])
		v.Hero = ParseInt(row["hero"])
		v.Item = ParseItemArray(row["item"])
		v.Stage = ParseInt(row["stage"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load base error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *BaseCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load base error invalid ID:%d", id))
		}
		if v.Level < 0 || GetRoleLevelCfg(v.Level) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load base error invalid Level:%d,configId:%d", v.Level, id))
		}
		if v.Hero != 0 && GetHeroBaseCfg(v.Hero) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load base error invalid Hero:%d,configId:%d", v.Hero, id))
		}
		if v.Item != nil {
			for _, item := range v.Item {
				if GetItemCfg(item.ID) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load base error invalid itemId:%v,configId:%d", v.Item, id))
				}
			}
		}
		if v.Stage == 0 || GetMainStageCfg(v.Stage) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load base error invalid Stage:%d,configId:%d", v.Stage, id))
		}
	}
	return nil
}

func (s *BaseCfgLoader) apply() {
	base.Store(s.temp1)
}

var base atomic.Value

type BaseCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 初始等级
	Level int32 `json:"level"`
	// 初始英雄
	Hero int32 `json:"hero"`
	// 初始道具
	Item []*ItemConfig `json:"item"`
	// 初始主线
	Stage int32 `json:"stage"`
}

func GetBaseCfg() *BaseCfg {
	cfgMap := base.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BaseCfg)[1]
}
