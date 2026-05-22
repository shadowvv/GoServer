package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("city", &CityCfgLoader{})
}

type CityConfig interface {
	GetItem() []*ItemConfig
	GetTime() int32
	GetUnlock() []int32
}

var statueAttrIndexMap atomic.Value    // map[int32]map[int32]int32
var statueAttrInfoByLevel atomic.Value // map[int32]map[int32]map[int32]int64
var statueAttrCfgMap atomic.Value      // map[int32]map[int32]*StatueAttrCfg
var statueAttrLevelMap atomic.Value    // map[int32]map[int32]*StatueAttrCfg
var lumber atomic.Value                // map[int32]*LumberCfg
var furniture atomic.Value             // map[int32]*FurnitureCfg
var furnitureIndex atomic.Value        // map[building]map[type]map[level]*FurnitureCfg

type CityCfgLoader struct {
	// 保持原先 temp 命名风格（按 city.json 子表顺序划分）
	temp1 map[int32]*CityCenterCfg
	temp2 map[int32]*CollectionCfg
	temp3 map[int32]*EquipCraftCfg
	temp4 map[int32]*HeritageStatueCfg
	temp5 map[int32]*PetSanctuaryCfg
	temp6 map[int32]*StatueAttrCfg
	temp7 map[int32]*LumberCfg
	temp8 map[int32]*FurnitureCfg
}

var _ configLoaderInterface = (*CityCfgLoader)(nil)

