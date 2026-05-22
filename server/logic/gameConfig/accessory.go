package gameConfig

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("accessory", &AccessoryCfgLoader{})
}

type group struct {
	Level         int32
	Magnification int32
}

func ParseGroupMatrix(s string) [][]*group {
	if s == "" {
		return nil
	}
	rows := strings.Split(s, ";")
	matrix := make([][]*ItemConfig, 0)
	for _, r := range rows {
		matrix = append(matrix, ParseItemArray(r))
	}
	groups := make([][]*group, len(matrix))
	for i, v := range matrix {
		for _, value := range v {
			groups[i] = append(groups[i], &group{Level: value.ID, Magnification: int32(value.Num)})
		}
	}
	return groups
}

var attr1 atomic.Value // map[int32]map[int32]map[int32]int64
var attr2 atomic.Value // map[int32]map[int32]map[int32]int64

type AccessoryCfgLoader struct {
	temp1 map[int32]*AccessoryBaseCfg
	temp2 map[int32]*AccessoryLevelUpCfg
	temp3 map[int32]*LuckyCfg
	temp4 map[int32]map[int32]*LuckyDropCfg
}

var _ configLoaderInterface = (*AccessoryCfgLoader)(nil)

func (s *AccessoryCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/accessory.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*AccessoryBaseCfg)
	for _, row := range rawData["accessoryBase"] {
		var v AccessoryBaseCfg
		v.Id = ParseInt(row["id"])
		v.Quality = ParseInt(row["quality"])
		v.NextId = ParseInt(row["nextId"])
		v.Rate = ParseInt(row["rate"])
		v.Attr1 = ParseIntArray(row["attr1"])
		v.Attr1Basic = ParseIntArray(row["attr1Basic"])
		v.Attr1Up = ParseIntArray(row["attr1Up"])
		v.Group1 = ParseGroupMatrix(row["group1"])
		v.Attr2 = ParseIntArray(row["attr2"])
		v.Attr2Basic = ParseIntArray(row["attr2Basic"])
		v.Attr2Up = ParseIntArray(row["attr2Up"])
		v.Group2 = ParseGroupMatrix(row["group2"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryBase error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*AccessoryLevelUpCfg)
	for _, row := range rawData["accessoryLevelUp"] {
		var v AccessoryLevelUpCfg
		v.Id = ParseInt(row["id"])
		v.Level = ParseInt(row["level"])
		v.Sum = ParseInt(row["sum"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryLevelUp error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*LuckyCfg)
	for _, row := range rawData["lucky"] {
		var v LuckyCfg
		v.Id = ParseInt(row["id"])
		v.ModId = ParseInt(row["groupId"])
		v.LuckyCoin = ParseInt(row["luckyCoin"])
		v.Num = ParseIntArray(row["num"])
		v.LuckyCoin2 = ParseInt(row["luckyCoin2"])
		v.Num2 = ParseInt(row["num2"])
		v.ActId = ParseInt(row["actId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load lucky error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	s.temp4 = make(map[int32]map[int32]*LuckyDropCfg)
	for _, row := range rawData["luckyDrop"] {
		var v LuckyDropCfg
		v.Id = ParseInt(row["id"])
		v.ModId = ParseInt(row["modId"])
		v.Level = ParseInt(row["level"])
		v.Unlock = ParseInt(row["unlock"])
		v.Num = ParseInt(row["num"])
		v.DropGroup = ParseInt(row["dropGroup"])
		if v.ModId <= 0 {
			continue
		}
		if s.temp4[v.ModId] == nil {
			s.temp4[v.ModId] = make(map[int32]*LuckyDropCfg)
		}

		if s.temp4[v.ModId][v.Level] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load luckyDrop error duplicate ID:%d", v.Id))
		} else {
			s.temp4[v.ModId][v.Level] = &v
		}
	}

	return nil
}

func (s *AccessoryCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.NextId != 0 {
			if s.temp1[v.NextId] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load accessoryBase error duplicate ID:%d", v.Id))
			}
		}
		if v.Attr1 == nil || v.Attr1Basic == nil || v.Attr1Up == nil || len(v.Attr1) != len(v.Attr1Basic) || len(v.Attr1) != len(v.Attr1Up) || len(v.Attr1) != len(v.Group1) {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryBase error invalid Attr1 configId:%d", v.Id))
		}
		if v.Attr2 == nil || v.Attr2Basic == nil || v.Attr2Up == nil || len(v.Attr2) != len(v.Attr2Basic) || len(v.Attr2) != len(v.Attr2Up) || len(v.Attr2) != len(v.Group2) {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryBase error invalid Attr2 configId:%d", v.Id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryBase error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load accessoryLevelUp error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load lucky error invalid ID:%d", id))
		}
		if v.Num == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load lucky error invalid Num is nil ID:%d", id))
		}
	}
	for _, value := range s.temp4 {
		for id, v := range value {
			if v.Id <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load luckyDrop error invalid ID:%d", id))
			}
		}
	}
	return nil
}

