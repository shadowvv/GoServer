package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("product", &ProductCfgLoader{})
}

type ProductCfgLoader struct {
	temp1 map[int32]*ProductCfg
}

var _ configLoaderInterface = (*ProductCfgLoader)(nil)

func (s *ProductCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/product.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*ProductCfg)
	for _, row := range rawData["product"] {
		var v ProductCfg
		v.Id = ParseInt(row["id"])
		v.ReflectID = ParseInt(row["reflectId"])
		v.Rmb = int32(ParseFloat(row["rmb"]) * 100)
		v.Dollar = int32(ParseFloat(row["dollar"]) * 100)
		v.Value = ParseItem(row["value"])
		v.DropId = ParseInt(row["dropId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load product error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *ProductCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load product error invalid ID:%d", id))
		}
		if v.DropId != 0 && GetDropCfg(v.DropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load product error invalid DropId:%d", v.DropId))
		}
		if v.Value != nil && GetItemCfg(v.Value.ID) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load product error invalid value:%d", v.Value.ID))
		}
		if v.ReflectID != 0 && GetProductReflectCfg(v.ReflectID) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load product error invalid ReflectID:%d", v.ReflectID))
		}
	}
	return nil
}

func (s *ProductCfgLoader) apply() {
	product.Store(s.temp1)
}

var product atomic.Value

type ProductCfg struct {
	// 礼包id
	Id int32 `json:"id"`
	// 价位id
	ReflectID int32 `json:"ReflectID"`
	// 人民币
	Rmb int32 `json:"rmb"`
	// 美元
	Dollar int32 `json:"dollar"`
	// 代金券价格
	Value *ItemConfig `json:"value"`
	// 购买立得
	DropId int32 `json:"dropId"`
}

func GetProductCfg(id int32) *ProductCfg {
	cfgMap := product.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ProductCfg)[id]
}
