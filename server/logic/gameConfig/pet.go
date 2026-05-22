package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("pet", &PetCfgLoader{})
}

type PetCfgLoader struct {
	temp1 map[int32]*PetAffinityCfg
	temp2 map[int32]*PetBaseCfg
	temp3 map[int32]*PetLevelCfg
	temp4 map[int32]*PetPassiveSkillCfg
	temp5 map[int32]*PetStarCfg
	temp6 map[int32]*PetSummonCfg

	// 运行时索引：为 O(1) 查询准备
	levelIndex map[int32]map[int32]*PetLevelCfg   // potential -> level -> cfg
	starIndex  map[int32]map[int32]*PetStarCfg    // petId -> star -> cfg（约定 PetStarCfg.Id = petId）
	psGroupIdx map[int32]*PetPassiveSkillGroupCfg // passiveSkillGroup -> cfg
}

var _ configLoaderInterface = (*PetCfgLoader)(nil)

func (s *PetCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/pet.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*PetAffinityCfg)
	for _, row := range rawData["petAffinity"] {
		var v PetAffinityCfg
		v.Id = ParseInt(row["id"])
		v.PetId = ParseIntArray(row["petId"])
		v.PetStar = ParseIntArray(row["petStar"])
		v.Attr = ParseInt(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*PetBaseCfg)
	for _, row := range rawData["petBase"] {
		var v PetBaseCfg
		v.Id = ParseInt(row["id"])
		v.PetRarity = ParseInt(row["petRarity"])
		v.PetPotential = ParseInt(row["petPotential"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.UniqueSkill = ParseInt(row["uniqueSkill"])
		v.Class = ParseInt(row["class"])
		v.PassiveSkillGroup = ParseIntArray(row["passiveSkillGroup"])
		v.SkillGroupWeight = ParseIntArray(row["skillGroupWeight"])
		v.SalvageYield = ParseItemArray(row["salvageYield"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petBase error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*PetLevelCfg)
	s.levelIndex = make(map[int32]map[int32]*PetLevelCfg)
	for _, row := range rawData["petLevel"] {
		var v PetLevelCfg
		v.Id = ParseInt(row["id"])
		v.PetPotential = ParseInt(row["petPotential"])
		v.PetLevel = ParseInt(row["petLevel"])
		v.UnlockId = ParseInt(row["unlockId"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.Cost = ParseItemArray(row["cost"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v

		if v.PetPotential > 0 && v.PetLevel > 0 {
			if s.levelIndex[v.PetPotential] == nil {
				s.levelIndex[v.PetPotential] = make(map[int32]*PetLevelCfg)
			}
			s.levelIndex[v.PetPotential][v.PetLevel] = &v
		}
	}

	s.temp4 = make(map[int32]*PetPassiveSkillCfg)
	s.psGroupIdx = make(map[int32]*PetPassiveSkillGroupCfg)
	for _, row := range rawData["petPassiveSkill"] {
		var v PetPassiveSkillCfg
		v.Id = ParseInt(row["id"])
		v.PassiveSkillGroup = ParseInt(row["passiveSkillGroup"])
		v.Skill = ParseInt(row["skill"])
		v.SkillWeight = ParseInt(row["skillWeight"])
		if v.Id <= 0 {
			continue
		}
		if s.temp4[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error duplicate ID:%d", v.Id))
		}
		s.temp4[v.Id] = &v

		// 构建按组聚合的运行时索引：group -> (skills, weights)
		if v.PassiveSkillGroup > 0 && v.Skill > 0 && v.SkillWeight > 0 {
			if s.psGroupIdx[v.PassiveSkillGroup] == nil {
				s.psGroupIdx[v.PassiveSkillGroup] = &PetPassiveSkillGroupCfg{
					PassiveSkillGroup: v.PassiveSkillGroup,
					Skill:             make([]int32, 0, 4),
					SkillWeight:       make([]int32, 0, 4),
				}
			}
			s.psGroupIdx[v.PassiveSkillGroup].Skill = append(s.psGroupIdx[v.PassiveSkillGroup].Skill, v.Skill)
			s.psGroupIdx[v.PassiveSkillGroup].SkillWeight = append(s.psGroupIdx[v.PassiveSkillGroup].SkillWeight, v.SkillWeight)
		}
	}

	s.temp5 = make(map[int32]*PetStarCfg)
	s.starIndex = make(map[int32]map[int32]*PetStarCfg)
	for _, row := range rawData["petStar"] {
		var v PetStarCfg
		v.Id = ParseInt(row["id"])
		v.PetId = ParseInt(row["petId"])
		v.PetStar = ParseInt(row["petStar"])
		v.Attr = ParseIntArray(row["attr"])
		v.AttrNum = ParseIntArray(row["attrNum"])
		v.CostNum1 = ParseInt(row["costNum1"])
		v.CostNum2 = ParseItem(row["costNum2"])
		v.ActiveSkill = ParseInt(row["activeSkill"])
		v.PassiveSkill = ParseInt(row["passiveSkill"])
		if v.Id <= 0 {
			continue
		}
		if s.temp5[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error duplicate ID:%d", v.Id))
		}
		s.temp5[v.Id] = &v

		// 运行时索引：按 petId + star 分层
		if v.PetId > 0 && v.PetStar >= 0 {
			if s.starIndex[v.PetId] == nil {
				s.starIndex[v.PetId] = make(map[int32]*PetStarCfg)
			}
			s.starIndex[v.PetId][v.PetStar] = &v
		}
	}

	s.temp6 = make(map[int32]*PetSummonCfg)
	for _, row := range rawData["petSummon"] {
		var v PetSummonCfg
		v.Id = ParseInt(row["id"])
		v.Weight = ParseInt(row["weight"])
		v.Value = ParseInt(row["value"])
		if v.Id <= 0 {
			continue
		}
		if s.temp6[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petSummon error duplicate ID:%d", v.Id))
		}
		s.temp6[v.Id] = &v
	}

	return nil
}

func (s *PetCfgLoader) checkData() error {
	// 1) 通用引用校验（道具/解锁/宠物ID存在）
	// 2) 数组形状/长度一致性
	// 3) petLevel 连续性（按 potential）
	// 4) petStar 连续性（按 petId）

	// petBase 基础校验 + 引用（道具）+ 被动池长度一致
	for id, v := range s.temp2 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petBase error invalid ID:%d", id))
		}
		if len(v.Attr) != len(v.AttrNum) {
			return errors.New(fmt.Sprintf("[gameConfig] load petBase error attr size mismatch petId:%d attrCount:%d attrNumCount:%d", v.Id, len(v.Attr), len(v.AttrNum)))
		}
		for i, attrID := range v.Attr {
			if attrID <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error invalid attrId petId:%d idx:%d", v.Id, i))
			}
		}
		// 被动技能池：group 与权重必须一一对应
		if len(v.PassiveSkillGroup) != len(v.SkillGroupWeight) {
			return errors.New(fmt.Sprintf("[gameConfig] load petBase error passiveSkillGroup size mismatch petId:%d groupCount:%d weightCount:%d", v.Id, len(v.PassiveSkillGroup), len(v.SkillGroupWeight)))
		}
		for i, g := range v.PassiveSkillGroup {
			if g <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error invalid passiveSkillGroup petId:%d idx:%d", v.Id, i))
			}
			if i < len(v.SkillGroupWeight) && v.SkillGroupWeight[i] <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error invalid skillGroupWeight petId:%d group:%d weight:%d", v.Id, g, v.SkillGroupWeight[i]))
			}
			// group 必须存在于聚合索引中（至少有 1 条 skill/weight）
			if s.psGroupIdx == nil || s.psGroupIdx[g] == nil || len(s.psGroupIdx[g].Skill) == 0 || len(s.psGroupIdx[g].Skill) != len(s.psGroupIdx[g].SkillWeight) {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error passiveSkillGroup not found or invalid group:%d petId:%d", g, v.Id))
			}
		}
		// 分解产出引用道具
		for _, it := range v.SalvageYield {
			if it == nil || it.ID <= 0 || it.Num <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error invalid salvageYield petId:%d", v.Id))
			}
			if GetItemCfg(it.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petBase error salvageYield item not found petId:%d itemId:%d", v.Id, it.ID))
			}
		}
	}

	// petAffinity：petId 存在、星级档位与数值对齐
	for id, v := range s.temp1 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error invalid ID:%d", id))
		}
		if len(v.PetId) == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error empty petId affinityId:%d", v.Id))
		}
		for _, pid := range v.PetId {
			if pid <= 0 || s.temp2[pid] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error petBase not found affinityId:%d petId:%d", v.Id, pid))
			}
		}
		if v.Attr <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error invalid attr affinityId:%d attr:%d", v.Id, v.Attr))
		}
		if len(v.PetStar) == 0 || len(v.AttrNum) == 0 || len(v.PetStar) != len(v.AttrNum) {
			return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error star/attrNum size mismatch affinityId:%d starCount:%d attrNumCount:%d", v.Id, len(v.PetStar), len(v.AttrNum)))
		}
		prev := int32(-1)
		for i, star := range v.PetStar {
			if star < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error invalid petStar affinityId:%d idx:%d star:%d", v.Id, i, star))
			}
			if i == 0 && star != 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error first petStar must be 0 affinityId:%d firstStar:%d", v.Id, star))
			}
			if star <= prev {
				return errors.New(fmt.Sprintf("[gameConfig] load petAffinity error petStar not increasing affinityId:%d idx:%d star:%d prev:%d", v.Id, i, star, prev))
			}
			prev = star
		}
	}

	// petLevel：基础引用（potential/level）、attr 形状、cost 道具引用、unlock 引用
	for id, v := range s.temp3 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error invalid ID:%d", id))
		}
		if v.PetPotential <= 0 || v.PetLevel <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error invalid potential/level id:%d potential:%d level:%d", v.Id, v.PetPotential, v.PetLevel))
		}
		if len(v.Attr) != len(v.AttrNum) {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error attr size mismatch id:%d potential:%d level:%d", v.Id, v.PetPotential, v.PetLevel))
		}
		for i, attrID := range v.Attr {
			if attrID <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petLevel error invalid attrId id:%d idx:%d", v.Id, i))
			}
		}
		for _, it := range v.Cost {
			if it == nil || it.ID <= 0 || it.Num <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petLevel error invalid cost id:%d potential:%d level:%d", v.Id, v.PetPotential, v.PetLevel))
			}
			if GetItemCfg(it.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petLevel error cost item not found id:%d itemId:%d", v.Id, it.ID))
			}
		}
		if v.UnlockId != 0 && GetUnlockCfg(v.UnlockId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error unlock not found id:%d unlockId:%d", v.Id, v.UnlockId))
		}
	}

	// petPassiveSkill：基础值域 + 按组聚合后形状
	for id, v := range s.temp4 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error invalid ID:%d", id))
		}
		if v.PassiveSkillGroup <= 0 || v.Skill <= 0 || v.SkillWeight <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error invalid values id:%d group:%d skill:%d weight:%d", v.Id, v.PassiveSkillGroup, v.Skill, v.SkillWeight))
		}
	}
	for group, gcfg := range s.psGroupIdx {
		if group <= 0 || gcfg == nil || gcfg.PassiveSkillGroup != group {
			return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error invalid group index group:%d", group))
		}
		if len(gcfg.Skill) == 0 || len(gcfg.Skill) != len(gcfg.SkillWeight) {
			return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error group size mismatch group:%d skillCount:%d weightCount:%d", group, len(gcfg.Skill), len(gcfg.SkillWeight)))
		}
		for i := range gcfg.Skill {
			if gcfg.Skill[i] <= 0 || gcfg.SkillWeight[i] <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petPassiveSkill error invalid group entry group:%d idx:%d skill:%d weight:%d", group, i, gcfg.Skill[i], gcfg.SkillWeight[i]))
			}
		}
	}

	// petStar：petId 存在、星级>=0、attr 形状
	for id, v := range s.temp5 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error invalid ID:%d", id))
		}
		if v.PetId <= 0 || s.temp2[v.PetId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error petBase not found id:%d petId:%d", v.Id, v.PetId))
		}
		if v.PetStar < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error invalid petStar id:%d petId:%d star:%d", v.Id, v.PetId, v.PetStar))
		}
		if len(v.Attr) != len(v.AttrNum) {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error attr size mismatch id:%d petId:%d star:%d", v.Id, v.PetId, v.PetStar))
		}
		for i, attrID := range v.Attr {
			if attrID <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petStar error invalid attrId id:%d idx:%d", v.Id, i))
			}
		}
		if v.CostNum1 < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error invalid costNum1 id:%d petId:%d star:%d", v.Id, v.PetId, v.PetStar))
		}
		if v.CostNum2 != nil {
			if v.CostNum2.ID <= 0 || v.CostNum2.Num <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load petStar error invalid costNum2 id:%d petId:%d star:%d", v.Id, v.PetId, v.PetStar))
			}
			if GetItemCfg(v.CostNum2.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petStar error costNum2 item not found id:%d itemId:%d", v.Id, v.CostNum2.ID))
			}
		}
	}

	// petSummon：weight/value 值域
	for id, v := range s.temp6 {
		if v == nil || v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petSummon error invalid ID:%d", id))
		}
		if v.Weight <= 0 || v.Value < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load petSummon error invalid values id:%d weight:%d value:%d", v.Id, v.Weight, v.Value))
		}
	}

	// 3) petLevel 连续性：每个 potential 从 1 开始连续到最大档
	for potential, m1 := range s.levelIndex {
		if potential <= 0 || m1 == nil {
			continue
		}
		if m1[1] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petLevel error missing level=1 potential:%d", potential))
		}
		maxLvl := int32(1)
		for lvl := range m1 {
			if lvl > maxLvl {
				maxLvl = lvl
			}
		}
		for lvl := int32(1); lvl <= maxLvl; lvl++ {
			if m1[lvl] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petLevel error level gap potential:%d missingLevel:%d maxLevel:%d", potential, lvl, maxLvl))
			}
		}
	}

	// 4) petStar 连续性：每个 petId 从 0 开始连续到最大星级
	for petId, m1 := range s.starIndex {
		if petId <= 0 || m1 == nil {
			continue
		}
		if m1[0] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load petStar error missing star=0 petId:%d", petId))
		}
		maxStar := int32(0)
		for star := range m1 {
			if star > maxStar {
				maxStar = star
			}
		}
		for star := int32(0); star <= maxStar; star++ {
			if m1[star] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load petStar error star gap petId:%d missingStar:%d maxStar:%d", petId, star, maxStar))
			}
		}
	}

	return nil
}

