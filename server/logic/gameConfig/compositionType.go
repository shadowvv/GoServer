package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("compositionType", &CompositionTypeCfgLoader{})
}

type CompositionTypeCfgLoader struct {
	temp1 map[int32]*CompositionTypeCfg
}

var _ configLoaderInterface = (*CompositionTypeCfgLoader)(nil)

func (s *CompositionTypeCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/compositionType.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CompositionTypeCfg)
	for _, row := range rawData["compositionType"] {
		var v CompositionTypeCfg
		v.Id = ParseInt(row["id"])
		v.Unlock = ParseIntArray(row["unlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load compositionType error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *CompositionTypeCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load compositionType error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *CompositionTypeCfgLoader) apply() {
	compositionType.Store(s.temp1)
}

var compositionType atomic.Value

type CompositionTypeCfg struct {
	// 阵容类型
	Id int32 `json:"id"`
	// 1号位
	Unlock []int32 `json:"unlock"`
}

func GetCompositionTypeCfg(id int32) *CompositionTypeCfg {
	cfgMap := compositionType.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CompositionTypeCfg)[id]
}
