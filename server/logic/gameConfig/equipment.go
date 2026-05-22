// File: equipment.go
// Description: 装备系统配置加载器
// Author: 木村凉太
// Create Time: 2025.11

package gameConfig

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("equipment", &EquipmentCfgLoader{})
}

type EquipmentCfgLoader struct {
	temp1 map[int32]*EquipmentBaseCfg      // 装备基础配置
	temp2 map[int32]*EquipmentAffixCfg     // 属性词条池配置
	temp3 map[int32]*EquipmentQualityCfg   // 品质关联配置
	temp4 map[int32]*EquipmentSetCfg       // 套装配置
	temp5 map[int64]*EquipmentLevelAttrCfg // 装备等级属性配置（equipment.json中的LevelAttr）
	temp6 map[int32]*EquipmentAttrCoeffCfg // 装备等级属性百分比提升

	temp7  map[int32]*EnhanceCostCfg        // 升级消耗配置
	temp8  map[int32]*EquipBlueprintCfg     // 装备蓝图配置
	temp9  map[int32]*EquipEnhanceCfg       // 装备升级配置
	temp10 map[int32]*EquipRefineCfg        // 装备精炼配置
	temp11 map[int32]*EquipmentLevelAttrCfg // 装备等级属性配置（equipment.json中的LevelAttr） 装备系统2.0使用
}

var _ configLoaderInterface = (*EquipmentCfgLoader)(nil)