func (s *AccessoryCfgLoader) apply() {
	accessoryBase.Store(s.temp1)
	accessoryLevelUp.Store(s.temp2)
	lucky.Store(s.temp3)
	luckyDrop.Store(s.temp4)
	attr1Sum()
	attr2Sum()
}

var accessoryBase atomic.Value
var accessoryLevelUp atomic.Value
var lucky atomic.Value
var luckyDrop atomic.Value

type AccessoryBaseCfg struct {
	// 饰品id
	Id int32 `json:"id"`
	// 品质
	Quality int32 `json:"quality"`
	// 溢出转换id
	NextId int32 `json:"nextId"`
	// 兑换比例
	Rate int32 `json:"rate"`
	// 拥有属性id
	Attr1 []int32 `json:"attr1"`
	// 拥有属性初始值
	Attr1Basic []int32 `json:"attr1Basic"`
	// 每级强化提升
	Attr1Up []int32 `json:"attr1Up"`
	// 等差组
	Group1 [][]*group `json:"group1"`
	// 佩戴属性id
	Attr2 []int32 `json:"attr2"`
	// 佩戴属性初始值
	Attr2Basic []int32 `json:"attr2Basic"`
	// 每级强化提升
	Attr2Up []int32 `json:"attr2Up"`
	// 等差组
	Group2 [][]*group `json:"group2"`
}

type AccessoryLevelUpCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 装备等级
	Level int32 `json:"level"`
	// 升级所需本体总数数量
	Sum int32 `json:"sum"`
}

type LuckyCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 抽奖组
	ModId int32 `json:"modId"`
	// 抽奖货币
	LuckyCoin int32 `json:"luckyCoin"`
	// 消耗个数
	Num []int32 `json:"num"`
	// 兑换货币
	LuckyCoin2 int32 `json:"luckyCoin2"`
	// 兑换个数
	Num2 int32 `json:"num2"`
	// 活动id
	ActId int32 `json:"actId"`
}

