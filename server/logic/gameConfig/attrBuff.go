package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("attrBuff", &AttrBuffCfgLoader{})
}

type AttrBuffCfgLoader struct {
	temp1 map[int32]*AttrBuffCfg
}

var _ configLoaderInterface = (*AttrBuffCfgLoader)(nil)

func (s *AttrBuffCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/attrBuff.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*AttrBuffCfg)
	for _, row := range rawData["attrBuff"] {
		var v AttrBuffCfg
		v.Id = ParseInt(row["id"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load attrBuff error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *AttrBuffCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if len(v.Attr) != len(v.AttrNum) {
			return errors.New(fmt.Sprintf("[gameConfig] load attrBuff error duplicate ID:%d", v.Id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load attrBuff error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *AttrBuffCfgLoader) apply() {
	attrBuff.Store(s.temp1)
}

var attrBuff atomic.Value

type AttrBuffCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
}

func GetAttrBuffCfg(id int32) *AttrBuffCfg {
	cfgMap := attrBuff.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AttrBuffCfg)[id]
}