func (s *PetCfgLoader) apply() {
	petAffinity.Store(s.temp1)
	petBase.Store(s.temp2)
	petLevel.Store(s.temp3)
	petPassiveSkill.Store(s.temp4)
	petStar.Store(s.temp5)
	petSummon.Store(s.temp6)
	petLevelIndex.Store(s.levelIndex)
	petStarIndex.Store(s.starIndex)
	petPassiveSkillGroupIndex.Store(s.psGroupIdx)
}

var petAffinity atomic.Value
var petBase atomic.Value
var petLevel atomic.Value
var petPassiveSkill atomic.Value
var petStar atomic.Value
var petSummon atomic.Value
var petLevelIndex atomic.Value
var petStarIndex atomic.Value
var petPassiveSkillGroupIndex atomic.Value

type PetAffinityCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 宠物id
	PetId []int32 `json:"petId"`
	// 宠物星级
	PetStar []int32 `json:"petStar"`
	// 缘分属性
	Attr int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
}

type PetBaseCfg struct {
	// 宠物id
	Id int32 `json:"id"`
	// 宠物品质
	PetRarity int32 `json:"petRarity"`
	// 宠物潜力
	PetPotential int32 `json:"petPotential"`
	// 初始属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 专属技能
	UniqueSkill int32 `json:"uniqueSkill"`
	// 专属职业
	Class int32 `json:"class"`
	// 被动技能池
	PassiveSkillGroup []int32 `json:"passiveSkillGroup"`
	// 被动技能池权重
	SkillGroupWeight []int32 `json:"skillGroupWeight"`
	// 分解产出
	SalvageYield []*ItemConfig `json:"salvageYield"`
}

type PetLevelCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 宠物潜力
	PetPotential int32 `json:"petPotential"`
	// 等级
	PetLevel int32 `json:"petLevel"`
	// 升级条件
	UnlockId int32 `json:"unlockId"`
	// 属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 升级消耗
	Cost []*ItemConfig `json:"cost"`
}

type PetPassiveSkillCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 被动技能组
	PassiveSkillGroup int32 `json:"passiveSkillGroup"`
	// 技能id
	Skill int32 `json:"skill"`
	// 权重
	SkillWeight int32 `json:"skillWeight"`
}

// PetPassiveSkillGroupCfg 运行时聚合：按被动技能组归并多行配置，供“按组随机”使用。
type PetPassiveSkillGroupCfg struct {
	PassiveSkillGroup int32   `json:"passiveSkillGroup"`
	Skill             []int32 `json:"skill"`
	SkillWeight       []int32 `json:"skillWeight"`
}

type PetStarCfg struct {
	// 序号id
	Id int32 `json:"id"`
	// 宠物id
	PetId int32 `json:"petId"`
	// 宠物星级
	PetStar int32 `json:"petStar"`
	// 属性
	Attr []int32 `json:"attr"`
	// 属性数值
	AttrNum []int32 `json:"attrNum"`
	// 消耗本体数量（同宠物材料数量）
	CostNum1 int32 `json:"costNum1"`
	// 消耗材料（额外道具）
	CostNum2 *ItemConfig `json:"costNum2"`
	// 主动技能
	ActiveSkill int32 `json:"activeSkill"`
	// 被动技能激活
	PassiveSkill int32 `json:"passiveSkill"`
}

