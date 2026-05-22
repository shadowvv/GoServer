package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("shop", &ShopCfgLoader{})
}

type ShopCfgLoader struct {
	temp4 map[int32]*WeeklyPassCfg
}

var _ configLoaderInterface = (*ShopCfgLoader)(nil)

func (s *ShopCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/shop.json`, &rawData); err != nil {
		return err
	}

	s.temp4 = make(map[int32]*WeeklyPassCfg)
	for _, row := range rawData["firstPackage"] {
		var v WeeklyPassCfg
		v.Id = ParseInt(row["id"])
		v.DropId = ParseIntArray(row["dropId"])
		v.ValidityPeriod = ParseInt(row["validityPeriod"])
		v.ServerLimit = ParseInt(row["serverLimit"])
		if v.Id <= 0 {
			continue
		}
		if s.temp4[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load firstPackage error duplicate ID:%d", v.Id))
		}
		s.temp4[v.Id] = &v
	}

	for _, row := range rawData["weeklyPass"] {
		var v WeeklyPassCfg
		v.Id = ParseInt(row["id"])
		v.DropId = ParseIntArray(row["dropId"])
		v.ValidityPeriod = ParseInt(row["validityPeriod"])
		v.ServerLimit = ParseInt(row["serverLimit"])
		if v.Id <= 0 {
			continue
		}
		if s.temp4[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load weeklyPass error duplicate ID:%d", v.Id))
		}
		s.temp4[v.Id] = &v
	}

	return nil
}

func (s *ShopCfgLoader) checkData() error {
	for id, v := range s.temp4 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load shop error invalid ID:%d", id))
		}
		if GetStillShopCfg(v.Id) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load shop error invalid ID:%d,not in pack", id))
		}
		if v.ValidityPeriod <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load shop error invalid ID:%d,validityPeriod <= 0", id))
		}
		if int32(len(v.DropId)) != v.ValidityPeriod {
			return errors.New(fmt.Sprintf("[gameConfig] load shop error invalid ID:%d,dropId length not equal validityPeriod", id))
		}
		for i := 0; i < len(v.DropId); i++ {
			if v.DropId[i] != 0 && GetDropCfg(v.DropId[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load shop error invalid ID:%d,dropId[%d] <= 0", id, i))
			}
		}
	}
	return nil
}

func (s *ShopCfgLoader) apply() {
	weeklyPass.Store(s.temp4)
}

var weeklyPass atomic.Value

type WeeklyPassCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 持续领取
	DropId []int32 `json:"dropId"`
	// 领取有效期
	ValidityPeriod int32 `json:"validityPeriod"`
	// 服务器有效期
	ServerLimit int32 `json:"serverLimit"`
}

func GetWeeklyPassCfg(id int32) *WeeklyPassCfg {
	cfgMap := weeklyPass.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*WeeklyPassCfg)[id]
}
