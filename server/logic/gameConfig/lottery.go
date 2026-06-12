package gameConfig

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("lottery", &LotteryCfgLoader{})
}

type LotteryCfgLoader struct {
	temp1 map[int32]*SummonPoolCfg
}

var _ configLoaderInterface = (*LotteryCfgLoader)(nil)

type LotteryCfg struct {
	Num             int32
	DropGroupIdList []int32
}

func StrToLotteryCfg(s string) *LotteryCfg {
	parts := strings.Split(s, "~")
	if len(parts) != 2 {
		return nil
	}
	num, err1 := strconv.Atoi(parts[0])
	parts1 := strings.Split(parts[1], "|")
	dropGroupId := make([]int32, 0)
	for _, id := range parts1 {
		id, err := strconv.Atoi(id)
		if err != nil {
			return nil
		}
		dropGroupId = append(dropGroupId, int32(id))
	}
	if err1 != nil {
		return nil
	}
	return &LotteryCfg{Num: int32(num), DropGroupIdList: dropGroupId}
}

func ParseLotteryCfgArray(s string) []*LotteryCfg {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	arr := make([]*LotteryCfg, 0)
	for _, p := range parts {
		cfg := StrToLotteryCfg(strings.TrimSpace(p))
		if cfg != nil {
			arr = append(arr, cfg)
		}
	}
	return arr
}

func (s *LotteryCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/lottery.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*SummonPoolCfg)
	for _, row := range rawData["summonPool"] {
		var v SummonPoolCfg
		v.Id = ParseInt(row["id"])
		v.Gashatype = ParseInt(row["gashatype"])
		v.DropGroupId1 = ParseIntArray(row["dropGroupId1"])
		v.Weight1 = ParseIntArray(row["weight1"])
		v.DrawNum = ParseInt(row["drawNum"])
		v.DropGroupId2 = ParseIntArray(row["dropGroupId2"])
		v.Weight2 = ParseIntArray(row["weight2"])
		v.DropToken = ParseInt(row["dropToken"])
		v.FirstDropFree = ParseInt(row["firstDropFree"])
		v.Guarantees = StrToLotteryCfg(row["guarantees"])
		v.GuaranteesWeight = ParseIntArray(row["guaranteesWeight"])
		v.Guarantees1 = ParseLotteryCfgArray(row["guarantees1"])
		v.Guarantees1Weight = ParseIntMatrix(row["guarantees1Weight"])
		v.Guarantees2 = ParseLotteryCfgArray(row["guarantees2"])
		v.Guarantees2Weight = ParseIntMatrix(row["guarantees2Weight"])
		v.Guarantees2Type = ParseInt(row["guarantees2Type"])
		v.ActModID = ParseInt(row["actModID"])
		v.UnlockID = ParseInt(row["unlockID"])
		v.LuckyGuarantees = ParseLotteryCfgArray(row["luckGuarantees"])
		v.LuckyGuaranteesWeight = ParseIntMatrix(row["luckGuaranteesWeight"])
		if v.Id <= 0 {
			continue
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *LotteryCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load summonPool error invalid ID:%d", id))
		}
		if len(v.DropGroupId1) != len(v.Weight1) || len(v.DropGroupId2) != len(v.Weight2) {
			return errors.New(fmt.Sprintf("[gameConfig] load summonPool error invalid DropGroupId or Weight ID:%d", id))
		}
		if len(v.LuckyGuaranteesWeight) != len(v.LuckyGuarantees) {
			return errors.New(fmt.Sprintf("[gameConfig] load summonPool error invalid LuckyGuaranteesWeight ID:%d", id))
		}
		// 如果有奖励式保底，则不能有单次保底和循环保底
		if len(v.LuckyGuarantees) > 0 && (len(v.Guarantees1) > 0 || len(v.Guarantees2) > 0) {
			return errors.New(fmt.Sprintf("[gameConfig] LuckyGuarantees is exist and can not have Guarantees1 or Guarantees2 lotteryId ID:%d", id))
		}
	}
	return nil
}

func (s *LotteryCfgLoader) apply() {
	summonPool.Store(s.temp1)
}

var summonPool atomic.Value

type SummonPoolCfg struct {
	// 卡池模板
	Id int32 `json:"id"`
	// 抽奖类型
	Gashatype int32 `json:"gashatype"`
	// 奖池掉落
	DropGroupId1 []int32 `json:"dropGroupId1"`
	// 权重
	Weight1 []int32 `json:"weight1"`
	// 抽取限制数
	DrawNum int32 `json:"drawNum"`
	// 奖池掉落
	DropGroupId2 []int32 `json:"dropGroupId2"`
	// 权重
	Weight2 []int32 `json:"weight2"`
	// 抽奖券使用ID
	DropToken int32 `json:"dropToken"`
	//首次抽取免费次数
	FirstDropFree int32 `json:"firstDropFree"`
	// 保底抽数
	Guarantees *LotteryCfg `json:"guarantees"`
	// 保底抽数权重
	GuaranteesWeight []int32 `json:"guaranteesWeight"`
	// 奖励式保底
	LuckyGuarantees []*LotteryCfg `json:"luckGuarantees"`
	// 奖励式保底权重
	LuckyGuaranteesWeight [][]int32 `json:"luckGuaranteesWeight"`
	// 单次保底抽数1
	Guarantees1 []*LotteryCfg `json:"guarantees1"`
	// 单次保底卡池权重
	Guarantees1Weight [][]int32 `json:"guarantees1Weight"`
	// 循环保底抽数2
	Guarantees2 []*LotteryCfg `json:"guarantees2"`
	//循环保底卡池权重
	Guarantees2Weight [][]int32 `json:"guarantees2Weight"`
	// 循环保底类型
	Guarantees2Type int32 `json:"guarantees2Type"`
	// 活动模板
	ActModID int32 `json:"actModID"`
	// 抽取条件
	UnlockID int32 `json:"unlockID"`
}

func GetSummonPoolCfg(id int32) *SummonPoolCfg {
	cfgMap := summonPool.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SummonPoolCfg)[id]
}

func WeightedRandomChoice(nums []int32, weights []int32) int32 {
	// 计算总权重
	total := int32(0)
	for _, w := range weights {
		total += w
	}

	// 生成随机数
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(int(total))

	// 查找对应的数字
	cumulative := int32(0)
	for i, w := range weights {
		cumulative += w
		if r < int(cumulative) {
			return nums[i]
		}
	}

	// 如果因为浮点精度问题没找到，返回最后一个
	return nums[len(nums)-1]
}

func CheckLotterIdIsGuarantess(id int32, dropGroupId int32, onceGuarantees, loopGuarantees bool) int32 {
	cfg := GetSummonPoolCfg(id)
	if cfg == nil {
		return 0
	}
	if onceGuarantees {
		for _, v := range cfg.Guarantees1 {
			for _, id := range v.DropGroupIdList {
				if id == dropGroupId {
					return 1
				}
			}
		}
	}
	if loopGuarantees {
		for _, v := range cfg.Guarantees2 {
			for _, id := range v.DropGroupIdList {
				if id == dropGroupId {
					return 2
				}
			}
		}
	}
	for _, v := range cfg.Guarantees.DropGroupIdList {
		if v == dropGroupId {
			return 3
		}
	}
	return 0
}

func CheckItemIsGuarantess(itemType int32) bool {
	switch itemType {
	case int32(enum.ITEM_TYPE_HERO):
		return true
	case int32(enum.ITEM_TYPE_COLLECTION):
		return true
	}
	return false
}
