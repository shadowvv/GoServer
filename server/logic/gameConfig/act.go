package gameConfig

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("act", &ActCfgLoader{})
}

type ActCfgLoader struct {
	temp1 map[int32]*TurnTableCfg
	temp2 map[int32]*TurnTableMainCfg
	temp3 map[int32]*UsuallyUseCfg
}

var _ configLoaderInterface = (*ActCfgLoader)(nil)

func (s *ActCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/act.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*TurnTableCfg)
	for _, row := range rawData["TurnTable"] {
		var v TurnTableCfg
		v.Id = ParseInt(row["id"])
		v.ModId = ParseInt(row["modId"])
		v.Drop = ParseInt(row["drop"])
		v.MinimumGuarantee = ParseIntArray(row["minimumGuarantee"])
		v.Limit = ParseInt(row["limit"])
		v.Weight = ParseInt(row["weight"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*TurnTableMainCfg)
	for _, row := range rawData["TurnTableMain"] {
		var v TurnTableMainCfg
		v.Id = ParseInt(row["id"])
		v.ActId = ParseInt(row["actId"])
		v.Use = ParseItem(row["use"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTableMain error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*UsuallyUseCfg)
	for _, row := range rawData["UsuallyUse"] {
		var v UsuallyUseCfg
		v.Id = ParseInt(row["id"])
		v.ModId = ParseInt(row["modId"])
		v.Type = ParseInt(row["type"])
		v.Param = ParseInt(row["param"])
		v.ItemId = ParseInt(row["itemId"])
		v.Drop = ParseItemArray(row["drop"])
		v.LoopLayer = ParseInt(row["loopLayer"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	return nil
}

func (s *ActCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid ID:%d", id))
		}
		if s.temp2[v.ModId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid modId:%d", id))
		}
		if v.Drop <= 0 || GetDropCfg(v.Drop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid drop:%d", id))
		}
		if len(v.MinimumGuarantee) != 0 && len(v.MinimumGuarantee) != 2 {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid minimumGuarantee:%d", id))
		}
		if len(v.MinimumGuarantee) == 2 && (v.MinimumGuarantee[0] <= 0 || v.MinimumGuarantee[1] <= 0 || v.MinimumGuarantee[1] > v.MinimumGuarantee[0]) {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid minimumGuarantee:%d", id))
		}
		if v.Limit < 0 || v.Weight <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTable error invalid weight:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTableMain error invalid ID:%d", id))
		}
		if v.ActId <= 0 || v.Use == nil || v.Use.ID <= 0 || v.Use.Num <= 0 || GetItemCfg(v.Use.ID) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load TurnTableMain error invalid use:%d", id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid ID:%d", id))
		}
		if s.temp2[v.ModId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid modId:%d", id))
		}
		if v.Type != 1 && v.Type != 2 {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid type:%d", id))
		}
		if v.Param <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid param:%d", id))
		}
		if v.Type == 2 && (v.ItemId <= 0 || GetItemCfg(v.ItemId) == nil) {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid itemId:%d", id))
		}
		for _, reward := range v.Drop {
			if reward == nil || reward.ID <= 0 || reward.Num <= 0 || GetItemCfg(reward.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid reward:%d", id))
			}
		}
		if v.LoopLayer < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load UsuallyUse error invalid loopLayer:%d", id))
		}
	}
	return nil
}

func (s *ActCfgLoader) apply() {
	TurnTable.Store(s.temp1)
	TurnTableMain.Store(s.temp2)
	UsuallyUse.Store(s.temp3)
	InitTurnTableAct()
}

var TurnTable atomic.Value
var TurnTableMain atomic.Value
var UsuallyUse atomic.Value
var turnTableByModId atomic.Value
var usuallyUseByModId atomic.Value
var usuallyUseByModIdType atomic.Value
var turnTableMainByActId atomic.Value

type TurnTableCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 模板id
	ModId int32 `json:"modId"`
	// 奖品
	Drop int32 `json:"drop"`
	// 保底限量次数
	MinimumGuarantee []int32 `json:"minimumGuarantee"`
	// 限量次数
	Limit int32 `json:"limit"`
	// 权重
	Weight int32 `json:"weight"`
}

type TurnTableMainCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 活动id
	ActId int32 `json:"actId"`
	// 单抽消耗
	Use *ItemConfig `json:"use"`
}

type UsuallyUseCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 模板id
	ModId int32 `json:"modId"`
	// 类型
	Type int32 `json:"type"`
	// 参数
	Param int32 `json:"param"`
	// 累计奖励参数
	ItemId int32 `json:"itemId"`
	// 累积奖励
	Drop []*ItemConfig `json:"drop"`
	// 循环层
	LoopLayer int32 `json:"loopLayer"`
}

func GetTurnTableCfg(id int32) *TurnTableCfg {
	cfgMap := TurnTable.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TurnTableCfg)[id]
}

func GetTurnTableMainCfg(id int32) *TurnTableMainCfg {
	cfgMap := TurnTableMain.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TurnTableMainCfg)[id]
}

func GetUsuallyUseCfg(id int32) *UsuallyUseCfg {
	cfgMap := UsuallyUse.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*UsuallyUseCfg)[id]
}

func InitTurnTableAct() {
	turnTableMap := make(map[int32][]*TurnTableCfg)
	turnTableCfgMap := TurnTable.Load()
	if turnTableCfgMap != nil {
		for _, cfg := range turnTableCfgMap.(map[int32]*TurnTableCfg) {
			turnTableMap[cfg.ModId] = append(turnTableMap[cfg.ModId], cfg)
		}
	}
	for _, cfgs := range turnTableMap {
		sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Id < cfgs[j].Id })
	}
	turnTableByModId.Store(turnTableMap)

	useMap := make(map[int32][]*UsuallyUseCfg)
	useTypeMap := make(map[int32]map[int32][]*UsuallyUseCfg)
	useCfgMap := UsuallyUse.Load()
	if useCfgMap != nil {
		for _, cfg := range useCfgMap.(map[int32]*UsuallyUseCfg) {
			useMap[cfg.ModId] = append(useMap[cfg.ModId], cfg)
			if useTypeMap[cfg.ModId] == nil {
				useTypeMap[cfg.ModId] = make(map[int32][]*UsuallyUseCfg)
			}
			useTypeMap[cfg.ModId][cfg.Type] = append(useTypeMap[cfg.ModId][cfg.Type], cfg)
		}
	}
	for _, cfgs := range useMap {
		sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Id < cfgs[j].Id })
	}
	for _, typeMap := range useTypeMap {
		for _, cfgs := range typeMap {
			sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Id < cfgs[j].Id })
		}
	}
	usuallyUseByModId.Store(useMap)
	usuallyUseByModIdType.Store(useTypeMap)

	mainByAct := make(map[int32][]*TurnTableMainCfg)
	mainCfgMap := TurnTableMain.Load()
	if mainCfgMap != nil {
		for _, cfg := range mainCfgMap.(map[int32]*TurnTableMainCfg) {
			mainByAct[cfg.ActId] = append(mainByAct[cfg.ActId], cfg)
		}
	}
	for _, cfgs := range mainByAct {
		sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Id < cfgs[j].Id })
	}
	turnTableMainByActId.Store(mainByAct)
}

func GetTurnTableCfgsByModId(modId int32) []*TurnTableCfg {
	v := turnTableByModId.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*TurnTableCfg)[modId]
}

func GetUsuallyUseCfgsByModId(modId int32) []*UsuallyUseCfg {
	v := usuallyUseByModId.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*UsuallyUseCfg)[modId]
}

func GetUsuallyUseCfgsByModIdAndType(modId, typ int32) []*UsuallyUseCfg {
	v := usuallyUseByModIdType.Load()
	if v == nil {
		return nil
	}
	typeMap := v.(map[int32]map[int32][]*UsuallyUseCfg)[modId]
	if typeMap == nil {
		return nil
	}
	return typeMap[typ]
}

func GetTurnTableMainCfgsByActId(actId int32) []*TurnTableMainCfg {
	v := turnTableMainByActId.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*TurnTableMainCfg)[actId]
}
