package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("combatCoefficient", &CombatCoefficientCfgLoader{})
}

type CombatCoefficientCfgLoader struct {
	temp1 map[int32]*CombatCoefficientCfg
}

var _ configLoaderInterface = (*CombatCoefficientCfgLoader)(nil)

func (s *CombatCoefficientCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/combatCoefficient.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CombatCoefficientCfg)
	for _, row := range rawData["combatCoefficient"] {
		var v CombatCoefficientCfg
		v.Id = ParseInt(row["id"])
		v.PowerMultiplier1 = ParseInt(row["powerMultiplier1"])
		v.PowerMultiplier2 = ParseInt(row["powerMultiplier2"])
		v.PowerMultiplier3 = ParseInt(row["powerMultiplier3"])
		v.PowerMultiplier4 = ParseInt(row["powerMultiplier4"])
		v.PowerMultiplier5 = ParseInt(row["powerMultiplier5"])
		v.PowerMultiplier6 = ParseInt(row["powerMultiplier6"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *CombatCoefficientCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid ID:%d", id))
		}
		if v.PowerMultiplier1 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier1:%d,configId:%d", v.PowerMultiplier1, v.Id))
		}
		if v.PowerMultiplier2 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier2:%d,configId:%d", v.PowerMultiplier2, v.Id))
		}
		if v.PowerMultiplier3 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier3:%d,configId:%d", v.PowerMultiplier3, v.Id))
		}
		if v.PowerMultiplier4 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier4:%d,configId:%d", v.PowerMultiplier4, v.Id))
		}
		if v.PowerMultiplier5 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier5:%d,configId:%d", v.PowerMultiplier5, v.Id))
		}
		if v.PowerMultiplier6 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load combatCoefficient error invalid PowerMultiplier6:%d,configId:%d", v.PowerMultiplier6, v.Id))
		}
	}
	return nil
}

func (s *CombatCoefficientCfgLoader) apply() {
	attrPowerRatioMap := make(map[int32]map[int32]int64)
	attrPowerRatioMap[1] = make(map[int32]int64)
	attrPowerRatioMap[2] = make(map[int32]int64)
	attrPowerRatioMap[3] = make(map[int32]int64)
	attrPowerRatioMap[4] = make(map[int32]int64)
	attrPowerRatioMap[5] = make(map[int32]int64)
	attrPowerRatioMap[6] = make(map[int32]int64)
	for _, v := range s.temp1 {
		attrPowerRatioMap[1][v.Id] = int64(v.PowerMultiplier1)
		attrPowerRatioMap[2][v.Id] = int64(v.PowerMultiplier2)
		attrPowerRatioMap[3][v.Id] = int64(v.PowerMultiplier3)
		attrPowerRatioMap[4][v.Id] = int64(v.PowerMultiplier4)
		attrPowerRatioMap[5][v.Id] = int64(v.PowerMultiplier5)
		attrPowerRatioMap[6][v.Id] = int64(v.PowerMultiplier6)
	}
	combatCoefficient.Store(attrPowerRatioMap)
}

var combatCoefficient atomic.Value

type CombatCoefficientCfg struct {
	// 属性id
	Id int32 `json:"id"`
	// 黑暗时代
	PowerMultiplier1 int32 `json:"powerMultiplier1"`
	// 石器时代
	PowerMultiplier2 int32 `json:"powerMultiplier2"`
	// 青铜时代
	PowerMultiplier3 int32 `json:"powerMultiplier3"`
	// 铁器时代
	PowerMultiplier4 int32 `json:"powerMultiplier4"`
	// 王国时代
	PowerMultiplier5 int32 `json:"powerMultiplier5"`
	// 帝国时代
	PowerMultiplier6 int32 `json:"powerMultiplier6"`
}

func GetCombatCoefficientCfg(id int32) map[int32]int64 {
	cfgMap := combatCoefficient.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]map[int32]int64)[id]
}
