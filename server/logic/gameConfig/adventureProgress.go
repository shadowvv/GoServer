package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("adventureProgress", &AdventureProgressCfgLoader{})
}

type AdventureProgressCfgLoader struct {
	temp1 map[int32]*AdventureProgressCfg
}

var _ configLoaderInterface = (*AdventureProgressCfgLoader)(nil)

func (s *AdventureProgressCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/adventureProgress.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*AdventureProgressCfg)
	for _, row := range rawData["adventureProgress"] {
		var v AdventureProgressCfg
		v.Id = ParseInt(row["id"])
		v.Progress = ParseInt(row["progress"])
		v.Adventure = ParseInt(row["adventure"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load adventureProgress error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *AdventureProgressCfgLoader) checkData() error {
	var maxId int32
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventureProgress error invalid ID:%d", id))
		}
		if id > maxId {
			maxId = id
		}
		if v.Progress <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load adventureProgress error invalid Progress:%d,configId:%d", v.Progress, id))
		}
		if v.Adventure != 0 && GetAdventureCfg(v.Adventure) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load adventureProgress error invalid Adventure:%d,configId:%d", v.Adventure, id))
		}
	}
	for id := int32(1); id <= maxId; id++ {
		cfg := s.temp1[id]
		if cfg == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load adventureProgress error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *AdventureProgressCfgLoader) apply() {
	adventureProgress.Store(s.temp1)
}

var adventureProgress atomic.Value

type AdventureProgressCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 进度值
	Progress int32 `json:"progress"`
	// 副本
	Adventure int32 `json:"adventure"`
}

func GetAdventureProgressCfg(id int32) *AdventureProgressCfg {
	cfgMap := adventureProgress.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AdventureProgressCfg)[id]
}

func GetAdventureProgressCfgByDailyTriggerCount(dailyTriggerCount int32) *AdventureProgressCfg {
	cfgMap := adventureProgress.Load()
	if cfgMap == nil {
		return nil
	}
	allCfg := cfgMap.(map[int32]*AdventureProgressCfg)
	targetId := dailyTriggerCount + 1
	if allCfg[targetId] != nil {
		return allCfg[targetId]
	}
	var maxId int32
	for id := range allCfg {
		if id > maxId {
			maxId = id
		}
	}
	return allCfg[maxId]
}