type PetSummonCfg struct {
	// 宠物潜力
	Id int32 `json:"id"`
	// 权重
	Weight int32 `json:"weight"`
	// 钻石招募价格
	Value int32 `json:"value"`
}

func GetPetAffinityCfg(id int32) *PetAffinityCfg {
	cfgMap := petAffinity.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetAffinityCfg)[id]
}

func GetPetBaseCfg(id int32) *PetBaseCfg {
	cfgMap := petBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetBaseCfg)[id]
}

func GetPetLevelCfg(id int32) *PetLevelCfg {
	cfgMap := petLevel.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetLevelCfg)[id]
}

func GetPetPassiveSkillCfg(id int32) *PetPassiveSkillCfg {
	cfgMap := petPassiveSkill.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetPassiveSkillCfg)[id]
}

// GetPetPassiveSkillGroupCfg 获取被动技能组聚合配置（group -> skills/weights）。
func GetPetPassiveSkillGroupCfg(group int32) *PetPassiveSkillGroupCfg {
	cfgMap := petPassiveSkillGroupIndex.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetPassiveSkillGroupCfg)[group]
}

func GetPetStarCfg(id int32) *PetStarCfg {
	cfgMap := petStar.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetStarCfg)[id]
}

func GetPetSummonCfg(id int32) *PetSummonCfg {
	cfgMap := petSummon.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetSummonCfg)[id]
}

