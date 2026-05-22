package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("guide", &GuideCfgLoader{})
}

type GuideCfgLoader struct {
	temp1 map[int32]*GuideCfg
}

var _ configLoaderInterface = (*GuideCfgLoader)(nil)

func (s *GuideCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/guide.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*GuideCfg)
	for _, row := range rawData["guide"] {
		var v GuideCfg
		v.Id = ParseInt(row["id"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load guide error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *GuideCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load guide error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *GuideCfgLoader) apply() {
	guide.Store(s.temp1)
}

var guide atomic.Value

type GuideCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 列表组件
	List string `json:"list"`
	// 列表中的按钮下标
	BtnIndex string `json:"btnIndex"`
}

func GetGuideCfg(id int32) *GuideCfg {
	cfgMap := guide.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*GuideCfg)[id]
}
