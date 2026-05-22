package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("productReflect", &ProductReflectCfgLoader{})
}

type ProductReflectCfgLoader struct {
	temp1 map[int32]*ProductReflectCfg
}

var _ configLoaderInterface = (*ProductReflectCfgLoader)(nil)

func (s *ProductReflectCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/productReflect.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*ProductReflectCfg)
	for _, row := range rawData["productReflect"] {
		var v ProductReflectCfg
		v.Id = ParseInt(row["id"])
		v.Points = ParseInt(row["points"])
		v.Diamond = ParseInt(row["diamond"])
		v.Value = ParseInt(row["value"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load productReflect error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *ProductReflectCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load productReflect error invalid ID:%d", id))
		}
		//TODO:累充积分
		if GetDropCfg(v.Diamond) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load productReflect error invalid Diamond:%d", v.Diamond))
		}
	}
	return nil
}

func (s *ProductReflectCfgLoader) apply() {
	productReflect.Store(s.temp1)
}

var productReflect atomic.Value

type ProductReflectCfg struct {
	// 价位
	Id int32 `json:"id"`
	// 累充积分
	Points int32 `json:"points"`
	// 钻石数量
	Diamond int32 `json:"diamond"`
	// 钻石价值
	Value int32 `json:"value"`
}

func GetProductReflectCfg(id int32) *ProductReflectCfg {
	cfgMap := productReflect.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ProductReflectCfg)[id]
}
