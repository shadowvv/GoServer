package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("role", &RoleCfgLoader{})
}

type RoleCfgLoader struct {
	temp1 map[int32]*RoleLevelCfg
	temp2 map[int32]*RoleNameCfg
}

var _ configLoaderInterface = (*RoleCfgLoader)(nil)

func (s *RoleCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/role.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*RoleLevelCfg)
	for _, row := range rawData["roleLevel"] {
		var v RoleLevelCfg
		v.Id = ParseInt(row["id"])
		v.Age = ParseInt(row["age"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load roleLevel error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*RoleNameCfg)
	for _, row := range rawData["roleName"] {
		var v RoleNameCfg
		v.Id = ParseInt(row["id"])
		v.Name = row["name"]
		v.NameOrder = ParseInt(row["nameOrder"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load roleName error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	return nil
}

func (s *RoleCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load roleLevel error invalid ID:%d", id))
		}
		if v.Age <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load roleLevel error invalid Age:%d,configId:%d", v.Age, id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load roleName error invalid ID:%d", id))
		}
		if v.NameOrder != 0 && v.NameOrder != 1 {
			return errors.New(fmt.Sprintf("[gameConfig] load roleName error invalid NameOrder:%d,configId:%d", v.NameOrder, id))
		}
		if v.Name == "" {
			return errors.New(fmt.Sprintf("[gameConfig] load roleName error invalid Name:%s,configId:%d", v.Name, id))
		}
	}
	return nil
}

func (s *RoleCfgLoader) apply() {
	roleLevel.Store(s.temp1)
	roleName.Store(s.temp2)

	temp := make(map[int32][]string)
	for _, v := range s.temp2 {
		if temp[v.NameOrder] == nil {
			temp[v.NameOrder] = make([]string, 0)
		}
		temp[v.NameOrder] = append(temp[v.NameOrder], v.Name)
	}
	roleNickname.Store(temp)
}

var roleLevel atomic.Value
var roleName atomic.Value
var roleNickname atomic.Value

type RoleLevelCfg struct {
	// 账号等级
	Id int32 `json:"id"`
	// 时代
	Age int32 `json:"age"`
}

type RoleNameCfg struct {
	// 名称id
	Id int32 `json:"id"`
	// 账号名称
	Name string `json:"name"`
	// 名称顺序
	NameOrder int32 `json:"nameOrder"`
}

// 计算战斗力辅助函数
func GetAttrMapPower(level int32, attrs map[int32]int64) float64 {
	levelCfg := GetRoleLevelCfg(level)
	if levelCfg == nil {
		levelCfg = GetRoleLevelCfg(1)
	}

	attrRatio := GetCombatCoefficientCfg(levelCfg.Age)
	if attrRatio == nil {
		attrRatio = GetCombatCoefficientCfg(1)
	}
	power := float64(0)
	for attrId, attr := range attrs {
		if _, ok := attrRatio[attrId]; !ok {
			continue
		}
		// 攻击速度默认初始值为10000，超过10000的部分才算战力
		if attrId == enum.AttributeBasicAttackSpeed && attr >= 10000 {
			power += float64(attr-10000) * (attrRatio[attrId] / 10000)
		} else {
			power += float64(attr) * (attrRatio[attrId] / 10000)
		}
	}

	return power
}

func GetRoleLevelCfg(id int32) *RoleLevelCfg {
	cfgMap := roleLevel.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*RoleLevelCfg)[id]
}

func RandomNickname() string {
	cfgMap := roleNickname.Load()
	if cfgMap == nil {
		return ""
	}
	cfg := cfgMap.(map[int32][]string)
	index := tool.RandInt(0, len(cfg[0])-1)
	nickname := cfg[0][index]
	index = tool.RandInt(0, len(cfg[1])-1)
	nickname += cfg[1][index]
	return nickname
}
