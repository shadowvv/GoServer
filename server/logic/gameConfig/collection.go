package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("collection", &CollectionCfgLoader{})
}

var collectionMap atomic.Value                    // map[int32]map[int32]*collectionEntityCfg
var EntryAttrNumByEntryIdAndLevelMap atomic.Value // map[int32]map[int32]map[int32]int64

type CollectionCfgLoader struct {
	temp1 map[int32]*EntryCfg
	temp2 map[int32]*CollectionMainCfg
	temp3 map[int32]*EntryConsumeCfg
}

var _ configLoaderInterface = (*CollectionCfgLoader)(nil)

func (s *CollectionCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/collection.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*EntryCfg)
	for _, row := range rawData["entry"] {
		var v EntryCfg
		v.Id = ParseInt(row["id"])
		v.Name = ParseInt(row["name"])
		v.Quality = ParseInt(row["quality"])
		v.Tendency = ParseIntArray(row["tendency"])
		v.Lvcap = ParseIntMatrix(row["lvcap"])
		v.MainId = ParseIntArray(row["main_id"])
		v.Attrid = ParseIntArray(row["attrid"])
		v.AttrInitial = ParseIntArray(row["attrInitial"])
		v.AttrAdded = ParseIntMatrix(row["attrAdded"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load entry error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*CollectionMainCfg)
	for _, row := range rawData["main"] {
		var v CollectionMainCfg
		v.Id = ParseInt(row["id"])
		v.Belonging = ParseInt(row["belonging"])
		v.CollectionLv = ParseInt(row["collectionLv"])
		v.NextId = ParseInt(row["nextId"])
		v.ItemId = ParseInt(row["itemId"])
		v.Spid = ParseInt(row["spid"])
		v.ExchangedNum = ParseInt(row["exchangedNum"])
		v.Upgrade1 = ParseInt(row["upgrade1"])
		v.Upgrade2 = ParseIntArray(row["upgrade2"])
		v.Upgrade2Num = ParseIntArray(row["upgrade2Num"])
		v.Attrid = ParseIntArray(row["attrid"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.Unlock = ParseInt(row["unlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load main error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*EntryConsumeCfg)
	for _, row := range rawData["entryConsume"] {
		var v EntryConsumeCfg
		v.Id = ParseInt(row["id"])
		v.Item = ParseItemArray(row["item"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load entryConsume error duplicate LEVEL:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	return nil
}

func (s *CollectionCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load entry error invalid ID:%d", id))
		}
		for index, v := range v.Lvcap {
			if len(v) != 2 {
				return errors.New(fmt.Sprintf("[gameConfig] load entry error invalid lvcap:%d", index))
			}
		}
		for index, value := range v.AttrAdded {
			if len(v.Attrid) != len(value) {
				return errors.New(fmt.Sprintf("[gameConfig] load entry error invalid AttrAdded:%d", index))
			}
		}
		if len(v.Lvcap) != len(v.AttrAdded) {
			return errors.New(fmt.Sprintf("[gameConfig] load entry error invalid Lvcap and AttrAdded:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load main error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *CollectionCfgLoader) apply() {
	entry.Store(s.temp1)
	main.Store(s.temp2)
	entryConsume.Store(s.temp3)
	InitCollectionConfig()
	InitEntryAttrInfo()
}

var entry atomic.Value
var main atomic.Value
var entryConsume atomic.Value

type EntryCfg struct {
	// 词条id
	Id int32 `json:"id"`
	// 词条名称
	Name int32 `json:"name"`
	// 词条品质
	Quality int32 `json:"quality"`
	// 词条倾向
	Tendency []int32 `json:"tendency"`
	// 等级上限
	Lvcap [][]int32 `json:"lvcap"`
	// 激活关联藏品id
	MainId []int32 `json:"main_id"`
	// 属性
	Attrid []int32 `json:"attrid"`
	// 起始属性
	AttrInitial []int32 `json:"attrInitial"`
	// 增加值
	AttrAdded [][]int32 `json:"attrAdded"`
}

type EntryConsumeCfg struct {
	// 词条等级
	Id int32 `json:"id"`
	// 升级消耗材料
	Item []*ItemConfig `json:"item"`
}

type CollectionMainCfg struct {
	// 藏品id
	Id int32 `json:"id"`
	// 藏品归属
	Belonging int32 `json:"belonging"`
	// 藏品星级
	CollectionLv int32 `json:"collectionLv"`
	// 下一星级id
	NextId int32 `json:"nextId"`
	// 对应物品
	ItemId int32 `json:"itemId"`
	// 对应碎片
	Spid int32 `json:"spid"`
	// 物品兑换碎片数量
	ExchangedNum int32 `json:"exchangedNum"`
	// 升级条件1
	Upgrade1 int32 `json:"upgrade1"`
	// 同品质碎片
	Upgrade2 []int32 `json:"upgrade2"`
	// 品质碎片数量
	Upgrade2Num []int32 `json:"upgrade2Num"`
	// 属性
	Attrid []int32 `json:"attrid"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
}

func GetEntryCfg(id int32) *EntryCfg {
	cfgMap := entry.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EntryCfg)[id]
}

func GetEntryConsumeCfg(id int32) *EntryConsumeCfg {
	cfgMap := entryConsume.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EntryConsumeCfg)[id]
}

func GetCollectionEntityCfg(id int32) *CollectionMainCfg {
	cfgMap := main.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CollectionMainCfg)[id]
}

func InitCollectionConfig() error {
	cfgMap := main.Load()
	if cfgMap == nil {
		return errors.New("[gameConfig] collection config not loaded")
	}
	newCfgMap := make(map[int32]map[int32]*CollectionMainCfg)
	for id, v := range cfgMap.(map[int32]*CollectionMainCfg) {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load collection entity error invalid ID:%d", id))
		}
		if newCfgMap[v.Belonging] == nil {
			newCfgMap[v.Belonging] = make(map[int32]*CollectionMainCfg)
		}
		newCfgMap[v.Belonging][v.CollectionLv] = v
	}
	collectionMap.Store(newCfgMap)
	return nil
}

func GetCollectionMainCfgByAtrAndLevel(attribution, level int32) *CollectionMainCfg {
	cfgMap := collectionMap.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]map[int32]*CollectionMainCfg)[attribution][level]
}

func InitEntryAttrInfo() error {
	cfgMap := entry.Load()
	if cfgMap == nil {
		return errors.New("entry config is nil")
	}
	newMap := make(map[int32]map[int32]map[int32]int64)
	for _, v := range cfgMap.(map[int32]*EntryCfg) {
		if newMap[v.Id] == nil {
			newMap[v.Id] = make(map[int32]map[int32]int64)
		}
		for capId, value := range v.Lvcap {
			for i := value[0]; i <= value[1]; i++ {
				if newMap[v.Id][i] == nil {
					newMap[v.Id][i] = make(map[int32]int64)
				}
				for index, attrId := range v.Attrid {
					if i == 1 {
						newMap[v.Id][i][attrId] = int64(v.AttrInitial[index])
					} else {
						newMap[v.Id][i][attrId] = newMap[v.Id][i-1][attrId] + int64(v.AttrAdded[capId][index])
					}
				}
			}
		}
	}
	EntryAttrNumByEntryIdAndLevelMap.Store(newMap)
	return nil
}

func GetEntryAttrInfoByEntryIdAndLevel(entryId int32, level int32, attrId int32) int64 {
	v := EntryAttrNumByEntryIdAndLevelMap.Load()
	if v == nil {
		return 0
	}
	return v.(map[int32]map[int32]map[int32]int64)[entryId][level][attrId]
}
