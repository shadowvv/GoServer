package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("sect", &SectCfgLoader{})
}

type SectCfgLoader struct {
	temp1 map[int32]*SectlevelCfg
	temp2 map[int32]*SectpositionCfg
}

var _ configLoaderInterface = (*SectCfgLoader)(nil)

func (s *SectCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/sect.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*SectlevelCfg)
	for _, row := range rawData["sectlevel"] {
		var v SectlevelCfg
		v.Id = ParseInt(row["id"])
		v.Num = ParseInt(row["num"])
		v.ContribValue = ParseInt(row["contribValue"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load sectlevel error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*SectpositionCfg)
	for _, row := range rawData["sectposition"] {
		var v SectpositionCfg
		v.Id = ParseInt(row["id"])
		v.Permit = ParseIntArray(row["permit"])
		v.PlayerNum = ParseInt(row["playerNum"])
		v.MinContribute = ParseInt(row["minContribute"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load sectposition error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	return nil
}

func (s *SectCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load sectlevel error invalid configId:%d", id))
		}
		if v.Num <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load sectlevel error num<=0 num:%d, configId:%d", v.Num, id))
		}
		if v.ContribValue < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load sectlevel error contribValue<0 num:%d, configId:%d", v.ContribValue, id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load sectposition error invalid ID:%d", id))
		}
		for _, permit := range v.Permit {
			if !enum.IsValidPermit(permit) {
				return errors.New(fmt.Sprintf("[gameConfig] load sectposition error invalid permit:%d,configId:%d", permit, v.Id))
			}
		}
	}
	return nil
}

func (s *SectCfgLoader) apply() {
	sectlevel.Store(s.temp1)
	sectposition.Store(s.temp2)
}

var sectlevel atomic.Value
var sectposition atomic.Value

type SectlevelCfg struct {
	// 联盟等级
	Id int32 `json:"id"`
	// 联盟人数上限
	Num int32 `json:"num"`
	// 联盟累计需求贡献值
	ContribValue int32 `json:"contribValue"`
}

type SectpositionCfg struct {
	// id
	Id int32 `json:"id"`
	// 权限
	Permit []int32 `json:"permit"`
	// 人数
	PlayerNum int32 `json:"playerNum"`
	// 最低贡献值
	MinContribute int32 `json:"minContribute"`
}

func GetAllianceLevelCfg(id int32) *SectlevelCfg {
	cfgMap := sectlevel.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SectlevelCfg)[id]
}

func GetAlliancePositionCfg(id int32) *SectpositionCfg {
	cfgMap := sectposition.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SectpositionCfg)[id]
}