// ====== 扩展查询：为宠物主体逻辑提供基础能力（不含奖池） ======

func GetAllPetAffinityCfg() map[int32]*PetAffinityCfg {
	cfgMap := petAffinity.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetAffinityCfg)
}

func GetAllPetBaseCfg() map[int32]*PetBaseCfg {
	cfgMap := petBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetBaseCfg)
}

func GetAllPetLevelCfg() map[int32]*PetLevelCfg {
	cfgMap := petLevel.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetLevelCfg)
}

func GetAllPetStarCfg() map[int32]*PetStarCfg {
	cfgMap := petStar.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PetStarCfg)
}

// GetPetLevelCfgByPotentialLevel 按“潜力 + 等级”查询等级配置。
// 目前 petLevel 仅按 id 存储，这里用遍历方式提供基础能力；后续可优化为索引 map。
func GetPetLevelCfgByPotentialLevel(potential int32, level int32) *PetLevelCfg {
	if potential <= 0 || level <= 0 {
		return nil
	}
	idx := petLevelIndex.Load()
	if idx != nil {
		m1 := idx.(map[int32]map[int32]*PetLevelCfg)[potential]
		if m1 != nil {
			return m1[level]
		}
		return nil
	}

	// 兼容：索引未构建时退化为遍历
	all := GetAllPetLevelCfg()
	for _, cfg := range all {
		if cfg == nil {
			continue
		}
		if cfg.PetPotential == potential && cfg.PetLevel == level {
			return cfg
		}
	}
	return nil
}