type LuckyDropCfg struct {
	// id
	Id int32 `json:"id"`
	// 抽奖组
	ModId int32 `json:"modId"`
	// 等级
	Level int32 `json:"level"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
	// 所需经验
	Num int32 `json:"num"`
	// dropgroup
	DropGroup int32 `json:"dropGroup"`
}

func GetAccessoryBaseCfg(id int32) *AccessoryBaseCfg {
	cfgMap := accessoryBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AccessoryBaseCfg)[id]
}

func GetAccessoryLevelUpCfg(id int32) *AccessoryLevelUpCfg {
	cfgMap := accessoryLevelUp.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AccessoryLevelUpCfg)[id]
}

func GetLuckyCfg(id int32) *LuckyCfg {
	cfgMap := lucky.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*LuckyCfg)[id]
}

func GetLuckyDropCfg(id int32, level int32) *LuckyDropCfg {
	cfgMap := luckyDrop.Load()
	if cfgMap == nil {
		return nil
	}
	m1 := cfgMap.(map[int32]map[int32]*LuckyDropCfg)[id]
	if m1 == nil {
		return nil
	}
	return m1[level]
}

func GetAccessoryAttr1(id, level, attrId int32) int64 {
	m, ok := attr1.Load().(map[int32]map[int32]map[int32]int64)
	if !ok || m == nil || m[id] == nil || m[id][level] == nil {
		return 0
	}
	return m[id][level][attrId]
}

func GetAccessoryAttr2(id, level, attrId int32) int64 {
	m, ok := attr2.Load().(map[int32]map[int32]map[int32]int64)
	if !ok || m == nil || m[id] == nil || m[id][level] == nil {
		return 0
	}
	return m[id][level][attrId]
}

func attr1Sum() {
	newMap := make(map[int32]map[int32]map[int32]int64)
	cfgMapRaw := accessoryBase.Load()
	if cfgMapRaw == nil {
		attr1.Store(newMap)
		return
	}
	cfgMap := cfgMapRaw.(map[int32]*AccessoryBaseCfg)
	for id, v := range cfgMap {
		if newMap[id] == nil {
			newMap[id] = make(map[int32]map[int32]int64)
		}
		if newMap[id][1] == nil {
			newMap[id][1] = make(map[int32]int64)
		}
		for idx, attrId := range v.Attr1 {
			newMap[id][1][attrId] = int64(v.Attr1Basic[idx])
		}
		for index, value := range v.Group1 {
			for ide, val := range value {
				var maxLevel int32
				if ide+1 == len(value) {
					maxLevel = 100
				} else {
					maxLevel = value[ide+1].Level
				}
				for i := val.Level + 1; i <= maxLevel; i++ {
					if newMap[id][i] == nil {
						newMap[id][i] = make(map[int32]int64)
					}
					newMap[id][i][v.Attr1[index]] = newMap[id][i-1][v.Attr1[index]] + int64(v.Attr1Up[index]*val.Magnification/10000)
				}
			}
		}
	}
	attr1.Store(newMap)
}

func attr2Sum() {
	newMap := make(map[int32]map[int32]map[int32]int64)
	cfgMapRaw := accessoryBase.Load()
	if cfgMapRaw == nil {
		attr2.Store(newMap)
		return
	}
	cfgMap := cfgMapRaw.(map[int32]*AccessoryBaseCfg)
	for id, v := range cfgMap {
		if newMap[id] == nil {
			newMap[id] = make(map[int32]map[int32]int64)
		}
		if newMap[id][1] == nil {
			newMap[id][1] = make(map[int32]int64)
		}
		for idx, attrId := range v.Attr2 {
			newMap[id][1][attrId] = int64(v.Attr2Basic[idx])
		}
		for index, value := range v.Group2 {
			for ide, val := range value {
				var maxLevel int32
				if ide+1 == len(value) {
					maxLevel = 100
				} else {
					maxLevel = value[ide+1].Level
				}
				for i := val.Level + 1; i <= maxLevel; i++ {
					if newMap[id][i] == nil {
						newMap[id][i] = make(map[int32]int64)
					}
					newMap[id][i][v.Attr2[index]] = newMap[id][i-1][v.Attr2[index]] + int64(v.Attr2Up[index]*val.Magnification/10000)
				}
			}
		}
	}
	attr2.Store(newMap)
}

func GetAccessoryPower(accessoryId int32, accessoryLevel int32, userLevel int32) int64 {
	attr := make(map[int32]int64)
	if GetAccessoryBaseCfg(accessoryId) == nil {
		return 0
	}
	//for _, v := range GetAccessoryBaseCfg(accessoryId).Attr1 {
	//	attr[v] += GetAccessoryAttr1(accessoryId, accessoryLevel, v)
	//}
	for _, v := range GetAccessoryBaseCfg(accessoryId).Attr2 {
		attr[v] += GetAccessoryAttr2(accessoryId, accessoryLevel, v)
	}
	return int64(GetAttrMapPower(userLevel, attr))
}