func (s *EquipmentCfgLoader) loadData() error {
	// 加载 equipment.json
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/equipment.json`, &rawData); err != nil {
		return err
	}

	// 加载装备基础配置 (base)
	s.temp1 = make(map[int32]*EquipmentBaseCfg)
	if base, ok := rawData["base"]; ok {
		for _, row := range base {
			var v EquipmentBaseCfg
			v.ID = ParseInt(row["id"])
			v.EquipmentID = ParseInt(row["equipmentId"])
			v.EquipmentQuality = ParseInt(row["equipmentQuality"])
			v.EquipmentSlot = ParseInt(row["equipmentSlot"])
			v.Type = ParseInt(row["type"])
			v.SetID = ParseInt(row["set"])
			// 解析词条权重 "1|5000,2|5000" -> map[int32]int32
			v.AttrEntryWeight = ParseWeightMap(row["attrEntryWeight"])
			v.SkillID = ParseInt(row["skillId"])
			v.Tier = ParseInt(row["tier"])
			v.Star = ParseInt(row["star"])
			v.EquipmentType = ParseInt(row["equipmentType"])
			v.DropType = ParseInt(row["dropType"])

			if v.ID <= 0 {
				continue
			}
			if s.temp1[v.ID] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment base error duplicate ID:%d", v.ID))
			}
			s.temp1[v.ID] = &v
		}
	}

	// 加载属性词条池配置 (attrEntryPool)
	s.temp2 = make(map[int32]*EquipmentAffixCfg)
	if attrEntryPool, ok := rawData["attrEntryPool"]; ok {
		for _, row := range attrEntryPool {
			var v EquipmentAffixCfg
			v.ID = ParseInt(row["id"])
			v.AttrEntryID = ParseInt(row["attrEntryId"])
			v.AttrID = ParseInt(row["attrId"])
			v.EntryQuality = ParseInt(row["entryQuality"])
			v.Weight = ParseInt(row["weight"])
			// 解析属性范围 "0,10;20,50" -> [][]int32{{0,10},{20,50}}
			v.AttributeRanges = ParseAttributeRanges(row["attributeRange1"])
			if v.ID <= 0 {
				continue
			}
			if s.temp2[v.ID] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment attrEntryPool error duplicate ID:%d", v.ID))
			}
			s.temp2[v.ID] = &v
		}
	}

	// 加载品质关联配置 (qualityAssociation)
	s.temp3 = make(map[int32]*EquipmentQualityCfg)
	if qualityAssociation, ok := rawData["qualityAssociation"]; ok {
		for _, row := range qualityAssociation {
			var v EquipmentQualityCfg
			v.ID = ParseInt(row["id"])
			v.EquipmentQuality = ParseInt(row["equipmentQuality"])
			v.AttrEntryNum = ParseInt(row["attrEntryNum"])
			v.Item1 = ParseInt(row["item1"])
			v.Item2 = ParseInt(row["item2"])
			if v.ID <= 0 {
				continue
			}
			if s.temp3[v.EquipmentQuality] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment qualityAssociation error duplicate ID:%d", v.ID))
			}
			s.temp3[v.EquipmentQuality] = &v // 使用品质作为key
		}
	}

	// 加载套装配置 (setAttr)
	s.temp4 = make(map[int32]*EquipmentSetCfg)
	if setAttr, ok := rawData["setAttr"]; ok {
		for _, row := range setAttr {
			var v EquipmentSetCfg
			v.ID = ParseInt(row["id"])
			v.SetID = ParseInt(row["setId"])
			// 解析套装件数 "2,3,5" -> []int32{2,3,5}
			v.SetLevels = ParseIntArray(row["setLevel"])
			// 解析技能ID "0,0,0" -> []int32{0,0,0}
			v.SkillIDs = ParseIntArray(row["skillId"])
			if v.ID <= 0 {
				continue
			}
			if s.temp4[v.SetID] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment setAttr error duplicate ID:%d", v.ID))
			}
			s.temp4[v.SetID] = &v // 使用套装ID作为key
		}
	}

	// 加载装备等级属性配置 (LevelAttr from equipment.json)
	s.temp5 = make(map[int64]*EquipmentLevelAttrCfg)
	s.temp11 = make(map[int32]*EquipmentLevelAttrCfg)
	if levelAttr, ok := rawData["LevelAttr"]; ok {
		for _, row := range levelAttr {
			var v EquipmentLevelAttrCfg
			v.ID = ParseInt(row["id"])
			v.EquipmentID = ParseInt(row["equipmentId"])
			v.EquipmentLevel = ParseInt(row["equipmentLevel"])
			//v.SkillID = ParseInt(row["skillId"])
			// 解析属性 "1,200" 或 "1,200|2,200" -> []AttrValue{{AttrID:1,Value:200},...}
			v.Attributes = ParseAttrValues(row["attr1"], row["attr2"], row["attr3"],
				row["attr4"], row["attr5"], row["attr6"])
			if v.ID <= 0 {
				continue
			}
			if s.temp5[int64(v.EquipmentID)*100+int64(v.EquipmentLevel)] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment LevelAttr error duplicate ID:%d", v.ID))
			}
			// 使用 equipmentId_level 作为复合key
			key := int64(v.EquipmentID)*100 + int64(v.EquipmentLevel)
			s.temp5[key] = &v
			s.temp11[v.ID] = &v
		}
	}

	// 加载装备等级属性百分比提升配置 (AttrCoeff from equipment.json)
	s.temp6 = make(map[int32]*EquipmentAttrCoeffCfg)
	if attrCoeff, ok := rawData["LevelAttrCoeff"]; ok {
		for _, row := range attrCoeff {
			var v EquipmentAttrCoeffCfg
			v.ID = ParseInt(row["id"])
			v.EquipmentLevel = ParseInt(row["equipmentLevel"])
			// 解析属性 "1,200" 或 "1,200|2,200" -> []AttrValue{{AttrID:1,Value:200},...}
			tmpCoefficient := ParseAttrValues(row["Coefficient1"], row["Coefficient2"], row["Coefficient3"],
				row["Coefficient4"], row["Coefficient5"], row["Coefficient6"])

			v.Coefficient = make(map[int32]int32)
			for _, attr := range tmpCoefficient {
				v.Coefficient[attr.AttrID] = attr.Value
			}
			if v.ID <= 0 {
				continue
			}
			if s.temp6[v.EquipmentLevel] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load equipment AttrCoeff error duplicate ID:%d", v.ID))
			}
			s.temp6[v.EquipmentLevel] = &v
		}
	}

	s.temp7 = make(map[int32]*EnhanceCostCfg)
	for _, row := range rawData["enhanceCost"] {
		var v EnhanceCostCfg
		v.Id = ParseInt(row["id"])
		v.Level = ParseIntArray(row["level"])
		v.EnhanceCost = ParseItemArray(row["enhanceCost"])
		v.SuccessCost = ParseItemArray(row["successCost"])
		v.SuccessRate = ParseInt(row["successRate"])
		if v.Id <= 0 {
			continue
		}
		if s.temp7[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load enhanceCost error duplicate ID:%d", v.Id))
		}
		s.temp7[v.Id] = &v
	}

	s.temp8 = make(map[int32]*EquipBlueprintCfg)
	for _, row := range rawData["equipBlueprint"] {
		var v EquipBlueprintCfg
		v.Id = ParseInt(row["id"])
		v.Class = ParseInt(row["class"])
		v.Group = ParseInt(row["group"])
		v.Tier = ParseInt(row["tier"])
		v.Quality = ParseInt(row["quality"])
		v.Star = ParseInt(row["star"])
		v.Cost = ParseItemArray(row["cost"])
		v.Equipment = ParseInt(row["equipment"])
		if v.Id <= 0 {
			continue
		}
		if s.temp8[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load equipBlueprint error duplicate ID:%d", v.Id))
		}
		s.temp8[v.Id] = &v
	}

	s.temp9 = make(map[int32]*EquipEnhanceCfg)
	for _, row := range rawData["equipEnhance"] {
		var v EquipEnhanceCfg
		v.Id = ParseInt(row["id"])
		v.Tier = ParseInt(row["tier"])
		v.Quality = ParseInt(row["quality"])
		v.Star = ParseInt(row["star"])
		v.Type = ParseInt(row["type"])
		v.Slot = ParseInt(row["slot"])
		v.Level = ParseIntArray(row["level"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		if v.Id <= 0 {
			continue
		}
		if s.temp9[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load equipEnhance error duplicate ID:%d", v.Id))
		}
		s.temp9[v.Id] = &v
	}

	s.temp10 = make(map[int32]*EquipRefineCfg)
	for _, row := range rawData["equipRefine"] {
		var v EquipRefineCfg
		v.Id = ParseInt(row["id"])
		v.Tier = ParseInt(row["tier"])
		v.Quality = ParseInt(row["quality"])
		v.Star = ParseInt(row["star"])
		v.AttrEntryWeight = row["attrEntryWeight"]
		v.Cost = ParseItemArray(row["cost"])
		if v.Id <= 0 {
			continue
		}
		if s.temp10[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load equipRefine error duplicate ID:%d", v.Id))
		}
		s.temp10[v.Id] = &v
	}

	return nil
}

func (s *EquipmentCfgLoader) checkData() error {
	// 预计算装备重生返还道具（在constant配置加载完成后执行）
	initRebirthEquipItems()
	return nil
}

func (s *EquipmentCfgLoader) apply() {
	equipmentBase.Store(s.temp1)
	equipmentAffix.Store(s.temp2)
	equipmentQuality.Store(s.temp3)
	equipmentSet.Store(s.temp4)
	equipmentLevelAttr.Store(s.temp5)
	equipmentAttrCoeff.Store(s.temp6)
	enhanceCost.Store(s.temp7)
	equipBlueprint.Store(s.temp8)
	equipEnhance.Store(s.temp9)
	equipRefine.Store(s.temp10)
	equipLevelAttr.Store(s.temp11)
}

// 配置存储
var equipmentBase atomic.Value
var equipmentAffix atomic.Value
var equipmentQuality atomic.Value
var equipmentSet atomic.Value
var equipmentLevelAttr atomic.Value
var equipmentAttrCoeff atomic.Value
var enhanceCost atomic.Value
var equipBlueprint atomic.Value
var equipEnhance atomic.Value
var equipRefine atomic.Value
var equipLevelAttr atomic.Value

// 装备重生返还道具预计算 map[strongLevel][]*ItemConfig
var rebirthEquipItems atomic.Value

// EquipmentBaseCfg 装备基础配置
type EquipmentBaseCfg struct {
	ID               int32           `json:"id"`
	EquipmentID      int32           `json:"equipmentId"`      // 装备模板ID
	EquipmentQuality int32           `json:"equipmentQuality"` // 装备品质
	EquipmentSlot    int32           `json:"equipmentSlot"`    // 装备槽位
	Type             int32           `json:"type"`             // 装备类型
	SetID            int32           `json:"set"`              // 套装ID（0表示无套装）
	AttrEntryWeight  map[int32]int32 `json:"attrEntryWeight"`  // 词条ID -> 权重映射
	SkillID          int32           `json:"skillId"`          //技能词条
	Tier             int32           `json:"tier"`
	Star             int32           `json:"star"`
	EquipmentType    int32           `json:"equipmentType"`
	DropType         int32           `json:"dropType"`
}

// AttributeRangeRule 属性范围规则
// 根据等级范围，从对应的属性值范围中随机
type AttributeRangeRule struct {
	LevelRange [2]int32 `json:"levelRange"` // [minLevel, maxLevel] 等级范围
	ValueRange [2]int32 `json:"valueRange"` // [minValue, maxValue] 属性值范围
}

// EquipmentAffixCfg 属性词条池配置
type EquipmentAffixCfg struct {
	ID              int32                `json:"id"`
	AttrEntryID     int32                `json:"attrEntryId"`     // 词条ID
	AttrID          int32                `json:"attrId"`          // 属性ID
	EntryQuality    int32                `json:"entryQuality"`    // 词条品质
	Weight          int32                `json:"weight"`          // 权重
	AttributeRanges []AttributeRangeRule `json:"attributeRanges"` // 属性范围规则列表
}

// EquipmentQualityCfg 品质关联配置
type EquipmentQualityCfg struct {
	ID               int32 `json:"id"`
	EquipmentQuality int32 `json:"equipmentQuality"`
	AttrEntryNum     int32 `json:"attrEntryNum"` // 该品质的词条数量
	Item1            int32 `json:"item1"`
	Item2            int32 `json:"item2"`
}

// EquipmentSetCfg 套装配置
type EquipmentSetCfg struct {
	ID        int32   `json:"id"`
	SetID     int32   `json:"setId"`
	SetLevels []int32 `json:"setLevels"` // 套装件数列表，如 [2,3,5]
	SkillIDs  []int32 `json:"skillIds"`  // 对应件数的技能ID列表
}

// AttrValue 属性值
type AttrValue struct {
	AttrID int32
	Value  int32
}

// EquipmentLevelAttrCfg 装备等级属性配置
type EquipmentLevelAttrCfg struct {
	ID             int32       `json:"id"`
	EquipmentID    int32       `json:"equipmentId"`
	EquipmentLevel int32       `json:"equipmentLevel"`
	Attributes     []AttrValue `json:"attributes"` // 属性列表
}

type EquipmentAttrCoeffCfg struct {
	ID             int32           `json:"id"`
	EquipmentLevel int32           `json:"equipmentLevel"`
	Coefficient    map[int32]int32 `json:"coefficient"` // 属性列表
}

type EnhanceCostCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 等级区间
	Level []int32 `json:"level"`
	// 强化消耗
	EnhanceCost []*ItemConfig `json:"enhanceCost"`
	// 成功率消耗
	SuccessCost []*ItemConfig `json:"successCost"`
	// 成功率
	SuccessRate int32 `json:"successRate"`
}

type EquipBlueprintCfg struct {
	// 图纸id
	Id int32 `json:"id"`
	// 图纸职业
	Class int32 `json:"class"`
	// 图纸组
	Group int32 `json:"group"`
	// 图纸阶数
	Tier int32 `json:"tier"`
	// 图纸品质
	Quality int32 `json:"quality"`
	// 图纸星级
	Star int32 `json:"star"`
	// 打造消耗
	Cost []*ItemConfig `json:"cost"`
	// 对应装备
	Equipment int32 `json:"equipment"`
}

type EquipEnhanceCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 装备阶数
	Tier int32 `json:"tier"`
	// 装备品质
	Quality int32 `json:"quality"`
	// 装备星级
	Star int32 `json:"star"`
	// 装备类型
	Type int32 `json:"type"`
	// 装备部位
	Slot int32 `json:"slot"`
	// 强化等级
	Level []int32 `json:"level"`
	// 强化属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
}

type EquipRefineCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 装备阶数
	Tier int32 `json:"tier"`
	// 装备品质
	Quality int32 `json:"quality"`
	// 装备星级
	Star int32 `json:"star"`
	// 装备属性词条权重
	AttrEntryWeight string `json:"attrEntryWeight"`
	// 洗练石消耗
	Cost []*ItemConfig `json:"cost"`
}

// ParseWeightMap 解析权重映射 "1,5000;2,5000" -> map[int32]int32
func ParseWeightMap(s string) map[int32]int32 {
	result := make(map[int32]int32)
	if s == "" {
		return result
	}
	parts := strings.Split(s, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		weightPair := strings.Split(part, "~")
		if len(weightPair) != 2 {
			continue
		}
		id := ParseInt(weightPair[0])
		weight := ParseInt(weightPair[1])
		if id > 0 {
			result[id] = weight
		}
	}
	return result
}

// ParseAttributeRanges 解析属性范围规则
// 格式: "0,10;20,50|11,20;50,100"
// 说明: 用 | 分割多个规则，每个规则格式为 "等级范围;属性值范围"
//
//	等级范围格式: "minLevel,maxLevel"
//	属性值范围格式: "minValue,maxValue"
//
// 示例: 等级8落在0-10区间，从20-50中随机；等级15落在11-20区间，从50-100中随机
func ParseAttributeRanges(rangeStrs ...string) []AttributeRangeRule {
	var result []AttributeRangeRule
	for _, rangeStr := range rangeStrs {
		if rangeStr == "" {
			continue
		}
		// 先按 | 分割多个规则
		rules := strings.Split(rangeStr, ";")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}
			// 按 ; 分割等级范围和属性值范围
			parts := strings.Split(rule, "~")
			if len(parts) != 2 {
				continue
			}

			// 解析等级范围 "minLevel,maxLevel"
			levelRangeStr := strings.TrimSpace(parts[0])
			levelParts := strings.Split(levelRangeStr, "|")
			if len(levelParts) != 2 {
				continue
			}
			minLevel := ParseInt(levelParts[0])
			maxLevel := ParseInt(levelParts[1])

			// 解析属性值范围 "minValue,maxValue"
			valueRangeStr := strings.TrimSpace(parts[1])
			valueParts := strings.Split(valueRangeStr, "|")
			if len(valueParts) != 2 {
				continue
			}
			minValue := ParseInt(valueParts[0])
			maxValue := ParseInt(valueParts[1])

			result = append(result, AttributeRangeRule{
				LevelRange: [2]int32{minLevel, maxLevel},
				ValueRange: [2]int32{minValue, maxValue},
			})
		}
	}
	return result
}

// ParseAttrValues 解析属性值 "1,200" 或 "1,200|2,200" -> []AttrValue
func ParseAttrValues(attrStrs ...string) []AttrValue {
	var result []AttrValue
	for _, attrStr := range attrStrs {
		if attrStr == "" {
			continue
		}
		// 按逗号分割 attrID,value
		parts := strings.Split(attrStr, "~")
		if len(parts) == 2 {
			attrID := ParseInt(parts[0])
			value := ParseInt(parts[1])
			if attrID > 0 {
				result = append(result, AttrValue{
					AttrID: attrID,
					Value:  value,
				})
			}
		}
	}
	return result
}

// GetEquipmentBaseCfg 获取装备基础配置（通过ID）
func GetEquipmentBaseCfg(id int32) *EquipmentBaseCfg {
	cfgMap := equipmentBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentBaseCfg)[id]
}

// GetEquipmentBaseCfgByEquipmentID 通过装备ID获取基础配置
func GetEquipmentBaseCfgByEquipmentID(equipmentID int32) *EquipmentBaseCfg {
	cfgMap := equipmentBase.Load()
	if cfgMap == nil {
		return nil
	}
	for _, cfg := range cfgMap.(map[int32]*EquipmentBaseCfg) {
		if cfg.EquipmentID == equipmentID {
			return cfg
		}
	}
	return nil
}

// GetEquipmentAffixCfg 获取属性词条配置
func GetEquipmentAffixCfg(id int32) *EquipmentAffixCfg {
	cfgMap := equipmentAffix.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentAffixCfg)[id]
}

// GetEquipmentAffixCfgByAttrEntryID 通过词条ID获取配置
func GetEquipmentAffixCfgByAttrEntryID(attrEntryID int32) *EquipmentAffixCfg {
	cfgMap := equipmentAffix.Load()
	if cfgMap == nil {
		return nil
	}
	for _, cfg := range cfgMap.(map[int32]*EquipmentAffixCfg) {
		if cfg.AttrEntryID == attrEntryID {
			return cfg
		}
	}
	return nil
}

// 随机获得一个未获取的配置
func GetRandomEquipmentAffixCfgByQuality(quality int32, used map[int32]bool) *EquipmentAffixCfg {
	cfgMap := equipmentAffix.Load()
	if cfgMap == nil {
		return nil
	}
	var candidates []*EquipmentAffixCfg
	var weights []int32
	var totalWeight int32

	// 获取备选
	for _, cfg := range cfgMap.(map[int32]*EquipmentAffixCfg) {
		if cfg.EntryQuality == quality {
			if used[cfg.AttrEntryID] || cfg.Weight <= 0 {
				continue
			}
			candidates = append(candidates, cfg)
			weights = append(weights, cfg.Weight)
			totalWeight += cfg.Weight
		}
	}

	if totalWeight <= 0 {
		return nil
	}

	// 根据权重随机选择
	randValue := tool.RandInt32(1, totalWeight)
	var currentWeight int32
	for i, weight := range weights {
		currentWeight += weight
		if randValue <= currentWeight {
			return candidates[i]
		}
	}

	return nil
}

// GetEquipmentQualityCfg 获取品质关联配置
func GetEquipmentQualityCfg(quality int32) *EquipmentQualityCfg {
	cfgMap := equipmentQuality.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentQualityCfg)[quality]
}

// GetEquipmentSetCfg 获取套装配置
func GetEquipmentSetCfg(setID int32) *EquipmentSetCfg {
	cfgMap := equipmentSet.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentSetCfg)[setID]
}

// GetEquipmentLevelAttrCfg 获取装备等级属性配置
func GetEquipmentLevelAttrCfg(equipmentID int32, level int32) *EquipmentLevelAttrCfg {
	equipBaseCfg := GetEquipmentBaseCfg(equipmentID) // 获取装备基础配置
	if equipBaseCfg == nil {
		return nil
	}
	cfgMap := equipmentLevelAttr.Load()
	if cfgMap == nil {
		return nil
	}
	if equipBaseCfg.DropType == 1 {
		if level <= 20 {
			key := int64(equipmentID)*100 + int64(level)
			return cfgMap.(map[int64]*EquipmentLevelAttrCfg)[key]
		}

		// 20 级别基础属性配置
		baseCfg := cfgMap.(map[int64]*EquipmentLevelAttrCfg)[int64(equipmentID)*100+20]
		attrs := make([]AttrValue, len(baseCfg.Attributes))
		copy(attrs, baseCfg.Attributes)
		resCfg := EquipmentLevelAttrCfg{
			ID:             baseCfg.ID,
			EquipmentID:    baseCfg.EquipmentID,
			EquipmentLevel: baseCfg.EquipmentLevel,
			Attributes:     attrs,
		}

		coeffCfg := GetEquipmentAttrCoeffCfg(level)
		if coeffCfg == nil {
			return &resCfg
		}
		for index, value := range resCfg.Attributes {
			if coeff, ok := coeffCfg.Coefficient[value.AttrID]; ok {
				resCfg.Attributes[index].Value = int32(float64(value.Value) * float64(coeff) / 10000)
			}
		}

		return &resCfg
	} else {
		key := int64(equipmentID)*100 + int64(level)
		return cfgMap.(map[int64]*EquipmentLevelAttrCfg)[key]
	}
}

func GetEquipLevelAttrCfg(id int32) *EquipmentLevelAttrCfg {
	cfgMap := equipLevelAttr.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentLevelAttrCfg)[id]
}

// GetAllEquipmentBaseCfg 获取所有装备基础配置
func GetAllEquipmentBaseCfg() map[int32]*EquipmentBaseCfg {
	cfgMap := equipmentBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentBaseCfg)
}

// GetAllEquipmentAffixCfg 获取所有属性词条配置
func GetAllEquipmentAffixCfg() map[int32]*EquipmentAffixCfg {
	cfgMap := equipmentAffix.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentAffixCfg)
}

func GetEquipmentAttrCoeffCfg(equipmentLevel int32) *EquipmentAttrCoeffCfg {
	cfgMap := equipmentAttrCoeff.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipmentAttrCoeffCfg)[equipmentLevel]
}

func GetEnhanceCostCfg(id int32) *EnhanceCostCfg {
	cfgMap := enhanceCost.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EnhanceCostCfg)[id]
}

func GetEquipBlueprintCfg(id int32) *EquipBlueprintCfg {
	cfgMap := equipBlueprint.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipBlueprintCfg)[id]
}

func GetEquipEnhanceCfg(id int32) *EquipEnhanceCfg {
	cfgMap := equipEnhance.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipEnhanceCfg)[id]
}

func GetEquipRefineCfg(id int32) *EquipRefineCfg {
	cfgMap := equipRefine.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*EquipRefineCfg)[id]
}

// initRebirthEquipItems 预计算所有强化等级的返还道具
func initRebirthEquipItems() {
	constCfg := GetConstantCfg(CONSTANT_refundRateForEnhanceMaterials)
	if constCfg == nil || len(constCfg.Value) == 0 {
		rebirthEquipItems.Store(make(map[int32]map[int32]int64))
		return
	}
	constRebirthItemsCount := constCfg.Value[0]

	addItems := make(map[int32]map[int32]int64) //level->itemID->num
	for i := int32(1); ; i++ {
		costCfg := GetEnhanceCostCfg(i)
		if costCfg == nil {
			rebirthEquipItems.Store(addItems)
			return
		}
		for j := costCfg.Level[0]; j <= costCfg.Level[1]; j++ {
			for _, itemCfg := range costCfg.EnhanceCost {
				if addItems[j] == nil {
					addItems[j] = make(map[int32]int64)
				}
				addItems[j][itemCfg.ID] += addItems[j-1][itemCfg.ID] + itemCfg.Num*int64(constRebirthItemsCount)/10000
			}
		}
	}
}

// GetRebirthEquipItems 根据强化等级获取返还道具（O(1)）
func GetRebirthEquipItems(strongLevel int32) map[int32]int64 {
	m := rebirthEquipItems.Load()
	if m == nil {
		return nil
	}
	return m.(map[int32]map[int32]int64)[strongLevel] // Changed from return m.(map[int32][]*ItemConfig)[strongLevel]
}
