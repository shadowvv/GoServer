package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

var HeroEvolutionTimeCfg int64 = 30 * 60 * 1000

func init() {
	RegisterConfigLoader("hero", &HeroCfgLoader{})
}

var heroAttrByLevelCfg atomic.Value // map[int32]map[int32]map[int32]map[int32]int64 潜力->职业->等级->属性ID->属性值
var heroAttrByStarCfg atomic.Value  // map[int32]map[int32]map[int32]map[int32]int64  潜力->职业->星级->属性ID->属性值
var heroAttrByBreakCfg atomic.Value // map[int32]map[int32]map[int32]map[int32]int64 潜力->职业->阶数->属性ID->属性值

type HeroCfgLoader struct {
	temp1  map[int32]*CodexRewardCfg
	temp2  map[int32]*ComboSkillCfg
	temp3  map[int32]*HeroBaseCfg
	temp4  map[int32]map[int32]map[int32]*HeroBreakCfg
	temp5  map[int32]*HeroClassCfg
	temp6  map[int32]*HeroCodexCfg
	temp7  map[int32]map[int32]map[int32]*HeroLevelCfg
	temp8  map[int32]map[int32]map[int32]*HeroStarCfg
	temp9  map[int32]*LevelRatioCfg
	temp10 map[int32]map[int32]*StarEffectCfg
	temp11 map[int32]*StarCfg
}

var _ configLoaderInterface = (*HeroCfgLoader)(nil)

