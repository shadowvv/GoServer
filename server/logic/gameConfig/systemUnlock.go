package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("systemUnlock", &SystemUnlockCfgLoader{})
}

type SystemUnlockCfgLoader struct {
	temp1 map[int32]*SystemUnlockCfg
}

var _ configLoaderInterface = (*SystemUnlockCfgLoader)(nil)

func (s *SystemUnlockCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/systemUnlock.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*SystemUnlockCfg)
	for _, row := range rawData["systemUnlock"] {
		var v SystemUnlockCfg
		v.Id = ParseInt(row["id"])
		v.ParentFunction = ParseInt(row["parentFunction"])
		v.UnlockId = ParseIntArray(row["unlockId"])
		v.ShowId = ParseIntArray(row["showId"])
		v.AreaLimit = ParseIntArray(row["areaLimit"])
		v.UnlockReward = ParseItem(row["unlockReward"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *SystemUnlockCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error invalid ID:%d", id))
		}
		if !enum.IsValidFunctionId(v.Id) {
			return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error function id:%d", id))
		}
		for _, unlockId := range v.UnlockId {
			if GetUnlockCfg(unlockId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error invalid unlockId:%d,config:%d", unlockId, id))
			}
		}
		for _, showId := range v.ShowId {
			if GetUnlockCfg(showId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error invalid showId:%d,config:%d", showId, id))
			}
		}
		// TODO: 地区限制
		if v.UnlockReward != nil {
			if GetItemCfg(v.UnlockReward.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load systemUnlock error invalid unlockReward:%d,config:%d", v.UnlockReward.ID, id))
			}
		}
	}
	return nil
}

func (s *SystemUnlockCfgLoader) apply() {
	systemUnlock.Store(s.temp1)
}

var systemUnlock atomic.Value

type SystemUnlockCfg struct {
	// 功能id
	Id int32 `json:"id"`
	// 父功能id
	ParentFunction int32 `json:"parentFunction"`
	// 解锁条件
	UnlockId []int32 `json:"unlockId"`
	// 显示条件
	ShowId []int32 `json:"showId"`
	// 地区限制
	AreaLimit []int32 `json:"areaLimit"`
	// 解锁奖励
	UnlockReward *ItemConfig `json:"unlockReward"`
}

func GetSystemUnlockCfg(id int32) *SystemUnlockCfg {
	cfgMap := systemUnlock.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SystemUnlockCfg)[id]
}

func GetAllSystemUnlockCfg() map[int32]*SystemUnlockCfg {
	cfgMap := systemUnlock.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SystemUnlockCfg)
}
