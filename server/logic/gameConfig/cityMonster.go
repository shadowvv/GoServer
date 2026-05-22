package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("cityMonster", &CityMonsterCfgLoader{})
}

type CityMonsterCfgLoader struct {
	temp1 map[int32]*CityMonsterCfg
}

var _ configLoaderInterface = (*CityMonsterCfgLoader)(nil)

func (s *CityMonsterCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/cityMonster.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CityMonsterCfg)
	for _, row := range rawData["cityMonster"] {
		var v CityMonsterCfg
		v.Id = ParseInt(row["id"])
		v.Lv = ParseInt(row["lv"])
		v.Quality = ParseInt(row["quality"])
		v.MonsterPower = ParseInt(row["monsterPower"])
		v.DropId = ParseInt(row["dropId"])
		v.Time = ParseInt(row["time"])
		v.Energy = ParseItem(row["energy"])
		v.Max = ParseInt(row["max"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *CityMonsterCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid ID:%d", id))
		}
		if v.MonsterPower <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid MonsterPower:%d,configId:%d", v.MonsterPower, v.Id))
		}
		if v.DropId != 0 && GetDropCfg(v.DropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid DropId:%d,configId:%d", v.DropId, v.Id))
		}
		if v.Time <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid Time:%d,configId:%d", v.Time, v.Id))
		}
		if v.Energy == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid Energy,configId:%d", v.Id))
		}
		if v.Max <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityMonster error invalid Max:%d,configId:%d", v.Max, v.Id))
		}
	}
	return nil
}

func (s *CityMonsterCfgLoader) apply() {
	cityMonster.Store(s.temp1)
}

var cityMonster atomic.Value

type CityMonsterCfg struct {
	// id
	Id int32 `json:"id"`
	// 等级
	Lv int32 `json:"lv"`
	// 品质
	Quality int32 `json:"quality"`
	// 推荐战力
	MonsterPower int32 `json:"monsterPower"`
	// 奖励物品
	DropId int32 `json:"dropId"`
	// 派遣时间/秒
	Time int32 `json:"time"`
	// 疲劳值消耗
	Energy *ItemConfig `json:"energy"`
	// 每日上限
	Max int32 `json:"max"`
}

func GetCityMonsterCfg(id int32) *CityMonsterCfg {
	cfgMap := cityMonster.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CityMonsterCfg)[id]
}
