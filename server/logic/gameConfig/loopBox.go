package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("loopBox", &LoopBoxCfgLoader{})
}

type LoopBoxCfgLoader struct {
	temp1 map[int32]*BoxPropCfg
	temp2 map[int32]*LevelCfg
	temp3 map[int32]*ProgressRewardCfg
}

var _ configLoaderInterface = (*LoopBoxCfgLoader)(nil)

func (s *LoopBoxCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/loopBox.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*BoxPropCfg)
	for _, row := range rawData["boxProp"] {
		var v BoxPropCfg
		v.Id = ParseInt(row["id"])
		v.LoopBoxPoint = ParseInt(row["loopBoxPoint"])
		v.LoopBoxExp = ParseInt(row["loopBoxExp"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load boxProp error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*LevelCfg)
	for _, row := range rawData["level"] {
		var v LevelCfg
		v.Id = ParseInt(row["id"])
		v.LoopBoxLevel = ParseInt(row["loopBoxLevel"])
		v.LevelUpExp = ParseInt(row["levelUpExp"])
		v.Unlock = ParseInt(row["unlock"])
		v.DropGroupId = ParseIntArray(row["dropGroupId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load level error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*ProgressRewardCfg)
	for _, row := range rawData["progressReward"] {
		var v ProgressRewardCfg
		v.Id = ParseInt(row["id"])
		v.BoxLoopPoint = ParseInt(row["boxLoopPoint"])
		v.BoxLoopUnlock = ParseInt(row["boxLoopUnlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load progressReward error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	return nil
}

func (s *LoopBoxCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load boxProp error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.DropGroupId == nil || len(v.DropGroupId) != 5 {
			return errors.New(fmt.Sprintf("[gameConfig] load boxProp error invalid ID:%d", id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load level error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load progressReward error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *LoopBoxCfgLoader) apply() {
	boxProp.Store(s.temp1)
	level.Store(s.temp2)
	progressReward.Store(s.temp3)
}

var boxProp atomic.Value
var level atomic.Value
var progressReward atomic.Value

type BoxPropCfg struct {
	// ID
	Id int32 `json:"id"`
	// 循环箱价值积分-进度
	LoopBoxPoint int32 `json:"loopBoxPoint"`
	// 循环箱经验-等级
	LoopBoxExp int32 `json:"loopBoxExp"`
}

type LevelCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 循环箱等级
	LoopBoxLevel int32 `json:"loopBoxLevel"`
	// 升下一级所需经验
	LevelUpExp int32 `json:"levelUpExp"`
	// 升级解锁条件
	Unlock int32 `json:"unlock"`
	// 当前等级掉落
	DropGroupId []int32 `json:"dropGroupId"`
}

type ProgressRewardCfg struct {
	// 兑换轮换ID
	Id int32 `json:"Id"`
	// 轮换所需积分
	BoxLoopPoint int32 `json:"boxLoopPoint"`
	// 轮换解锁宝箱ID
	BoxLoopUnlock int32 `json:"boxLoopUnlock"`
}

func GetBoxPropCfg(id int32) *BoxPropCfg {
	cfgMap := boxProp.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BoxPropCfg)[id]
}

func GetLevelCfg(id int32) *LevelCfg {
	cfgMap := level.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*LevelCfg)[id]
}

func GetProgressRewardCfg(id int32) *ProgressRewardCfg {
	cfgMap := progressReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ProgressRewardCfg)[id]
}