func HasPetLevelUpgradeStep(cfg *PetLevelCfg) bool {
	if cfg == nil {
		return false
	}
	if cfg.UnlockId > 0 {
		return true
	}
	for _, it := range cfg.Cost {
		if it != nil && it.ID > 0 && it.Num > 0 {
			return true
		}
	}
	for i := 0; i < len(cfg.Attr) && i < len(cfg.AttrNum); i++ {
		if cfg.Attr[i] > 0 && cfg.AttrNum[i] != 0 {
			return true
		}
	}
	return false
}

func GetPetMaxLevelByPotential(potential int32) int32 {
	if potential <= 0 {
		return 1
	}
	maxCfgLevel := int32(0)
	for lvl := int32(1); lvl < 100000; lvl++ {
		cfg := GetPetLevelCfgByPotentialLevel(potential, lvl)
		if cfg == nil {
			break
		}
		maxCfgLevel = lvl
	}
	if maxCfgLevel <= 0 {
		return 1
	}
	if HasPetLevelUpgradeStep(GetPetLevelCfgByPotentialLevel(potential, maxCfgLevel)) {
		return maxCfgLevel + 1
	}
	return maxCfgLevel
}

// GetPetStarCfgByPetIdStar 按“宠物ID + 星级”查询星级配置。
func GetPetStarCfgByPetIdStar(petId int32, star int32) *PetStarCfg {
	if petId <= 0 || star < 0 {
		return nil
	}
	idx := petStarIndex.Load()
	if idx != nil {
		m1 := idx.(map[int32]map[int32]*PetStarCfg)[petId]
		if m1 != nil {
			return m1[star]
		}
		return nil
	}

	// 兼容：索引未构建时退化为遍历
	all := GetAllPetStarCfg()
	for _, cfg := range all {
		if cfg == nil {
			continue
		}
		if cfg.PetId == petId && cfg.PetStar == star {
			return cfg
		}
	}
	return nil
}

func HasPetStarUpgradeStep(cfg *PetStarCfg) bool {
	if cfg == nil {
		return false
	}
	if cfg.CostNum1 > 0 {
		return true
	}
	for i := 0; i < len(cfg.Attr) && i < len(cfg.AttrNum); i++ {
		if cfg.Attr[i] > 0 && cfg.AttrNum[i] != 0 {
			return true
		}
	}
	return false
}

func GetPetMaxStarByPetId(petId int32) int32 {
	if petId <= 0 {
		return 0
	}
	maxCfgStar := int32(-1)
	for star := int32(0); star < 100000; star++ {
		cfg := GetPetStarCfgByPetIdStar(petId, star)
		if cfg == nil {
			break
		}
		maxCfgStar = star
	}
	if maxCfgStar < 0 {
		return 0
	}
	return maxCfgStar
}

func getAttrValue(attrIds []int32, attrNums []int32, attrId int32) int64 {
	for i, id := range attrIds {
		if id != attrId {
			continue
		}
		if i >= 0 && i < len(attrNums) {
			return int64(attrNums[i])
		}
		return 0
	}
	return 0
}

// CalcPetAttr 计算单只宠物对指定属性的加成（基础+等级+星级）。
// 不含缘分（缘分为全局加成，应另行处理）。
func CalcPetAttr(petId int32, level int32, star int32, attrId int32) int64 {
	if petId <= 0 || attrId <= 0 {
		return 0
	}
	base := GetPetBaseCfg(petId)
	if base == nil {
		return 0
	}
	total := int64(0)
	// base
	total += getAttrValue(base.Attr, base.AttrNum, attrId)
	// level: 按潜力曲线叠加 1..(level-1)。
	// 语义：petLevel=n 表示 n->n+1 的属性增量。
	if level < 1 {
		level = 1
	}
	for fromLevel := int32(1); fromLevel < level; fromLevel++ {
		lcfg := GetPetLevelCfgByPotentialLevel(base.PetPotential, fromLevel)
		if lcfg == nil {
			continue
		}
		total += getAttrValue(lcfg.Attr, lcfg.AttrNum, attrId)
	}
	// star: 叠加 0..(star-1)（0 星视为未激活，不额外加成）
	if star < 1 {
		return total
	}
	for fromStar := int32(0); fromStar < star; fromStar++ {
		scfg := GetPetStarCfgByPetIdStar(petId, fromStar)
		if scfg == nil {
			continue
		}
		total += getAttrValue(scfg.Attr, scfg.AttrNum, attrId)
	}
	return total
}
