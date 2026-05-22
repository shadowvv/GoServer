package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("scene", &SceneCfgLoader{})
}

type SceneCfgLoader struct {
	temp1 map[int32]*SceneCfg
}

var _ configLoaderInterface = (*SceneCfgLoader)(nil)

func (s *SceneCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/scene.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*SceneCfg)
	for _, row := range rawData["scene"] {
		var v SceneCfg
		v.Id = ParseInt(row["id"])
		v.SceneType = ParseInt(row["sceneType"])
		v.IsPlayerHide = ParseInt(row["isPlayerHide"])
		v.DisableItem = ParseIntArray(row["disableItem"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load scene error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *SceneCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load scene error invalid ID:%d", id))
		}
		for _, item := range v.DisableItem {
			if GetItemCfg(item) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load scene error invalid item:%d", item))
			}
		}
	}
	return nil
}

func (s *SceneCfgLoader) apply() {
	scene.Store(s.temp1)
}

var scene atomic.Value

type SceneCfg struct {
	// 场景编号
	Id int32 `json:"id"`
	// 场景类型
	SceneType int32 `json:"sceneType"`
	// 是否显示其他玩家
	IsPlayerHide int32 `json:"isPlayerHide"`
	// 禁用道具
	DisableItem []int32 `json:"disableItem"`
}

func GetSceneCfg(id int32) *SceneCfg {
	cfgMap := scene.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SceneCfg)[id]
}