func (s *HeroCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/hero.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*CodexRewardCfg)
	for _, row := range rawData["codexReward"] {
		var v CodexRewardCfg
		v.Id = ParseInt(row["id"])
		v.CodexPoints = ParseInt(row["codexPoints"])
		v.Reward = ParseItemArray(row["reward"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load hero codexReward duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*ComboSkillCfg)
	for _, row := range rawData["comboSkill"] {
		var v ComboSkillCfg
		v.Id = ParseInt(row["id"])
		v.Level = ParseInt(row["level"])
		v.LimitedHero = ParseIntArray(row["limitedHero"])
		v.StarLimit = ParseIntArray(row["starLimit"])
		v.LeaderId = ParseInt(row["leaderId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load hero comboSkill duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*HeroBaseCfg)
	for _, row := range rawData["heroBase"] {
		var v HeroBaseCfg
		v.HeroId = ParseInt(row["id"])
		v.UnitsId = ParseInt(row["unitsId"])
		v.HeroPotential = ParseInt(row["heroPotential"])
		v.HeroStar = ParseInt(row["heroStar"])
		v.HeroClass = ParseInt(row["heroClass"])
		v.ComboSkill = ParseIntArray(row["comboSkill"])
		v.HeroSkill = ParseIntArray(row["heroSkill"])
		v.Stat = ParseIntArray(row["stat"])
		v.StatValue = ParseIntArray(row["statValue"])
		v.AttackRange = ParseInt(row["attackRange"])
		v.Cap = ParseIntArray(row["cap"])
		v.PatrolRange = ParseInt(row["patrolRange"])
		v.AggroRange = ParseInt(row["aggroRange"])
		if v.HeroId <= 0 {
			continue
		}
		if s.temp3[v.HeroId] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load hero heroBase duplicate ID:%d", v.HeroId))
		}
		s.temp3[v.HeroId] = &v
	}

	s.temp4 = make(map[int32]map[int32]map[int32]*HeroBreakCfg)
	for _, row := range rawData["heroBreak"] {
		var v HeroBreakCfg
		v.Id = ParseInt(row["id"])
		v.HeroPotential = ParseInt(row["heroPotential"])
		v.HeroClass = ParseInt(row["heroClass"])
		v.Rank = ParseInt(row["rank"])
		v.HeroLevel = ParseInt(row["heroLevel"])
		v.BreakStat = ParseIntArray(row["breakStat"])
		v.StatValue = ParseIntArray(row["statValue"])
		v.BreakMaterials = ParseIntArray(row["breakMaterials"])
		v.BreakCost = ParseIntArray(row["breakCost"])
		if v.Id <= 0 {
			continue
		}
		m1 := s.temp4[v.HeroPotential]
		if m1 == nil {
			m1 = make(map[int32]map[int32]*HeroBreakCfg)
			s.temp4[v.HeroPotential] = m1
		}
		m2 := m1[v.HeroClass]
		if m2 == nil {
			m2 = make(map[int32]*HeroBreakCfg)
			m1[v.HeroClass] = m2
		}
		m2[v.Rank] = &v
	}

	s.temp5 = make(map[int32]*HeroClassCfg)
	for _, row := range rawData["heroClass"] {
		var v HeroClassCfg
		v.Id = ParseInt(row["id"])
		v.ChangeClass = ParseIntArray(row["changeClass"])
		v.SwitchClass = ParseInt(row["switchClass"])
		//v.ClassSynergy = ParseIntArray(row["classSynergy"])
		//v.SynergyLevel = ParseIntArray(row["synergyLevel"])
		v.ArmorType = ParseIntArray(row["armorType"])
		if v.Id <= 0 {
			continue
		}
		if s.temp5[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load hero heroClass duplicate ID:%d", v.Id))
		}
		s.temp5[v.Id] = &v
	}

	s.temp6 = make(map[int32]*HeroCodexCfg)
	for _, row := range rawData["heroCodex"] {
		var v HeroCodexCfg
		v.HeroId = ParseInt(row["id"])
		v.Order = ParseInt(row["order"])
		v.StarPoints = ParseInt(row["starPoints"])
		if v.HeroId <= 0 {
			continue
		}
		if s.temp6[v.HeroId] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load hero heroCodex duplicate ID:%d", v.HeroId))
		}
		s.temp6[v.HeroId] = &v
	}

	s.temp7 = make(map[int32]map[int32]map[int32]*HeroLevelCfg)
	for _, row := range rawData["heroLevel"] {
		var v HeroLevelCfg
		v.Id = ParseInt(row["id"])
		v.HeroPotential = ParseInt(row["heroPotential"])
		v.HeroClass = ParseInt(row["heroClass"])
		v.HeroLevel = ParseInt(row["heroLevel"])
		v.LevelStat = ParseIntArray(row["levelStat"])
		v.StatValue = ParseIntArray(row["statValue"])
		v.LevelMaterials = ParseIntArray(row["levelMaterials"])
		v.Cost = ParseIntArray(row["cost"])
		if v.Id <= 0 {
			continue
		}
		m1 := s.temp7[v.HeroPotential]
		if m1 == nil {
			m1 = make(map[int32]map[int32]*HeroLevelCfg)
			s.temp7[v.HeroPotential] = m1
		}
		m2 := m1[v.HeroClass]
		if m2 == nil {
			m2 = make(map[int32]*HeroLevelCfg)
			m1[v.HeroClass] = m2
		}
		m2[v.HeroLevel] = &v
	}

	s.temp8 = make(map[int32]map[int32]map[int32]*HeroStarCfg)
	for _, row := range rawData["heroStar"] {
		var v HeroStarCfg
		v.Id = ParseInt(row["id"])
		v.HeroPotential = ParseInt(row["heroPotential"])
		v.HeroClass = ParseInt(row["heroClass"])
		v.HeroStar = ParseInt(row["heroStar"])
		v.StarStat = ParseIntArray(row["starStat"])
		v.StatNum = ParseIntArray(row["statNum"])
		v.BaseCard = ParseIntArray(row["baseCard"])
		v.SameClass = ParseIntArray(row["sameClass"])
		v.AnyClass = ParseIntArray(row["anyClass"])
		v.UniversalHero = ParseIntArray(row["universalHero"])
		v.Cost = ParseItem(row["cost"])
		if v.Id <= 0 {
			continue
		}
		m1 := s.temp8[v.HeroPotential]
		if m1 == nil {
			m1 = make(map[int32]map[int32]*HeroStarCfg)
			s.temp8[v.HeroPotential] = m1
		}
		m2 := m1[v.HeroClass]
		if m2 == nil {
			m2 = make(map[int32]*HeroStarCfg)
			m1[v.HeroClass] = m2
		}
		m2[v.HeroStar] = &v
	}

	s.temp9 = make(map[int32]*LevelRatioCfg)
	for _, row := range rawData["levelRatio"] {
		var v LevelRatioCfg
		v.Id = ParseInt(row["id"])
		v.LevelRange = ParseIntArray(row["levelRange"])
		v.LevelRatio = ParseIntArray(row["levelRatio"])
		v.CostRatio = ParseIntArray(row["costRatio"])
		if v.Id <= 0 {
			continue
		}
		s.temp9[v.Id] = &v
	}

	s.temp10 = make(map[int32]map[int32]*StarEffectCfg)
	for _, row := range rawData["starEffect"] {
		var v StarEffectCfg
		v.Id = ParseInt(row["id"])
		v.HeroId = ParseInt(row["heroId"])
		v.HeroStar = ParseInt(row["heroStar"])
		v.BasicSkill = ParseInt(row["basicSkill"])
		v.ActiveSkill = ParseInt(row["activeSkill"])
		v.SkillType1 = ParseInt(row["skillType1"])
		v.PassiveSkill1 = ParseInt(row["passiveSkill1"])
		v.SkillType2 = ParseInt(row["skillType2"])
		v.PassiveSkill2 = ParseInt(row["passiveSkill2"])
		v.ChangeClass = ParseIntArray(row["changeClass"])
		v.ClassSkill = ParseIntArray(row["classSkill"])
		v.UnlockClass = ParseInt(row["unlockClass"])
		v.UintsId = ParseIntArray(row["uintsId"])
		if v.Id <= 0 {
			continue
		}
		m1 := s.temp10[v.HeroId]
		if m1 == nil {
			m1 = make(map[int32]*StarEffectCfg)
			s.temp10[v.HeroId] = m1
		}
		m1[v.HeroStar] = &v
	}

	s.temp11 = make(map[int32]*StarCfg)
	for _, row := range rawData["star"] {
		var v StarCfg
		v.Id = ParseInt(row["id"])
		v.Rarity = ParseInt(row["rarity"])
		v.StarColor = row["starColor"]
		v.StarNum = ParseInt(row["starNum"])
		if v.Id <= 0 {
			continue
		}
		s.temp11[v.Id] = &v
	}

	return nil
}

func (s *HeroCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if id < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load hero codeReward ID:%d", id))
		}
		for _, itemId := range v.Reward {
			if GetItemCfg(int32(itemId.ID)) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load hero codeReward item:%d", itemId.ID))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load comboSkill error skill ID:%d", id))
		}
		if v.Level == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load comboSkill error skill ID:%d level is nil", id))
		}
		if v.StarLimit == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load comboSkill error skill ID:%d starLimit is nil", id))
		}
		for _, heroList := range v.LimitedHero {
			if GetHeroBaseCfg(heroList) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load comboSkill error hero ID:%d not in heroBase", heroList))
			}
		}
	}
	for id, v := range s.temp3 {
		if v.HeroId <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d", id))
		}
		if v.Stat == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d stat is nil", id))
		}
		if v.StatValue == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d statValue is nil", id))
		}
		if v.Cap == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d cap is nil", id))
		}
		if v.AttackRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d attackRange is invalid", id))
		}
		if v.PatrolRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d patrolRange is invalid", id))
		}
		if v.AttackRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d attackRange is invalid", id))
		}
		for _, skillId := range v.ComboSkill {
			if GetComboSkillCfg(skillId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load heroBase error hero ID:%d comboSkill:%d not in comboSkill", id, skillId))
			}
		}
	}
	for _, m1 := range s.temp4 {
		for _, m2 := range m1 {
			for id, v := range m2 {
				if v.Id <= 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load heroBreak error invalid ID:%d", id))
				}
				if v.BreakStat == nil || v.StatValue == nil || len(v.BreakStat) != len(v.StatValue) {
					return errors.New(fmt.Sprintf("[gameConfig] load heroBreak error invalid ID:%d", id))
				}
				if v.BreakMaterials == nil || v.BreakCost == nil || len(v.BreakMaterials) != len(v.BreakCost) {
					return errors.New(fmt.Sprintf("[gameConfig] load heroBreak error invalid ID:%d", id))
				}
				if v.HeroPotential < 1 || v.HeroPotential > 4 {
					return errors.New(fmt.Sprintf("[gameConfig] load heroBreak error invalid ID:%d", id))
				}
				if GetHeroClassCfg(v.HeroClass) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load heroBreak error class ID:%d not in heroClass", v.HeroClass))
				}
				for _, itemId := range v.BreakMaterials {
					if GetItemCfg(int32(itemId)) == nil {
						return errors.New(fmt.Sprintf("[gameConfig] load heroLevel item:%d", itemId))
					}
				}
			}
		}
	}
	for id, v := range s.temp5 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroClass error invalid ID:%d", id))
		}
		for _, classId := range v.ChangeClass {
			if classId > 0 && GetHeroClassCfg(classId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load heroClass error changeClass ID:%d not in heroClass", classId))
			}
		}
		if v.SwitchClass > 0 && GetHeroClassCfg(v.SwitchClass) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heroClass error switchClass ID:%d not in heroClass", v.SwitchClass))
		}
		//if v.ClassSynergy == nil || v.SynergyLevel == nil || len(v.ClassSynergy) != len(v.SynergyLevel) {
		//	return errors.New(fmt.Sprintf("[gameConfig] load heroClass error invalid ID:%d ", id))
		//}
		if v.ArmorType == nil || len(v.ArmorType) != 2 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroClass error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp6 {
		if v.HeroId <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroCodex error invalid ID:%d", id))
		}
		if GetHeroBaseCfg(v.HeroId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load heroCodex error hero ID :%d not in heroBase", v.HeroId))
		}
	}
	for _, m1 := range s.temp7 {
		for _, m2 := range m1 {
			for id, v := range m2 {
				if v.Id <= 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load heroLevel error invalid ID:%d", id))
				}
				if GetHeroClassCfg(v.HeroClass) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load heroLevel error hero class:%d not in heroClass", v.HeroClass))
				}
				for _, itemId := range v.LevelMaterials {
					if GetItemCfg(int32(itemId)) == nil {
						return errors.New(fmt.Sprintf("[gameConfig] load heroLevel item:%d", itemId))
					}
				}
				if v.LevelStat == nil || v.StatValue == nil || len(v.LevelStat) != len(v.StatValue) {
					return errors.New(fmt.Sprintf("[gameConfig] load heroLevel error invalid ID:%d", id))
				}
				if v.LevelMaterials == nil || v.Cost == nil || len(v.LevelMaterials) != len(v.Cost) {
					return errors.New(fmt.Sprintf("[gameConfig] load heroLevel error invalid ID:%d", id))
				}
			}
		}
	}
	for _, m1 := range s.temp8 {
		for _, m2 := range m1 {
			for id, v := range m2 {
				if v.Id <= 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load heroStar error invalid ID:%d", id))
				}
				if GetHeroClassCfg(v.HeroClass) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load heroStar error hero class:%d not in heroClass", v.HeroClass))
				}
				if v.StarStat == nil || v.StatNum == nil || len(v.StarStat) != len(v.StatNum) {
					return errors.New(fmt.Sprintf("[gameConfig] load heroStar error invalid ID:%d", id))
				}
			}
		}
	}
	for _, v := range s.temp9 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load heroLevel item:%d", v.Id))
		}
		if v.LevelRange == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load levelRatio error invalid ID:%d", v.Id))
		}
		if v.LevelRatio == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load levelRatio error invalid ID:%d", v.Id))
		}
		if v.CostRatio == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load costRatio error invalid ID:%d", v.Id))
		}
	}
	for _, m1 := range s.temp10 {
		for id, v := range m1 {
			if v.Id <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load starEffect error invalid ID:%d", id))
			}
			for _, HeroClass := range v.ChangeClass {
				if GetHeroClassCfg(HeroClass) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load starEffect error hero class:%d not in heroClass", HeroClass))
				}
			}
			if GetHeroBaseCfg(v.HeroId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load starEffect error invalid ID:%d", id))
			}
		}
	}
	for id, v := range s.temp11 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load star error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *HeroCfgLoader) apply() {
	codexReward.Store(s.temp1)
	comboSkill.Store(s.temp2)
	heroBase.Store(s.temp3)
	heroBreak.Store(s.temp4)
	heroClass.Store(s.temp5)
	heroCodex.Store(s.temp6)
	heroLevel.Store(s.temp7)
	heroStar.Store(s.temp8)
	levelRatio.Store(s.temp9)
	starEffect.Store(s.temp10)
	star.Store(s.temp11)
	heroAttrByLevelCfg.Store(s.HeroAttrByLevelCfg())
	heroAttrByBreakCfg.Store(s.HeroAttrByBreakCfg())
	heroAttrByStarCfg.Store(s.HeroAttrByStarCfg())
}

