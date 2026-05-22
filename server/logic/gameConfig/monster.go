package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("monster", &MonsterCfgLoader{})
}

type MonsterCfgLoader struct {
	temp1 map[int32]*MonsterCfg
}

var _ configLoaderInterface = (*MonsterCfgLoader)(nil)

func (s *MonsterCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/monster.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*MonsterCfg)
	for _, row := range rawData["monster"] {
		var v MonsterCfg
		v.Id = ParseInt(row["id"])
		v.Type = ParseInt(row["type"])
		v.Units = ParseInt(row["units"])
		v.HeroId = ParseInt(row["heroId"])
		v.Level = ParseInt(row["level"])
		v.Star = ParseInt(row["star"])
		v.DmgType = ParseInt(row["dmgType"])
		v.Hp = ParseInt(row["hp"])
		v.PhyATK = ParseInt(row["phyATK"])
		v.MagATK = ParseInt(row["magATK"])
		v.PhyDEF = ParseInt(row["phyDEF"])
		v.MagDEF = ParseInt(row["magDEF"])
		v.PhyCrit = ParseInt(row["phyCrit"])
		v.MagCrit = ParseInt(row["magCrit"])
		v.PhyCritRes = ParseInt(row["phyCritRes"])
		v.MagCritRes = ParseInt(row["magCritRes"])
		v.PhyCritDam = ParseInt(row["phyCritDam"])
		v.MagCritDam = ParseInt(row["magCritDam"])
		v.PhyHit = ParseInt(row["phyHit"])
		v.MagHit = ParseInt(row["magHit"])
		v.PhyDodge = ParseInt(row["phyDodge"])
		v.MagDodge = ParseInt(row["magDodge"])
		v.PhyPen = ParseInt(row["phyPen"])
		v.MagPen = ParseInt(row["magPen"])
		v.PhyBlock = ParseInt(row["phyBlock"])
		v.MagBlock = ParseInt(row["magBlock"])
		v.FinalDamReduction = ParseInt(row["finalDamReduction"])
		v.Toughness = ParseInt(row["toughness"])
		v.AtkSpeed = ParseInt(row["atkSpeed"])
		v.MoveSpeed = ParseInt(row["moveSpeed"])
		v.Skill = ParseIntArray(row["skill"])
		v.PatrolRange = ParseInt(row["patrolRange"])
		v.AggroRange = ParseInt(row["aggroRange"])
		v.AttackRange = ParseInt(row["attackRange"])
		v.NormalAtk = ParseInt(row["normalAtk"])
		v.Attr = map[int32]int64{
			enum.AttributeBasicHp:                     int64(v.Hp),
			enum.AttributeBasicPhysicalAttack:         int64(v.PhyATK),
			enum.AttributeBasicMagicalAttack:          int64(v.MagATK),
			enum.AttributeBasicPhysicalDefense:        int64(v.PhyDEF),
			enum.AttributeBasicMagicalDefense:         int64(v.MagDEF),
			enum.AttributeBasicPhysicalCritRate:       int64(v.PhyCrit),
			enum.AttributeBasicMagicalCritRate:        int64(v.MagCrit),
			enum.AttributeBasicPhysicalCritResistance: int64(v.PhyCritRes),
			enum.AttributeBasicMagicalCritResistance:  int64(v.MagCritRes),
			enum.AttributeBasicPhysicalCritDamage:     int64(v.PhyCritDam),
			enum.AttributeBasicMagicalCritDamage:      int64(v.MagCritDam),
			enum.AttributeBasicPhysicalHit:            int64(v.PhyHit),
			enum.AttributeBasicMagicalHit:             int64(v.MagHit),
			enum.AttributeBasicPhysicalDodge:          int64(v.PhyDodge),
			enum.AttributeBasicMagicalDodge:           int64(v.MagDodge),
			enum.AttributeBasicPhysicalPenetration:    int64(v.PhyPen),
			enum.AttributeBasicMagicalPenetration:     int64(v.MagPen),
			enum.AttributeBasicPhysicalBlock:          int64(v.PhyBlock),
			enum.AttributeBasicMagicalBlock:           int64(v.MagBlock),
			enum.AttributeBasicFinalDamageReduction:   int64(v.FinalDamReduction),
			enum.AttributeBasicToughness:              int64(v.Toughness),
			enum.AttributeBasicAttackSpeed:            int64(v.AtkSpeed),
			enum.AttributeBasicMoveSpeed:              int64(v.MoveSpeed),
		}
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *MonsterCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid ID:%d", id))
		}
		if !enum.IsValidMonsterType(v.Type) {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid Type:%d,configId:%d", v.Type, id))
		}
		if v.Level < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid Level:%d,configId:%d", v.Level, id))
		}
		if v.Star < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid Star:%d,configId:%d", v.Star, id))
		}
		if !enum.IsValidDamageType(v.DmgType) {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid DmgType:%d,configId:%d", v.DmgType, id))
		}
		if v.Hp <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid Hp:%d,configId:%d", v.Hp, id))
		}
		if v.PhyATK < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyATK:%d,configId:%d", v.PhyATK, id))
		}
		if v.MagATK < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagATK:%d,configId:%d", v.MagATK, id))
		}
		if v.PhyDEF < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyDEF:%d,configId:%d", v.PhyDEF, id))
		}
		if v.MagDEF < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagDEF:%d,configId:%d", v.MagDEF, id))
		}
		if v.PhyCrit < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyCrit:%d,configId:%d", v.PhyCrit, id))
		}
		if v.MagCrit < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagCrit:%d,configId:%d", v.MagCrit, id))
		}
		if v.PhyCritRes < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyCritRes:%d,configId:%d", v.PhyCritRes, id))
		}
		if v.MagCritRes < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagCritRes:%d,configId:%d", v.MagCritRes, id))
		}
		if v.PhyCritDam < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyCritDam:%d,configId:%d", v.PhyCritDam, id))
		}
		if v.MagCritDam < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagCritDam:%d,configId:%d", v.MagCritDam, id))
		}
		if v.PhyHit < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyHit:%d,configId:%d", v.PhyHit, id))
		}
		if v.MagHit < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagHit:%d,configId:%d", v.MagHit, id))
		}
		if v.PhyDodge < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyDodge:%d,configId:%d", v.PhyDodge, id))
		}
		if v.MagDodge < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagDodge:%d,configId:%d", v.MagDodge, id))
		}
		if v.PhyPen < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyPen:%d,configId:%d", v.PhyPen, id))
		}
		if v.MagPen < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagPen:%d,configId:%d", v.MagPen, id))
		}
		if v.PhyBlock < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PhyBlock:%d,configId:%d", v.PhyBlock, id))
		}
		if v.MagBlock < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MagBlock:%d,configId:%d", v.MagBlock, id))
		}
		if v.FinalDamReduction < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid FinalDamReduction:%d,configId:%d", v.FinalDamReduction, id))
		}
		if v.Toughness < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid Toughness:%d,configId:%d", v.Toughness, id))
		}
		if v.AtkSpeed < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid AtkSpeed:%d,configId:%d", v.AtkSpeed, id))
		}
		if v.MoveSpeed < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid MoveSpeed:%d,configId:%d", v.MoveSpeed, id))
		}
		if v.PatrolRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid PatrolRange:%d,configId:%d", v.PatrolRange, id))
		}
		if v.AggroRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid AggroRange:%d,configId:%d", v.AggroRange, id))
		}
		if v.AttackRange < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monster error invalid AttackRange:%d,configId:%d", v.AggroRange, id))
		}
		//TODO:技能检测
	}
	return nil
}

