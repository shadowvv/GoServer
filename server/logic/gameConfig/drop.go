package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("drop", &DropCfgLoader{})
}

type DropCfgLoader struct {
	temp1 map[int32]*DropCfg
}

var _ configLoaderInterface = (*DropCfgLoader)(nil)

func (s *DropCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/drop.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*DropCfg)
	for _, row := range rawData["drop"] {
		var v DropCfg
		v.Id = ParseInt(row["id"])
		v.FixedItem = ParseItemArray(row["fixedItem"])
		v.ProbabilityItem = ParseItemMatrix(row["probabilityItem"])
		v.ProbabilityItemWeight = ParseIntMatrix(row["probabilityItemWeight"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load drop error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *DropCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid ID:%d", id))
		}

		for _, item := range v.FixedItem {
			if item.ID < 0 || GetItemCfg(item.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid fixedItem configId:%d", id))
			}
			if item.Num < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid fixedItem configId:%d", id))
			}
		}
		for _, group := range v.ProbabilityItem {
			for _, item := range group {
				if item.ID < 0 || (item.ID != 0 && GetItemCfg(item.ID) == nil) {
					return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid probabilityItem configId:%d,item not exist itemId:%d", id, item.ID))
				}
				if item.Num < 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid probabilityItem configId:%d", id))
				}
			}
		}
		if len(v.ProbabilityItem) != len(v.ProbabilityItemWeight) {
			return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid probabilityItemWeight configId:%d", id))
		}
		for i := 0; i < len(v.ProbabilityItemWeight); i++ {
			if len(v.ProbabilityItem[i]) != len(v.ProbabilityItemWeight[i]) {
				return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid probabilityItemWeight configId:%d", id))
			}
			for j := 0; j < len(v.ProbabilityItemWeight[i]); j++ {
				if v.ProbabilityItemWeight[i][j] <= 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load drop error invalid probabilityItemWeight configId:%d", id))
				}
			}
		}

		// 构建概率组
		v.buildGroups()
	}
	return nil
}

func (s *DropCfgLoader) apply() {
	drop.Store(s.temp1)

}

var drop atomic.Value

type DropCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 固定掉落
	FixedItem []*ItemConfig `json:"fixedItem"`
	// 概率掉落
	ProbabilityItem [][]*ItemConfig `json:"probabilityItem"`
	// 掉率权重
	ProbabilityItemWeight [][]int32 `json:"probabilityItemWeight"`
	// ===== 新增预处理结构 =====
	Groups []*DropGroup
}

type DropGroup struct {
	Items       []*ItemConfig
	Weights     []int32
	TotalWeight int32
}

func (v *DropCfg) buildGroups() {
	groupCount := len(v.ProbabilityItem)
	if groupCount == 0 {
		return
	}

	v.Groups = make([]*DropGroup, 0, groupCount)

	for i := 0; i < groupCount; i++ {
		items := v.ProbabilityItem[i]
		weights := v.ProbabilityItemWeight[i]

		if len(items) != len(weights) {
			// 这里你可以选择 panic 或忽略
			continue
		}

		g := &DropGroup{
			Items:   items,
			Weights: weights,
		}

		// 计算总权重
		var total int32
		for _, w := range weights {
			total += w
		}
		g.TotalWeight = total

		v.Groups = append(v.Groups, g)
	}
}

func GetDropCfg(id int32) *DropCfg {
	cfgMap := drop.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DropCfg)[id]
}

func Drop(id int32) []*ItemConfig {
	cfg := GetDropCfg(id)
	if cfg == nil {
		return nil
	}
	return cfg.randomDrop()
}

func (cfg *DropCfg) randomDrop() []*ItemConfig {
	if cfg == nil {
		return nil
	}

	// 结果 = 固定掉落 + 每一组抽取一个概率掉落
	var result []*ItemConfig

	// 固定掉落
	result = append(result, cfg.FixedItem...)

	// 概率掉落
	for _, group := range cfg.Groups {
		if group.TotalWeight <= 0 {
			continue
		}

		r := tool.RandInt32(1, group.TotalWeight) // [1, total]
		var sum int32
		for i, w := range group.Weights {
			sum += w
			if r <= sum {
				if group.Items[i] == nil {
					continue
				}
				if group.Items[i].ID <= 0 {
					break
				}
				if group.Items[i].Num <= 0 {
					break
				}
				result = append(result, group.Items[i])
				break
			}
		}
	}

	return result
}

func GetDropAll(id ...int32) []*ItemConfig {
	return GetDropMap(id)
}

func GetDropMap(ids []int32) []*ItemConfig {
	res := make([]*ItemConfig, 0)
	for _, id := range ids {
		res = append(res, Drop(id)...)
	}
	return res
}
