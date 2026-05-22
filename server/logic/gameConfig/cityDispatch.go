package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("cityDispatch", &CityDispatchCfgLoader{})
}

type CityDispatchCfgLoader struct {
	temp1 map[int32]*CityDispatchCfg
}

var _ configLoaderInterface = (*CityDispatchCfgLoader)(nil)

func (s *CityDispatchCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/cityDispatch.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CityDispatchCfg)
	for _, row := range rawData["cityDispatch"] {
		var v CityDispatchCfg
		v.Id = ParseInt(row["id"])
		v.Area = ParseInt(row["area"])
		v.Name = ParseInt(row["name"])
		v.Level = ParseInt(row["level"])
		v.Unlock = ParseInt(row["unlock"])
		v.MonsterPoint = ParseIntArray(row["monsterPoint"])
		v.AllMonsterNum = ParseInt(row["allMonsterNum"])
		v.CityMonsterID = ParseIntArray(row["cityMonsterID"])
		v.Probability = ParseIntArray(row["probability"])
		v.Cd = ParseInt(row["cd"])
		v.Drop1 = ParseInt(row["drop1"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *CityDispatchCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid ID:%d", id))
		}
		if v.Area <= 0 || v.Area > 3 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid Area:%d,configId:%d", v.Area, v.Id))
		}
		if v.Level <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid Level:%d,configId:%d", v.Level, v.Id))
		}
		if v.Unlock != 0 && GetUnlockCfg(v.Unlock) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid Unlock:%d,configId:%d", v.Unlock, v.Id))
		}
		if v.AllMonsterNum <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid AllMonsterNum:%d,configId:%d", v.AllMonsterNum, v.Id))
		}
		if int32(len(v.MonsterPoint)) < v.AllMonsterNum {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid MonsterPoint length:%d,configId:%d", len(v.MonsterPoint), v.Id))
		}
		if len(v.CityMonsterID) != len(v.Probability) {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid CityMonsterID length:%d,configId:%d", len(v.CityMonsterID), v.Id))
		}
		for i := 0; i < len(v.CityMonsterID); i++ {
			if GetCityMonsterCfg(v.CityMonsterID[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid CityMonsterID:%d,configId:%d", v.CityMonsterID[i], v.Id))
			}
		}
		if v.Cd <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid Cd:%d,configId:%d", v.Cd, v.Id))
		}
		if v.Drop1 != 0 && GetDropCfg(v.Drop1) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityDispatch error invalid Drop1:%d,configId:%d", v.Drop1, v.Id))
		}
	}
	return nil
}

func (s *CityDispatchCfgLoader) apply() {
	cityDispatch.Store(s.temp1)
}

var cityDispatch atomic.Value

type CityDispatchCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 区域
	Area int32 `json:"area"`
	// 名称
	Name int32 `json:"name"`
	// 等级
	Level int32 `json:"level"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
	// 刷怪点
	MonsterPoint []int32 `json:"monsterPoint"`
	// 同时存在的怪物数量
	AllMonsterNum int32 `json:"allMonsterNum"`
	// 怪物id
	CityMonsterID []int32 `json:"cityMonsterID"`
	// 权重（万分比）
	Probability []int32 `json:"probability"`
	// 重置时间/秒
	Cd int32 `json:"cd"`
	// 基础奖励
	Drop1 int32 `json:"drop1"`
}

func (c *CityDispatchCfg) RandomMonsterId(monsterCount map[int32]int32) int32 {
	candidateIDs := make([]int32, 0, len(c.CityMonsterID))
	candidateWeights := make([]int32, 0, len(c.Probability))
	for i := 0; i < len(c.CityMonsterID); i++ {
		monsterCfg := GetCityMonsterCfg(c.CityMonsterID[i])
		if monsterCfg == nil {
			logger.ErrorBySprintf("GetCityMonsterCfg() monsterCfg == nil")
			continue
		}
		if monsterCount[c.CityMonsterID[i]] >= monsterCfg.Max {
			continue
		}
		weight := c.Probability[i]
		if weight <= 0 {
			continue
		}
		candidateIDs = append(candidateIDs, c.CityMonsterID[i])
		candidateWeights = append(candidateWeights, weight)
	}

	totalWeight := int32(0)
	for _, weight := range candidateWeights {
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return 0
	}

	randWeight := tool.RandInt32(0, totalWeight-1)
	for i, weight := range candidateWeights {
		if randWeight < weight {
			return candidateIDs[i]
		}
		randWeight -= weight
	}
	return 0
}

func GetCityDispatchCfg(battleId int32, level int32) *CityDispatchCfg {
	cfgMap := cityDispatch.Load()
	if cfgMap == nil {
		return nil
	}
	for _, v := range cfgMap.(map[int32]*CityDispatchCfg) {
		if v.Area == battleId && v.Level == level {
			return v
		}
	}
	return nil
}

func GetAllCityDispatchCfg() map[int32]*CityDispatchCfg {
	cfgMap := cityDispatch.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CityDispatchCfg)
}
