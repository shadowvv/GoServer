package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("cityAgeUp", &CityAgeUpCfgLoader{})
}

type CityAgeUpCfgLoader struct {
	temp1 map[int32]*CityAgeUpCfg
}

var _ configLoaderInterface = (*CityAgeUpCfgLoader)(nil)

func (s *CityAgeUpCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/cityAgeUp.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CityAgeUpCfg)
	for _, row := range rawData["cityAgeUp"] {
		var v CityAgeUpCfg
		v.Id = ParseInt(row["id"])
		v.Age = ParseInt(row["age"])
		v.Period = ParseInt(row["period"])
		v.UnlockAge1 = ParseIntArray(row["unlockAge1"])
		v.UnlockAge2 = ParseIntArray(row["unlockAge2"])
		v.UnlockAge3 = ParseIntArray(row["unlockAge3"])
		v.UnlockAge4 = ParseIntArray(row["unlockAge4"])
		v.UnlockAge5 = ParseIntArray(row["unlockAge5"])
		v.Drop = ParseIntArray(row["drop"])
		v.Drop2 = ParseInt(row["drop2"])
		v.DropAge = ParseInt(row["dropAge"])
		v.AttrAge = make(map[int32]int64)
		for _, attr := range ParseItemArray(row["attrAge"]) {
			if attr != nil {
				v.AttrAge[attr.ID] += attr.Num
			}
		}
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *CityAgeUpCfgLoader) checkData() error {
	cfgList := make([]*CityAgeUpCfg, 0, len(s.temp1))
	for id, cfg := range s.temp1 {
		if cfg.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid ID:%d", id))
		}
		if cfg.Age < 0 || cfg.Period < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid age/period ID:%d", id))
		}
		if cfg.Drop2 < 0 || (cfg.Drop2 > 0 && GetDropCfg(cfg.Drop2) == nil) {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid drop2 ID:%d", id))
		}
		if len(cfg.Drop) != 0 && len(cfg.Drop) != 5 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid drop length ID:%d", id))
		}
		for _, dropID := range cfg.Drop {
			if dropID == 0 {
				continue
			}
			if dropID < 0 || GetDropCfg(dropID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid drop ID:%d drop:%d", id, dropID))
			}
		}
		if cfg.DropAge > 0 && GetDropCfg(cfg.DropAge) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid dropAge ID:%d dropAge:%d", id, cfg.DropAge))
		}
		for attrID := range cfg.AttrAge {
			if attrID <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid attrAge ID:%d", id))
			}
		}
		for groupIndex := int32(1); groupIndex <= 5; groupIndex++ {
			taskIDs := cfg.GetGroupTasks(groupIndex)
			dropID := cfg.GetGroupDrop(groupIndex)
			if len(taskIDs) == 0 && dropID > 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error empty unlockAge group with drop ID:%d group:%d drop:%d", id, groupIndex, dropID))
			}
			if len(taskIDs) > 0 && dropID <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid drop ID:%d group:%d drop:%d", id, groupIndex, dropID))
			}
			for _, taskID := range taskIDs {
				if taskID <= 0 || GetCoreCfg(taskID) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid core task ID:%d group:%d task:%d", id, groupIndex, taskID))
				}
			}
		}
		cfgList = append(cfgList, cfg)
	}

	sortCityAgeUpCfgList(cfgList)
	for index, cfg := range cfgList {
		if index == len(cfgList)-1 {
			continue
		}
		if len(cfg.Drop) != 5 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid drop length for upgrade ID:%d", cfg.Id))
		}
		if cfg.DropAge <= 0 || GetDropCfg(cfg.DropAge) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityAgeUp error invalid dropAge for upgrade ID:%d", cfg.Id))
		}
	}
	return nil
}

func (s *CityAgeUpCfgLoader) apply() {
	cityAgeUp.Store(s.temp1)
}

var cityAgeUp atomic.Value

type CityAgeUpCfg struct {
	Id         int32           `json:"id"`
	Age        int32           `json:"age"`
	Period     int32           `json:"period"`
	UnlockAge1 []int32         `json:"unlockAge1"`
	UnlockAge2 []int32         `json:"unlockAge2"`
	UnlockAge3 []int32         `json:"unlockAge3"`
	UnlockAge4 []int32         `json:"unlockAge4"`
	UnlockAge5 []int32         `json:"unlockAge5"`
	Drop       []int32         `json:"drop"`
	Drop2      int32           `json:"drop2"`
	DropAge    int32           `json:"dropAge"`
	AttrAge    map[int32]int64 `json:"attrAge"`
}

func sortCityAgeUpCfgList(cfgList []*CityAgeUpCfg) {
	for i := 0; i < len(cfgList)-1; i++ {
		for j := i + 1; j < len(cfgList); j++ {
			left := cfgList[i]
			right := cfgList[j]
			if right.Age < left.Age ||
				(right.Age == left.Age && right.Period < left.Period) ||
				(right.Age == left.Age && right.Period == left.Period && right.Id < left.Id) {
				cfgList[i], cfgList[j] = cfgList[j], cfgList[i]
			}
		}
	}
}

func GetCityAgeUpCfg(id int32) *CityAgeUpCfg {
	cfgMap := cityAgeUp.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CityAgeUpCfg)[id]
}

func GetAllCityAgeUpCfg() []*CityAgeUpCfg {
	cfgMap := cityAgeUp.Load()
	if cfgMap == nil {
		return nil
	}

	res := make([]*CityAgeUpCfg, 0, len(cfgMap.(map[int32]*CityAgeUpCfg)))
	for _, cfg := range cfgMap.(map[int32]*CityAgeUpCfg) {
		res = append(res, cfg)
	}
	sortCityAgeUpCfgList(res)
	return res
}

func GetFirstCityAgeUpCfg() *CityAgeUpCfg {
	cfgList := GetAllCityAgeUpCfg()
	if len(cfgList) == 0 {
		return nil
	}
	return cfgList[0]
}

func GetNextCityAgeUpCfg(id int32) *CityAgeUpCfg {
	cfgList := GetAllCityAgeUpCfg()
	for index, cfg := range cfgList {
		if cfg.Id != id {
			continue
		}
		if index+1 >= len(cfgList) {
			return nil
		}
		return cfgList[index+1]
	}
	return nil
}

func IsCityAgeUpReached(currentId int32, targetId int32) bool {
	currentIndex := int32(-1)
	targetIndex := int32(-1)
	for index, cfg := range GetAllCityAgeUpCfg() {
		if cfg.Id == currentId {
			currentIndex = int32(index)
		}
		if cfg.Id == targetId {
			targetIndex = int32(index)
		}
	}
	return currentIndex >= 0 && targetIndex >= 0 && currentIndex >= targetIndex
}

func (c *CityAgeUpCfg) GetGroupTasks(groupIndex int32) []int32 {
	if c == nil {
		return nil
	}
	switch groupIndex {
	case 1:
		return c.UnlockAge1
	case 2:
		return c.UnlockAge2
	case 3:
		return c.UnlockAge3
	case 4:
		return c.UnlockAge4
	case 5:
		return c.UnlockAge5
	default:
		return nil
	}
}

func (c *CityAgeUpCfg) GetGroupDrop(groupIndex int32) int32 {
	if c == nil || groupIndex < 1 || groupIndex > int32(len(c.Drop)) {
		return 0
	}
	return c.Drop[groupIndex-1]
}
