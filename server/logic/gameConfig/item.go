package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("item", &ItemCfgLoader{})
}

var loopBoxItemMap atomic.Value // map[int32]int32
var petCardItemMap atomic.Value // map[int32]int32 (petId -> cardItemId)

type ItemCfgLoader struct {
	temp1 map[int32]*ItemCfg
}

var _ configLoaderInterface = (*ItemCfgLoader)(nil)

func (s *ItemCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/item.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*ItemCfg)
	for _, row := range rawData["item"] {
		var v ItemCfg
		v.Id = ParseInt(row["id"])
		v.Type = ParseInt(row["type"])
		v.ShowGroup = ParseInt(row["showGroup"])
		v.Quality = ParseInt(row["quality"])
		v.TargetId = ParseInt(row["targetId"])
		v.TargetId2 = ParseInt(row["targetId2"])
		v.Value = ParseInt(row["value"])
		v.Level = ParseInt(row["level"])
		v.Star = ParseInt(row["star"])
		v.IsAutoUse = ParseInt(row["isAutoUse"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load item error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *ItemCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load item error invalid ID:%d", id))
		}
		if !enum.IsValidItemType(v.ShowGroup) {
			return errors.New(fmt.Sprintf("[gameConfig] load item error invalid showGroup:%d,configId:%d", v.ShowGroup, id))
		}
		if !enum.IsValidItemQuality(v.Quality) {
			return errors.New(fmt.Sprintf("[gameConfig] load item error invalid quality:%d,configId:%d", v.Quality, id))
		}
		if v.Level < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load item error invalid Level:%d,configId:%d", v.Level, id))
		}
		if v.Star < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load item error invalid Star:%d,configId:%d", v.Star, id))
		}
		switch v.ShowGroup {
		case int32(enum.ITEM_TYPE_HERO):
			if GetHeroBaseCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d,configId:%d", v.TargetId, id))
			}
		case int32(enum.ITEM_TYPE_NORMAL_CHEST):
			if GetDropCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d,configId:%d", v.TargetId, id))
			}
		case int32(enum.ITEM_TYPE_PICK_CHEST):
			if GetDropCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d,configId:%d", v.TargetId, id))
			}
		case int32(enum.ITEM_TYPE_SCRAP):
			if GetExchangeCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d,configId:%d", v.TargetId, id))
			}
		case int32(enum.ITEM_TYPE_EQUIP):
			if GetEquipmentBaseCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d,configId:%d", v.TargetId, id))
			}
		case int32(enum.ITEM_TYPE_AD_CHEST):
			if GetLimitedAdChestCfg(v.TargetId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load item error invalid TargetId:%d for adChest,configId:%d", v.TargetId, id))
			}
		}
	}
	return nil
}

func (s *ItemCfgLoader) apply() {
	item.Store(s.temp1)
	initLoopBox()
	initPetCardItemMap()
}

var item atomic.Value

type ItemCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 大类
	Type int32 `json:"type"`
	// 子类别
	ShowGroup int32 `json:"showGroup"`
	// 道具品质
	Quality int32 `json:"quality"`
	// 等价钻石
	Value int32 `json:"value"`
	// 目标id
	TargetId int32 `json:"targetId"`
	// 额外参数
	TargetId2 int32 `json:"targetId2"`
	// 等级
	Level int32 `json:"level"`
	// 星级
	Star int32 `json:"star"`
	// 是否自动使用
	IsAutoUse int32 `json:"isAutoUse"`
}

func GetItemCfg(id int32) *ItemCfg {
	cfgMap := item.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ItemCfg)[id]
}

func GetLoopBoxItemMap() map[int32]int32 {
	v := loopBoxItemMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]int32)
}

func GetPetCardItemID(petId int32) int32 {
	if petId <= 0 {
		return 0
	}
	v := petCardItemMap.Load()
	if v == nil {
		return 0
	}
	return v.(map[int32]int32)[petId]
}

func initLoopBox() {
	newMap := make(map[int32]int32)
	itemRaw := item.Load()
	if itemRaw == nil {
		loopBoxItemMap.Store(newMap)
		return
	}
	for _, v := range itemRaw.(map[int32]*ItemCfg) {
		if v.ShowGroup == int32(enum.ITEM_TYPE_LOOP_BOX) {
			newMap[v.TargetId] = v.Id
		}
	}
	loopBoxItemMap.Store(newMap)
}

func initPetCardItemMap() {
	newMap := make(map[int32]int32)
	for _, v := range item.Load().(map[int32]*ItemCfg) {
		if v == nil {
			continue
		}
		if v.ShowGroup != int32(enum.ITEM_TYPE_PET) {
			continue
		}
		// 仅保留第一张卡即可；同一个 petId 若存在多张卡，默认选择配置中扫描到的第一张。
		if _, ok := newMap[v.TargetId]; ok {
			continue
		}
		newMap[v.TargetId] = v.Id
	}
	petCardItemMap.Store(newMap)
}
