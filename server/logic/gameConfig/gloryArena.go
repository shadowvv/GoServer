package gameConfig

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("gloryArena", &GloryArenaCfgLoader{})
}

type GloryArenaCfgLoader struct {
	temp1 map[int32]*GloryArenaBaseCfg
	temp2 []*GloryArenaRewardCfg
	temp3 map[int32]*PergameCfg
}

var _ configLoaderInterface = (*GloryArenaCfgLoader)(nil)

func (s *GloryArenaCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/gloryArena.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*GloryArenaBaseCfg)
	for _, row := range rawData["base"] {
		var v GloryArenaBaseCfg
		v.Id = ParseInt(row["id"])
		v.Rank = ParseIntArray(row["rank"])
		v.Battle = ParseIntArray(row["battle"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena base error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp3 = make(map[int32]*PergameCfg)
	for _, row := range rawData["pergame"] {
		var v PergameCfg
		v.Id = ParseInt(row["id"])
		v.Drop = ParseInt(row["drop"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load pergame error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	rewardCfgMap := make(map[int32]*GloryArenaRewardCfg)
	for _, row := range rawData["reward"] {
		var v GloryArenaRewardCfg
		v.Id = ParseInt(row["id"])
		v.Drop = ParseInt(row["drop"])
		if v.Id <= 0 {
			continue
		}
		if rewardCfgMap[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena reward error duplicate ID:%d", v.Id))
		}
		rewardCfgMap[v.Id] = &v
	}
	s.temp2 = make([]*GloryArenaRewardCfg, 0, len(rewardCfgMap))
	for _, cfg := range rewardCfgMap {
		s.temp2 = append(s.temp2, cfg)
	}
	sort.Slice(s.temp2, func(i, j int) bool {
		return s.temp2[i].Id < s.temp2[j].Id
	})

	return nil
}

func (s *GloryArenaCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena base error invalid ID:%d", id))
		}
		if err := checkPerTenThousandRange(v.Rank, "Rank", id, "gloryArena base"); err != nil {
			return err
		}
		if err := checkPerTenThousandRange(v.Battle, "Battle", id, "gloryArena base"); err != nil {
			return err
		}

	}
	for _, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena reward error invalid ID:%d", v.Id))
		}
		if v.Drop <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena reward error invalid Drop:%d,configId:%d", v.Drop, v.Id))
		}
		if GetDropCfg(v.Drop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena reward error invalid Drop:%d,configId:%d", v.Drop, v.Id))
		}
	}
	for _, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena pergame error invalid ID:%d", v.Id))
		}
		if GetDropCfg(v.Drop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load gloryArena pergame error invalid Drop:%d,configId:%d", v.Drop, v.Id))
		}
	}
	return nil
}

func checkPerTenThousandRange(values []int32, field string, configId int32, cfgType string) error {
	if len(values) == 0 {
		return nil
	}
	if len(values) != 2 {
		return errors.New(fmt.Sprintf("[gameConfig] load %s error invalid %s:%v,configId:%d", cfgType, field, values, configId))
	}
	left := values[0]
	right := values[1]
	if left < 0 || right < 0 {
		return errors.New(fmt.Sprintf("[gameConfig] load %s error invalid %s:%v,configId:%d", cfgType, field, values, configId))
	}
	if left > right {
		return errors.New(fmt.Sprintf("[gameConfig] load %s error invalid %s:%v,configId:%d", cfgType, field, values, configId))
	}
	return nil
}

func (s *GloryArenaCfgLoader) apply() {
	gloryArenaBase.Store(s.temp1)
	gloryArenaReward.Store(s.temp2)
	pergame.Store(s.temp3)
}

var gloryArenaBase atomic.Value
var gloryArenaReward atomic.Value
var pergame atomic.Value

type GloryArenaBaseCfg struct {
	// 胜利次数
	Id int32 `json:"id"`
	// 匹配排名
	Rank []int32 `json:"rank"`
	// 匹配战力
	Battle []int32 `json:"battle"`
}

type PergameCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 奖励
	Drop int32 `json:"drop"`
}

type GloryArenaRewardCfg struct {
	// 累计胜场
	Id int32 `json:"id"`
	// 奖励
	Drop int32 `json:"drop"`
}

func GetGloryArenaBaseCfg(id int32) *GloryArenaBaseCfg {
	cfgMap := gloryArenaBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*GloryArenaBaseCfg)[id]
}

func GetAllGloryArenaRewardCfg() []*GloryArenaRewardCfg {
	cfgMap := gloryArenaReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.([]*GloryArenaRewardCfg)
}

func GetGloryArenaPerGameCfg(id int32) *PergameCfg {
	cfgMap := pergame.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PergameCfg)[id]
}