func (s *MonsterCfgLoader) apply() {
	monster.Store(s.temp1)
}

var monster atomic.Value

type MonsterCfg struct {
	// 怪物id
	Id int32 `json:"id"`
	// 类型
	Type int32 `json:"type"`
	// 单位Id
	Units int32 `json:"units"`
	// 英雄Id
	HeroId int32 `json:"heroId"`
	// 等级
	Level int32 `json:"level"`
	// 星级
	Star int32 `json:"star"`
	// 伤害类型
	DmgType int32 `json:"dmgType"`
	// 怪物生命
	Hp int32 `json:"hp"`
	// 物理攻击
	PhyATK int32 `json:"phyATK"`
	// 魔法攻击
	MagATK int32 `json:"magATK"`
	// 物理防御
	PhyDEF int32 `json:"phyDEF"`
	// 魔法防御
	MagDEF int32 `json:"magDEF"`
	// 物理暴击
	PhyCrit int32 `json:"phyCrit"`
	// 魔法暴击
	MagCrit int32 `json:"magCrit"`
	// 物理暴击抗性
	PhyCritRes int32 `json:"phyCritRes"`
	// 魔法暴击抗性
	MagCritRes int32 `json:"magCritRes"`
	// 物理暴击伤害
	PhyCritDam int32 `json:"phyCritDam"`
	// 魔法暴击伤害
	MagCritDam int32 `json:"magCritDam"`
	// 物理命中
	PhyHit int32 `json:"phyHit"`
	// 魔法命中
	MagHit int32 `json:"magHit"`
	// 物理闪避
	PhyDodge int32 `json:"phyDodge"`
	// 魔法闪避
	MagDodge int32 `json:"magDodge"`
	// 物理穿透
	PhyPen int32 `json:"phyPen"`
	// 魔法穿透
	MagPen int32 `json:"magPen"`
	// 物理格挡
	PhyBlock int32 `json:"phyBlock"`
	// 魔法格挡
	MagBlock int32 `json:"magBlock"`
	// 受到最终伤害减少
	FinalDamReduction int32 `json:"finalDamReduction"`
	// 韧性
	Toughness int32 `json:"toughness"`
	// 攻击速度
	AtkSpeed int32 `json:"atkSpeed"`
	// 移动速度
	MoveSpeed int32 `json:"moveSpeed"`
	// 技能
	Skill []int32 `json:"skill"`
	// 巡逻范围
	PatrolRange int32 `json:"patrolRange"`
	// 索敌范围
	AggroRange int32 `json:"aggroRange"`
	// 攻击距离
	AttackRange int32 `json:"attackRange"`
	// 普攻技能
	NormalAtk int32 `json:"normalAtk"`
	//所有属性
	Attr map[int32]int64
}

func GetMonsterCfg(id int32) *MonsterCfg {
	cfgMap := monster.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*MonsterCfg)[id]
}
