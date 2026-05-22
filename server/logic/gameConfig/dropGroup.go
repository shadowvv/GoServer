package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("dropGroup", &DropGroupCfgLoader{})
}

type DropGroupCfgLoader struct {
	temp1 map[int32]*DropGroupCfg
}

var _ configLoaderInterface = (*DropGroupCfgLoader)(nil)

func (s *DropGroupCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/dropGroup.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*DropGroupCfg)
	for _, row := range rawData["dropGroup"] {
		var v DropGroupCfg
		v.Id = ParseInt(row["id"])
		v.DropId = ParseIntMatrix(row["dropId"])
		v.DropRate = ParseIntMatrix(row["dropRate"])
		v.Career = ParseIntArray(row["career"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *DropGroupCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid ID:%d", id))
		}
		for _, dropIds := range v.DropId {
			for _, dropId := range dropIds {
				if dropId < 0 || (dropId > 0 && GetDropCfg(dropId) == nil) {
					return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid dropId:%d,configId:%d", dropId, id))
				}
			}
		}
		for _, dropRates := range v.DropRate {
			for _, dropRate := range dropRates {
				if dropRate <= 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid dropRate:%d,configId:%d", dropRate, id))
				}
			}
		}
		for _, career := range v.Career {
			if career <= 0 || GetHeroClassCfg(career) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid career:%d,configId:%d", career, id))
			}
		}
		if len(v.DropId) != len(v.DropRate) {
			return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid dropRate configId:%d", id))
		}
		for i := 0; i < len(v.DropId); i++ {
			if len(v.DropId[i]) != len(v.DropRate[i]) {
				return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid dropRate configId:%d", id))
			}
			if len(v.Career) != 0 && len(v.Career) != len(v.DropId[i]) {
				return errors.New(fmt.Sprintf("[gameConfig] load dropGroup error invalid dropId configId:%d", id))
			}
		}
	}
	return nil
}
func (cfg *DropGroupCfg) Precompute() {

	rowCount := len(cfg.DropId)
	cfg.RowPrefixSum = make([][]int32, rowCount)

	for r := 0; r < rowCount; r++ {
		row := cfg.DropRate[r]
		prefix := make([]int32, len(row))
		var sum int32
		for i, w := range row {
			sum += w
			prefix[i] = sum
		}
		cfg.RowPrefixSum[r] = prefix
	}

	// 职业模式
	if len(cfg.Career) > 0 {
		cfg.ValidCols = make(map[int32][]int)

		for col, career := range cfg.Career {
			cfg.ValidCols[career] = append(cfg.ValidCols[career], col)
		}

		// 每个职业、每一行都要一个 prefix sum
		cfg.RowCareerPrefixSum = make([][]int32, rowCount)
		for r := 0; r < rowCount; r++ {
			cfg.RowCareerPrefixSum[r] = make([]int32, len(cfg.DropRate[r]))
		}
	}
}

func (s *DropGroupCfgLoader) apply() {
	for _, cfg := range s.temp1 {
		cfg.Precompute()
	}
	dropGroup.Store(s.temp1)
}

var dropGroup atomic.Value

type DropGroupCfg struct {
	// 序号
	Id int32 `json:"id"`
	// drop掉落组
	DropId [][]int32 `json:"dropId"`
	// drop掉落组权重
	DropRate [][]int32 `json:"dropRate"`
	// 职业判断（可多填）
	Career []int32 `json:"career"`

	RowPrefixSum       [][]int32       // 普通模式（每一行的 prefix sum）
	ValidCols          map[int32][]int // career → valid columns
	RowCareerPrefixSum [][]int32       // career 模式，每行每个职业的 prefix sum
}

func GetDropGroupCfg(id int32) *DropGroupCfg {
	cfgMap := dropGroup.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DropGroupCfg)[id]
}

func DropGroupItems(dropGroupId int32, careers []int32) []*ItemConfig {
	items := make([]*ItemConfig, 0, 4)

	cfg := GetDropGroupCfg(dropGroupId)
	if cfg == nil {
		return items
	}

	rowCount := len(cfg.DropId)

	// -----------------------
	// 普通模式（无职业限制）
	// -----------------------
	if len(cfg.Career) == 0 {

		for r := 0; r < rowCount; r++ {
			prefix := cfg.RowPrefixSum[r]
			if len(prefix) == 0 {
				continue
			}

			total := prefix[len(prefix)-1]
			rand := tool.RandInt32(1, total)

			col := binarySearchPrefix(prefix, rand)

			dropId := cfg.DropId[r][col]
			if dropId > 0 && GetDropCfg(dropId) != nil {
				items = append(items, Drop(dropId)...)
			}
		}

		return items
	}

	// -----------------------
	// 职业模式
	// -----------------------

	// 合并所有有效职业列
	validCols := make([]int, 0, len(careers)*2)
	seen := map[int]bool{}

	for _, c := range careers {
		if cols := cfg.ValidCols[c]; len(cols) > 0 {
			for _, col := range cols {
				if !seen[col] {
					seen[col] = true
					validCols = append(validCols, col)
				}
			}
		}
	}

	if len(validCols) == 0 {
		return items
	}

	// 逐行掉落
	for r := 0; r < rowCount; r++ {

		// 构建仅包含有效列的 prefix sum
		var sum int32
		prefix := make([]int32, len(validCols))

		for i, col := range validCols {
			w := cfg.DropRate[r][col]
			sum += w
			prefix[i] = sum
		}

		if sum <= 0 {
			continue
		}

		rand := tool.RandInt32(1, sum)
		idx := binarySearchPrefix(prefix, rand)
		col := validCols[idx]

		dropId := cfg.DropId[r][col]
		if dropId > 0 && GetDropCfg(dropId) != nil {
			items = append(items, Drop(dropId)...)
		}
	}

	return items
}

func binarySearchPrefix(prefix []int32, target int32) int {
	low, high := 0, len(prefix)-1
	for low <= high {
		mid := (low + high) >> 1
		if target <= prefix[mid] {
			if mid == 0 || target > prefix[mid-1] {
				return mid
			}
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return len(prefix) - 1
}