var codexReward atomic.Value
var comboSkill atomic.Value
var heroBase atomic.Value
var heroBreak atomic.Value
var heroClass atomic.Value
var heroCodex atomic.Value
var heroLevel atomic.Value
var heroStar atomic.Value
var levelRatio atomic.Value
var starEffect atomic.Value
var star atomic.Value

type CodexRewardCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 积分
	CodexPoints int32 `json:"codexPoints"`
	// 奖励
	Reward []*ItemConfig `json:"reward"`
}

type ComboSkillCfg struct {
	// 合体技id
	Id int32 `json:"id"`
	// 合体技等级
	Level int32 `json:"level"`
	// 限定英雄
	LimitedHero []int32 `json:"limitedHero"`
	// 星级限制
	StarLimit []int32 `json:"starLimit"`
	//主位id
	LeaderId int32 `json:"leaderId"`
}

type HeroBaseCfg struct {
	// 英雄id
	HeroId int32 `json:"heroId"`
	// 单位id
	UnitsId int32 `json:"unitsId"`
	// 潜力
	HeroPotential int32 `json:"heroPotential"`
	// 初始星级
	HeroStar int32 `json:"heroStar"`
	// 职业
	HeroClass int32 `json:"heroClass"`
	// 合体技
	ComboSkill []int32 `json:"comboSkill"`
	// 技能
	HeroSkill []int32 `json:"heroSkill"`
	// 初始属性
	Stat []int32 `json:"stat"`
	// 属性数值
	StatValue []int32 `json:"statValue"`
	// 攻击距离
	AttackRange int32 `json:"attackRange"`
	// 满星满级
	Cap []int32 `json:"cap"`
	// 巡逻范围
	PatrolRange int32 `json:"patrolRange"`
	// 索敌范围
	AggroRange int32 `json:"aggroRange"`
}

type HeroBreakCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 潜力
	HeroPotential int32 `json:"heroPotential"`
	// 职业
	HeroClass int32 `json:"heroClass"`
	// 阶数
	Rank int32 `json:"rank"`
	// 英雄等级
	HeroLevel int32 `json:"heroLevel"`
	// 进阶属性
	BreakStat []int32 `json:"breakStat"`
	// 属性数值
	StatValue []int32 `json:"statValue"`
	// 进阶材料
	BreakMaterials []int32 `json:"breakMaterials"`
	// 进阶消耗
	BreakCost []int32 `json:"breakCost"`
}

type HeroClassCfg struct {
	//职业
	Id int32 `json:"id"`
	// 向上转职
	ChangeClass []int32 `json:"changeClass"`
	// 平行转职
	SwitchClass int32 `json:"switchClass"`
	//// 职业羁绊
	//ClassSynergy []int32 `json:"classSynergy"`
	//// 羁绊等级
	//SynergyLevel []int32 `json:"synergyLevel"`
	// 装备类型
	ArmorType []int32 `json:"armorType"`
}

type HeroCodexCfg struct {
	// 英雄id
	HeroId int32 `json:"id"`
	// 排序
	Order int32 `json:"order"`
	// 星级积分
	StarPoints int32 `json:"starPoints"`
}

type HeroLevelCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 潜力
	HeroPotential int32 `json:"heroPotential"`
	// 职业
	HeroClass int32 `json:"heroClass"`
	// 英雄等级
	HeroLevel int32 `json:"heroLevel"`
	// 升级属性
	LevelStat []int32 `json:"levelStat"`
	// 属性数值
	StatValue []int32 `json:"statValue"`
	// 升级材料
	LevelMaterials []int32 `json:"levelMaterials"`
	// 材料消耗
	Cost []int32 `json:"cost"`
}

type HeroStarCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 潜力
	HeroPotential int32 `json:"heroPotential"`
	// 职业
	HeroClass int32 `json:"heroClass"`
	// 英雄星级
	HeroStar int32 `json:"heroStar"`
	// 升星属性
	StarStat []int32 `json:"starStat"`
	// 升星数值
	StatNum []int32 `json:"statNum"`
	// 消耗本体
	BaseCard []int32 `json:"baseCard"`
	// 同职业英雄
	SameClass []int32 `json:"sameClass"`
	// 任意职业英雄
	AnyClass []int32 `json:"anyClass"`
	// 通用英雄材料
	UniversalHero []int32 `json:"universalHero"`
	// 消耗材料
	Cost *ItemConfig `json:"cost"`
}

type LevelRatioCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 等级区间
	LevelRange []int32 `json:"levelRange"`
	// 属性系数
	LevelRatio []int32 `json:"levelRatio"`
	// 消耗系数
	CostRatio []int32 `json:"costRatio"`
}

type StarEffectCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 英雄id
	HeroId int32 `json:"heroId"`
	// 英雄星级
	HeroStar int32 `json:"heroStar"`
	// 普攻
	BasicSkill int32 `json:"basicSkill"`
	// 主动
	ActiveSkill int32 `json:"activeSkill"`
	// 被动类型
	SkillType1 int32 `json:"skillType1"`
	// 被动1
	PassiveSkill1 int32 `json:"passiveSkill1"`
	// 被动类型
	SkillType2 int32 `json:"skillType2"`
	// 被动2
	PassiveSkill2 int32 `json:"passiveSkill2"`
	// 转职
	ChangeClass []int32 `json:"changeClass"`
	// 转职技能
	ClassSkill []int32 `json:"classSkill"`
	// unlockId
	UnlockClass int32 `json:"unlockClass"`
	// uintsId 单位Id
	UintsId []int32 `json:"uintsId"`
}

type StarCfg struct {
	// 星级
	Id int32 `json:"id"`
	// 品质框
	Rarity int32 `json:"rarity"`
	// 星星颜色
	StarColor string `json:"starColor"`
	// 星星数量
	StarNum int32 `json:"starNum"`
}

func GetStarCfg(id int32) *StarCfg {
	cfgMap := star.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*StarCfg)[id]
}

func GetCodexRewardCfg(id int32) *CodexRewardCfg {
	cfgMap := codexReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CodexRewardCfg)[id]
}

func GetComboSkillCfg(id int32) *ComboSkillCfg {
	cfgMap := comboSkill.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ComboSkillCfg)[id]
}

func GetAllComboSkillCfg() map[int32]*ComboSkillCfg {
	cfgMap := comboSkill.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ComboSkillCfg)
}

func GetHeroBaseCfg(id int32) *HeroBaseCfg {
	cfgMap := heroBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*HeroBaseCfg)[id]
}

func GetHeroBreakCfg(potential int32, class int32, rank int32) *HeroBreakCfg {
	cfgMap := heroBreak.Load()
	if cfgMap == nil {
		return nil
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroBreakCfg)[potential]
	if m1 == nil {
		return nil
	}
	m2 := m1[class]
	if m2 == nil {
		return nil
	}
	return m2[rank]
}

func GetHeroClassCfg(id int32) *HeroClassCfg {
	cfgMap := heroClass.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*HeroClassCfg)[id]
}

func GetHeroCodexCfg(id int32) *HeroCodexCfg {
	cfgMap := heroCodex.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*HeroCodexCfg)[id]
}

func GetHeroLevelCfg(potential int32, class int32, level int32) *HeroLevelCfg {
	cfgMap := heroLevel.Load()
	if cfgMap == nil {
		return nil
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroLevelCfg)[potential]
	if m1 == nil {
		return nil
	}
	m2 := m1[class]
	if m2 == nil {
		return nil
	}
	return m2[level]
}

func GetHeroStarCfg(potential int32, class int32, star int32) *HeroStarCfg {
	cfgMap := heroStar.Load()
	if cfgMap == nil {
		return nil
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroStarCfg)[potential]
	if m1 == nil {
		return nil
	}
	m2 := m1[class]
	if m2 == nil {
		return nil
	}
	return m2[star]
}

func GetLevelRatioCfg(id int32) *LevelRatioCfg {
	cfgMap := levelRatio.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*LevelRatioCfg)[id]
}

func GetStarEffectCfg(heroId int32, star int32) *StarEffectCfg {
	cfgMap := starEffect.Load()
	if cfgMap == nil {
		return nil
	}
	m1 := cfgMap.(map[int32]map[int32]*StarEffectCfg)[heroId]
	if m1 == nil {
		return nil
	}
	return m1[star]
}