func (s *CityCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/city.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CityCenterCfg)
	for _, row := range rawData["cityCenter"] {
		var v CityCenterCfg
		v.Id = ParseInt(row["id"])
		v.Age = ParseInt(row["age"])
		v.Period = ParseInt(row["period"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.Item = ParseItemArray(row["item"])
		v.Time = ParseInt(row["time"])
		v.HeroLevel = ParseInt(row["heroLevel"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.EffectType = ParseIntArray(row["effectType"])
		v.EffectPara = ParseIntArray(row["effectPara"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load cityCenter error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*CollectionCfg)
	for _, row := range rawData["collection"] {
		var v CollectionCfg
		v.Id = ParseInt(row["id"])
		v.UnlockId = ParseIntArray(row["unlockId"])
		v.Item = ParseItemArray(row["item"])
		v.Time = ParseInt(row["time"])
		v.AttrId = ParseIntArray(row["attrId"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.UnlockCollect = ParseIntArray(row["unlockCollect"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load collection error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*EquipCraftCfg)
	for _, row := range rawData["equipCraft"] {
		var v EquipCraftCfg
		v.Id = ParseInt(row["id"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.Item = ParseItemArray(row["item"])
		v.Time = ParseInt(row["time"])
		v.Group = ParseIntArray(row["group"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load equipCraft error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	s.temp4 = make(map[int32]*HeritageStatueCfg)
	for _, row := range rawData["heritageStatue"] {
		var v HeritageStatueCfg
		v.Id = ParseInt(row["id"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.Item = ParseItemArray(row["item"])
		v.Time = ParseInt(row["time"])
		v.Attr = ParseIntArray(row["attr"])
		if v.Id <= 0 {
			continue
		}
		if s.temp4[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heritageStatue error duplicate ID:%d", v.Id))
		}
		s.temp4[v.Id] = &v
	}

	s.temp5 = make(map[int32]*PetSanctuaryCfg)
	for _, row := range rawData["petSanctuary"] {
		var v PetSanctuaryCfg
		v.Id = ParseInt(row["id"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.Cost = ParseItemArray(row["cost"])
		v.Time = ParseInt(row["time"])
		v.DropGroupId1 = ParseInt(row["dropGroupId1"])
		v.DropGroupId2 = ParseInt(row["dropGroupId2"])
		v.Img = row["img"]
		v.Spine = row["spine"]
		if v.Id <= 0 {
			continue
		}
		if s.temp5[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petSanctuary error duplicate ID:%d", v.Id))
		}
		s.temp5[v.Id] = &v
	}

	s.temp6 = make(map[int32]*StatueAttrCfg)
	for _, row := range rawData["statueAttr"] {
		var v StatueAttrCfg
		v.Id = ParseInt(row["id"])
		v.HeroClass = ParseInt(row["heroClass"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrLevel = ParseIntArray(row["attrLevel"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.Cost = ParseItemArray(row["cost"])
		v.StatueLevel = ParseInt(row["statueLevel"])
		if v.Id <= 0 {
			continue
		}
		if s.temp6[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load statueAttr error duplicate ID:%d", v.Id))
		}
		s.temp6[v.Id] = &v
	}

	s.temp7 = make(map[int32]*LumberCfg)
	for _, row := range rawData["lumber"] {
		var v LumberCfg
		v.Id = ParseInt(row["id"])
		v.Progress = ParseInt(row["progress"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.Item = ParseItemArray(row["item"])
		v.Time = ParseInt(row["time"])
		v.Output = ParseItemArray(row["output"])
		v.Limit = ParseItemArray(row["limit"])
		v.HeroSlots = ParseInt(row["heroSlots"])
		v.Furniture = ParseInt(row["furniture"])
		if v.Id <= 0 {
			continue
		}
		if s.temp7[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load lumber error duplicate ID:%d", v.Id))
		}
		s.temp7[v.Id] = &v
	}

	s.temp8 = make(map[int32]*FurnitureCfg)
	if furnitureData, ok := rawData["furniture"]; ok {
		for _, row := range furnitureData {
			var v FurnitureCfg
			v.Id = ParseInt(row["id"])
			v.Building = ParseInt(row["building"])
			v.Type = ParseInt(row["type"])
			v.Level = ParseInt(row["level"])
			v.Item = ParseItemArray(row["item"])
			v.EffectType = ParseInt(row["effectType"])
			v.BaseEffect = ParseItemArray(row["baseEffect"])
			v.Progress = ParseInt(row["progress"])
			if v.Id <= 0 {
				continue
			}
			if s.temp8[v.Id] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load furniture error duplicate ID:%d", v.Id))
			}
			s.temp8[v.Id] = &v
		}
	}

	return nil
}

func (s *CityCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load cityCenter error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load collection error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load equipCraft error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp4 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heritageStatue error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp5 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petSanctuary error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp6 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load statueAttr error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp7 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load lumber error invalid ID:%d", id))
		}
	}
	furnitureTypesByBuilding := make(map[int32]map[int32]bool)
	furnitureKeys := make(map[[3]int32]int32)
	for id, v := range s.temp8 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load furniture error invalid ID:%d", id))
		}
		if !enum.IsValidArchitectureType(v.Building) {
			return errors.New(fmt.Sprintf("[gameConfig] load furniture error invalid building:%d ID:%d", v.Building, id))
		}
		if v.Level == 0 {
			continue
		}
		key := [3]int32{v.Building, v.Type, v.Level}
		if oldID, ok := furnitureKeys[key]; ok {
			return errors.New(fmt.Sprintf("[gameConfig] load furniture error duplicate building:%d type:%d level:%d ID:%d oldID:%d", v.Building, v.Type, v.Level, id, oldID))
		}
		furnitureKeys[key] = id
		if furnitureTypesByBuilding[v.Building] == nil {
			furnitureTypesByBuilding[v.Building] = make(map[int32]bool)
		}
		furnitureTypesByBuilding[v.Building][v.Type] = true
	}
	for _, v := range s.temp7 {
		if v.Furniture <= 0 {
			continue
		}
		building := int32(enum.ARCHITECTURE_TYPE_LUMBERYARD)
		if furnitureTypesByBuilding[building] == nil || !furnitureTypesByBuilding[building][v.Furniture] {
			return errors.New(fmt.Sprintf("[gameConfig] load lumber error furniture type %d not found ID:%d", v.Furniture, v.Id))
		}
	}
	return nil
}

func (s *CityCfgLoader) apply() {
	cityCenter.Store(s.temp1)
	collection.Store(s.temp2)
	equipCraft.Store(s.temp3)
	heritageStatue.Store(s.temp4)
	petSanctuary.Store(s.temp5)
	statueAttr.Store(s.temp6)
	lumber.Store(s.temp7)
	furniture.Store(s.temp8)
	InitStatueAttrIndexMap()
	InitStatueAttrInfoByLevel()
	initFurnitureIndex()
}

var cityCenter atomic.Value
var equipCraft atomic.Value
var heritageStatue atomic.Value
var petSanctuary atomic.Value
var statueAttr atomic.Value
var collection atomic.Value

type CityCenterCfg struct {
	// 账号等级
	Id int32 `json:"id"`
	// 时代
	Age int32 `json:"age"`
	// 阶段
	Period int32 `json:"period"`
	// 解锁条件
	Unlock []int32 `json:"unlock"`
	// 升级消耗材料道具id
	Item []*ItemConfig `json:"item"`
	// 升级时长s
	Time int32 `json:"time"`
	// 英雄等级
	HeroLevel int32 `json:"heroLevel"`
	// 属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 解锁效果类型
	EffectType []int32 `json:"effectType"`
	// 解锁效果参数
	EffectPara []int32 `json:"effectPara"`
}

var _ CityConfig = (*CityCenterCfg)(nil)

func GetCityCenterCfg(id int32) *CityCenterCfg {
	cfgMap := cityCenter.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CityCenterCfg)[id]
}

func (c *CityCenterCfg) GetItem() []*ItemConfig {
	return c.Item
}

func (c *CityCenterCfg) GetTime() int32 {
	return c.Time
}

func (c *CityCenterCfg) GetUnlock() []int32 {
	return c.Unlock
}

type HeritageStatueCfg struct {
	// 石像等级
	Id int32 `json:"id"`
	// 解锁条件
	Unlock []int32 `json:"unlock"`
	// 升级消耗材料道具id
	Item []*ItemConfig `json:"item"`
	// 升级时长s
	Time int32 `json:"time"`
	// 解锁属性
	Attr []int32 `json:"attr"`
}

var _ CityConfig = (*HeritageStatueCfg)(nil)

func GetHeritageStatueCfg(id int32) *HeritageStatueCfg {
	cfgMap := heritageStatue.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*HeritageStatueCfg)[id]
}

func (h *HeritageStatueCfg) GetItem() []*ItemConfig {
	return h.Item
}

func (h *HeritageStatueCfg) GetTime() int32 {
	return h.Time
}

func (h *HeritageStatueCfg) GetUnlock() []int32 {
	return h.Unlock
}

type EquipCraftCfg struct {
	// 建筑等级
	Id int32 `json:"id"`
	// 升级条件
	Unlock []int32 `json:"unlock"`
	// 升级消耗
	Item []*ItemConfig `json:"item"`
	// 升级时间
	Time int32 `json:"time"`
	// 解锁装备图纸组
	Group []int32 `json:"group"`
}

var _ CityConfig = (*EquipCraftCfg)(nil)

func GetEquipCraftCfg(id int32) *EquipCraftCfg {
	cfgMap := equipCraft.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipCraftCfg)[id]
}

func (e *EquipCraftCfg) GetItem() []*ItemConfig {
	return e.Item
}

func (e *EquipCraftCfg) GetTime() int32 {
	return e.Time
}

func (e *EquipCraftCfg) GetUnlock() []int32 {
	return e.Unlock
}

type PetSanctuaryCfg struct {
	// 建筑等级
	Id int32 `json:"id"`
	// 升级条件
	Unlock []int32 `json:"unlock"`
	// 升级消耗
	Cost []*ItemConfig `json:"cost"`
	// 升级时间s
	Time int32 `json:"time"`
	// 掉落id
	DropGroupId1 int32 `json:"dropGroupId1"`
	// 特权掉落id
	DropGroupId2 int32 `json:"dropGroupId2"`
	// 建筑图片
	Img string `json:"img"`
	// 建筑spine
	Spine string `json:"spine"`
}

var _ CityConfig = (*PetSanctuaryCfg)(nil)

func GetPetSanctuaryCfg(id int32) *PetSanctuaryCfg {
	cfgMap := petSanctuary.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetSanctuaryCfg)[id]
}

func (p *PetSanctuaryCfg) GetItem() []*ItemConfig {
	return p.Cost
}

func (p *PetSanctuaryCfg) GetTime() int32 {
	return p.Time
}

func (p *PetSanctuaryCfg) GetUnlock() []int32 {
	return p.Unlock
}

type StatueAttrCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 职业
	HeroClass int32 `json:"heroClass"`
	// 属性
	Attr []int32 `json:"attr"`
	// 等级区间
	AttrLevel []int32 `json:"attrLevel"`
	// 每级提升
	AttrNum []int32 `json:"attrNum"`
	// 每级消耗
	Cost []*ItemConfig `json:"cost"`
	// 石像等级
	StatueLevel int32 `json:"statueLevel"`
}

func GetStatueAttrCfg(id int32) *StatueAttrCfg {
	cfgMap := statueAttr.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*StatueAttrCfg)[id]
}

type CollectionCfg struct {
	// 藏品楼等级
	Id int32 `json:"id"`
	// 藏品楼升级条件
	UnlockId []int32 `json:"unlockId"`
	// 升级消耗材料
	Item []*ItemConfig `json:"item"`
	// 升级所需时间
	Time int32 `json:"time"`
	// 属性
	AttrId []int32 `json:"attrId"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 解锁藏品
	UnlockCollect []int32 `json:"unlockCollect"`
}

var _ CityConfig = (*CollectionCfg)(nil)

func GetCollectionCfg(id int32) *CollectionCfg {
	cfgMap := collection.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CollectionCfg)[id]
}

func (c *CollectionCfg) GetItem() []*ItemConfig {
	return c.Item
}

func (c *CollectionCfg) GetTime() int32 {
	return c.Time
}

func (c *CollectionCfg) GetUnlock() []int32 {
	return c.UnlockId
}

func GetStatueAttrIndexMap() map[int32]map[int32]int32 {
	v := statueAttrIndexMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]map[int32]int32)
}

func GetStatueAttrInfoByLevel() map[int32]map[int32]map[int32]int64 {
	v := statueAttrInfoByLevel.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]map[int32]map[int32]int64)
}

func GetStatueAttrCfgMap() map[int32]map[int32]*StatueAttrCfg {
	v := statueAttrCfgMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]map[int32]*StatueAttrCfg)
}

func GetStatueAttrLevelMap() map[int32]map[int32]*StatueAttrCfg {
	v := statueAttrLevelMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]map[int32]*StatueAttrCfg)
}

func InitStatueAttrIndexMap() error {
	cfgMap := statueAttr.Load()
	if cfgMap == nil {
		return errors.New("statueAttr config is nil")
	}
	newMap := make(map[int32]map[int32]int32)
	for _, v := range cfgMap.(map[int32]*StatueAttrCfg) {
		if newMap[v.HeroClass] == nil {
			newMap[v.HeroClass] = make(map[int32]int32)
		}
		for id, i := range v.Attr {
			newMap[v.HeroClass][i] = int32(id)
		}
	}
	statueAttrIndexMap.Store(newMap)
	return nil
}

func InitStatueAttrInfoByLevel() error {
	cfgMap := statueAttr.Load()
	if cfgMap == nil {
		return errors.New("statueAttr config is nil")
	}
	newInfoByLevel := make(map[int32]map[int32]map[int32]int64)
	newCfgMap := make(map[int32]map[int32]*StatueAttrCfg)
	newLevelMap := make(map[int32]map[int32]*StatueAttrCfg)

	classLevelMapList := make(map[int32][]int32)

	for _, v := range cfgMap.(map[int32]*StatueAttrCfg) {
		if classLevelMapList[v.HeroClass] == nil {
			classLevelMapList[v.HeroClass] = make([]int32, 0)
			classLevelMapList[v.HeroClass] = append(classLevelMapList[v.HeroClass], v.AttrLevel[0])
			classLevelMapList[v.HeroClass] = append(classLevelMapList[v.HeroClass], v.AttrLevel[1])
		} else {
			classLevelMapList[v.HeroClass][0] = min(classLevelMapList[v.HeroClass][0], v.AttrLevel[0])
			classLevelMapList[v.HeroClass][1] = max(classLevelMapList[v.HeroClass][1], v.AttrLevel[1])
		}

		if newCfgMap[v.StatueLevel] == nil {
			newCfgMap[v.StatueLevel] = make(map[int32]*StatueAttrCfg)
		}
		newCfgMap[v.StatueLevel][v.HeroClass] = v

		if newLevelMap[v.HeroClass] == nil {
			newLevelMap[v.HeroClass] = make(map[int32]*StatueAttrCfg)
		}
		for level := v.AttrLevel[0]; level <= v.AttrLevel[1]; level++ {
			newLevelMap[v.HeroClass][level] = v
		}
	}
	for key, v := range classLevelMapList {
		for i := v[0]; i <= v[1]; i++ {
			cfg := newLevelMap[key][i]
			for index, value := range cfg.Attr {
				if newInfoByLevel[key] == nil {
					newInfoByLevel[key] = make(map[int32]map[int32]int64)
				}
				if newInfoByLevel[key][value] == nil {
					newInfoByLevel[key][value] = make(map[int32]int64)
				}
				newInfoByLevel[key][value][i] += newInfoByLevel[key][value][i-1] + int64(cfg.AttrNum[index])
			}
		}
	}
	statueAttrInfoByLevel.Store(newInfoByLevel)
	statueAttrCfgMap.Store(newCfgMap)
	statueAttrLevelMap.Store(newLevelMap)
	return nil
}

func GetCityLevelCfg(ttype int32, id int32) CityConfig {
	var cfg CityConfig

	switch enum.ArchitectureType(ttype) {
	case enum.ARCHITECTURE_TYPE_MAIN:
		cfg = GetCityCenterCfg(id)
	case enum.ARCHITECTURE_TYPE_STONE:
		cfg = GetHeritageStatueCfg(id)
	case enum.ARCHITECTURE_TYPE_COLLECTION:
		cfg = GetCollectionCfg(id)
	case enum.ARCHITECTURE_TYPE_PET:
		cfg = GetPetSanctuaryCfg(id)
	case enum.ARCHITECTURE_TYPE_EQUIPMENT:
		cfg = GetEquipCraftCfg(id)
	case enum.ARCHITECTURE_TYPE_LUMBERYARD:
		cfg = GetLumberCfg(id)
	}

	return cfg
}

type LumberCfg struct {
	Id        int32         `json:"id"`
	Progress  int32         `json:"progress"`
	Unlock    []int32       `json:"unlock"`
	Item      []*ItemConfig `json:"item"`
	Time      int32         `json:"time"`
	Output    []*ItemConfig `json:"output"`
	Limit     []*ItemConfig `json:"limit"`
	HeroSlots int32         `json:"heroSlots"`
	Furniture int32         `json:"furniture"`
}

var _ CityConfig = (*LumberCfg)(nil)

func (p *LumberCfg) GetItem() []*ItemConfig { return p.Item }
func (p *LumberCfg) GetTime() int32         { return p.Time }
func (p *LumberCfg) GetUnlock() []int32     { return p.Unlock }

type FurnitureCfg struct {
	Id         int32         `json:"id"`
	Building   int32         `json:"building"`
	Type       int32         `json:"type"`
	Level      int32         `json:"level"`
	Item       []*ItemConfig `json:"item"`
	EffectType int32         `json:"effectType"`
	BaseEffect []*ItemConfig `json:"baseEffect"`
	Progress   int32         `json:"progress"`
}

func initFurnitureIndex() {
	cfgMap := furniture.Load()
	if cfgMap == nil {
		furnitureIndex.Store(make(map[int32]map[int32]map[int32]*FurnitureCfg))
		return
	}
	idx := make(map[int32]map[int32]map[int32]*FurnitureCfg)
	for _, v := range cfgMap.(map[int32]*FurnitureCfg) {
		if v.Level == 0 {
			continue
		}
		if idx[v.Building] == nil {
			idx[v.Building] = make(map[int32]map[int32]*FurnitureCfg)
		}
		if idx[v.Building][v.Type] == nil {
			idx[v.Building][v.Type] = make(map[int32]*FurnitureCfg)
		}
		idx[v.Building][v.Type][v.Level] = v
	}
	furnitureIndex.Store(idx)
}

func GetLumberCfg(level int32) *LumberCfg {
	v := lumber.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]*LumberCfg)[level]
}

func GetFurnitureCfgByBuildingTypeLevel(building int32, furnitureType int32, level int32) *FurnitureCfg {
	v := furnitureIndex.Load()
	if v == nil {
		return nil
	}
	idx := v.(map[int32]map[int32]map[int32]*FurnitureCfg)
	if idx[building] == nil || idx[building][furnitureType] == nil {
		return nil
	}
	return idx[building][furnitureType][level]
}

func GetFurnitureMaxLevel(building int32, furnitureType int32) int32 {
	v := furnitureIndex.Load()
	if v == nil {
		return 0
	}
	idx := v.(map[int32]map[int32]map[int32]*FurnitureCfg)
	if idx[building] == nil || idx[building][furnitureType] == nil {
		return 0
	}
	maxLv := int32(0)
	for lv := range idx[building][furnitureType] {
		if lv > maxLv {
			maxLv = lv
		}
	}
	return maxLv
}

func GetUnlockedFurnitureTypes(buildingType int32, buildingLevel int32) []int32 {
	if buildingLevel <= 0 {
		return nil
	}
	var cfgMap map[int32]*LumberCfg
	switch buildingType {
	case int32(enum.ARCHITECTURE_TYPE_LUMBERYARD):
		v := lumber.Load()
		if v == nil {
			return nil
		}
		cfgMap = v.(map[int32]*LumberCfg)
	default:
		return nil
	}
	result := make([]int32, 0)
	seen := make(map[int32]bool)
	for level := int32(1); level <= buildingLevel; level++ {
		cfg := cfgMap[level]
		if cfg == nil || cfg.Furniture <= 0 || seen[cfg.Furniture] {
			continue
		}
		seen[cfg.Furniture] = true
		result = append(result, cfg.Furniture)
	}
	return result
}

func IsFurnitureUnlocked(buildingType int32, buildingLevel int32, furnitureType int32) bool {
	for _, ft := range GetUnlockedFurnitureTypes(buildingType, buildingLevel) {
		if ft == furnitureType {
			return true
		}
	}
	return false
}