func GetAllCodexRewardCfg() map[int32]*CodexRewardCfg {
	cfgMap := codexReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CodexRewardCfg)
}

func GetMaxBreakNum(potential int32, class int32) int32 {
	cfgMap := heroBreak.Load()
	if cfgMap == nil {
		return 0
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroBreakCfg)[potential]
	if m1 == nil {
		return 0
	}
	m2 := m1[class]
	if m2 == nil {
		return 0
	}
	var maxRank int32 = 0
	for rank := range m2 {
		if rank > maxRank {
			maxRank = rank
		}
	}
	return maxRank
}

func GetMaxStarNumByHeroId(heroId int32) int32 {
	cfg := GetHeroBaseCfg(heroId)
	if cfg == nil {
		return 0
	}
	if len(cfg.Cap) >= 2 {
		return cfg.Cap[1]
	}
	return 0
}

func GetMaxStarNum(potential int32, class int32) int32 {
	cfgMap := heroStar.Load()
	if cfgMap == nil {
		return 0
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroStarCfg)[potential]
	if m1 == nil {
		return 0
	}
	m2 := m1[class]
	if m2 == nil {
		return 0
	}
	var maxStar int32 = 0
	for star := range m2 {
		if star > maxStar {
			maxStar = star
		}
	}
	return maxStar
}

func GetMinStarNum(potential int32, class int32) int32 {
	cfgMap := heroStar.Load()
	if cfgMap == nil {
		return 0
	}
	m1 := cfgMap.(map[int32]map[int32]map[int32]*HeroStarCfg)[potential]
	if m1 == nil {
		return 0
	}
	m2 := m1[class]
	if m2 == nil {
		return 0
	}
	var minStar int32 = 1000
	for star := range m2 {
		if star < minStar {
			minStar = star
		}
	}
	return minStar
}

func (h *HeroCfgLoader) HeroAttrByLevelCfg() map[int32]map[int32]map[int32]map[int32]int64 {
	attr := make(map[int32]map[int32]map[int32]map[int32]int64) // 潜力->职业->等级->属性ID->属性值
	cfgMap := heroLevel.Load()
	maxLevel := int32(0)
	levelRatioRaw := levelRatio.Load()
	if levelRatioRaw == nil {
		return attr
	}
	for _, v := range levelRatioRaw.(map[int32]*LevelRatioCfg) {
		if v.LevelRange[1] > maxLevel {
			maxLevel = v.LevelRange[1]
		}
	}
	if cfgMap == nil {
		return attr
	}
	for potential, m1 := range cfgMap.(map[int32]map[int32]map[int32]*HeroLevelCfg) {
		if attr[potential] == nil {
			attr[potential] = make(map[int32]map[int32]map[int32]int64)
		}
		for class, m2 := range m1 {
			if attr[potential][class] == nil {
				attr[potential][class] = make(map[int32]map[int32]int64)
			}
			for i := int32(1); i <= maxLevel; i++ {
				if attr[potential][class][i] == nil {
					attr[potential][class][i] = make(map[int32]int64)
				}
				cfg := m2[(i-1)%100+1]
				for id, statId := range cfg.LevelStat {
					if id < len(cfg.StatValue) {
						if i-1 > 0 {
							levelRatioRaw := levelRatio.Load()
							if levelRatioRaw == nil {
								continue
							}
							levelRatioMap := levelRatioRaw.(map[int32]*LevelRatioCfg)
							ratioCfg := levelRatioMap[(i-1)/100+1]
							if ratioCfg == nil {
								continue
							}
							attr[potential][class][i][statId] = int64(attr[potential][class][i-1][statId]) + int64(cfg.StatValue[id])*int64(ratioCfg.LevelRatio[id])/10000
						} else {
							attr[potential][class][i][statId] = int64(cfg.StatValue[id])
						}
					}
				}
			}
		}
	}
	return attr
}

func (h *HeroCfgLoader) HeroAttrByBreakCfg() map[int32]map[int32]map[int32]map[int32]int64 {
	attr := make(map[int32]map[int32]map[int32]map[int32]int64) // 潜力->职业->阶数->属性ID->属性值
	cfgMap := heroBreak.Load()
	if cfgMap == nil {
		return attr
	}
	for potential, m1 := range cfgMap.(map[int32]map[int32]map[int32]*HeroBreakCfg) {
		if attr[potential] == nil {
			attr[potential] = make(map[int32]map[int32]map[int32]int64)
		}
		for class, m2 := range m1 {
			if attr[potential][class] == nil {
				attr[potential][class] = make(map[int32]map[int32]int64)
			}
			for i := int32(1); i <= GetMaxBreakNum(potential, class); i++ {
				if attr[potential][class][i] == nil {
					attr[potential][class][i] = make(map[int32]int64)
				}
				cfg := m2[i]
				for id, statId := range cfg.BreakStat {
					if id < len(cfg.StatValue) {
						if i > 1 {
							attr[potential][class][i][statId] = int64(attr[potential][class][i-1][statId]) + int64(cfg.StatValue[id])
						} else {
							attr[potential][class][i][statId] = int64(cfg.StatValue[id])
						}
					}
				}
			}
		}
	}
	return attr
}
func (h *HeroCfgLoader) HeroAttrByStarCfg() map[int32]map[int32]map[int32]map[int32]int64 {
	attr := make(map[int32]map[int32]map[int32]map[int32]int64) // 潜力->职业->星级->属性ID->属性值
	cfgMap := heroStar.Load()
	if cfgMap == nil {
		return attr
	}
	for potential, m1 := range cfgMap.(map[int32]map[int32]map[int32]*HeroStarCfg) {
		if attr[potential] == nil {
			attr[potential] = make(map[int32]map[int32]map[int32]int64)
		}
		for class, m2 := range m1 {
			if attr[potential][class] == nil {
				attr[potential][class] = make(map[int32]map[int32]int64)
			}
			for i := GetMinStarNum(potential, class); i <= GetMaxStarNum(potential, class); i++ {
				if attr[potential][class][i] == nil {
					attr[potential][class][i] = make(map[int32]int64)
				}
				cfg := m2[i]
				for id, statId := range cfg.StarStat {
					if id < len(cfg.StatNum) {
						if i-1 >= 0 {
							attr[potential][class][i][statId] = int64(attr[potential][class][i-1][statId]) + int64(cfg.StatNum[id])
						} else {
							attr[potential][class][i][statId] = int64(cfg.StatNum[id])
						}
					}
				}
			}
		}
	}
	return attr
}

func GetHeroBaseAttr(heroId int32, attrId int32) int64 {
	for i, v := range GetHeroBaseCfg(heroId).Stat {
		if v == attrId {
			return int64(GetHeroBaseCfg(heroId).StatValue[i])
		}
	}
	return 0
}

func GetSecondAttr(potential, class, level, breakNum, starLevel int32, attrId int32) int64 {
	var attr int64 = 0
	if level > 1 {
		if m, ok := heroAttrByLevelCfg.Load().(map[int32]map[int32]map[int32]map[int32]int64); ok && m != nil {
			attr += m[potential][class][level-1][attrId]
		}
	}
	if breakNum > 0 {
		if m, ok := heroAttrByBreakCfg.Load().(map[int32]map[int32]map[int32]map[int32]int64); ok && m != nil {
			attr += m[potential][class][breakNum][attrId]
		}
	}
	if starLevel >= GetMinStarNum(potential, class) {
		if m, ok := heroAttrByStarCfg.Load().(map[int32]map[int32]map[int32]map[int32]int64); ok && m != nil {
			attr += m[potential][class][starLevel][attrId]
		}
	}
	return attr
}

func GetHeroUnitsId(heroBaseId, start, class int32) int32 {
	heroCfg := GetHeroBaseCfg(heroBaseId)
	if heroCfg == nil {
		return 0
	}
	heroMaxStar := GetMaxStarNumByHeroId(heroBaseId)
	for i := heroCfg.HeroStar; i <= heroMaxStar; i++ {
		cfg := GetStarEffectCfg(heroBaseId, i)
		if cfg == nil {
			continue
		}
		if len(cfg.ChangeClass) == 0 {
			if class == heroCfg.HeroClass {
				return cfg.UintsId[0]
			}
		}
		for id, v := range cfg.ChangeClass {
			if v == class {
				return cfg.UintsId[id]
			}
		}
	}
	return 0
}
