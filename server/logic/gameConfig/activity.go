package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
	"strings"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("activity", &ActivityCfgLoader{})
}

type ActivityCfgLoader struct {
	temp1 map[int32]*ActivityCfg
	temp2 map[int32]*ActivityOriginalCfg
}

var _ configLoaderInterface = (*ActivityCfgLoader)(nil)

func (s *ActivityCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/activity.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*ActivityCfg)
	s.temp2 = make(map[int32]*ActivityOriginalCfg)
	for _, row := range rawData["activity"] {
		var v ActivityCfg
		v.Id = ParseInt(row["id"])
		v.ServerType = ParseInt(row["serverType"])
		v.ServerUnit = ParseIntMatrix(row["serverUnit"])
		v.UnlockId = ParseIntArray(row["unlockId"])
		v.UnlockAttendId = ParseIntArray(row["unlockAttendId"])
		v.EventOpen = row["eventOpen"]
		v.EventEnd = row["eventEnd"]
		v.WeekOpen = ParseIntArray(row["weekOpen"])
		v.MonthOpen = ParseIntArray(row["monthOpen"])
		v.Duration = ParseIntArray(row["duration"])
		v.SettleTime = ParseInt(row["settleTime"])
		v.NextId = ParseInt(row["nextId"])
		v.Cd = ParseInt(row["cd"])
		v.IfFirst = ParseInt(row["ifFirst"])
		v.OpenLoopMax = ParseInt(row["openLoopMax"])
		v.IfBlockServer = ParseIntArray(row["ifBlockServer"])
		v.IfBlock = ParseInt(row["ifBlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load activity error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v

		var vv ActivityOriginalCfg
		vv.Id = ParseInt(row["id"])
		vv.ServerType = ParseInt(row["serverType"])
		vv.ServerUnit = row["serverUnit"]
		vv.UnlockId = row["unlockId"]
		vv.UnlockAttendId = row["unlockAttendId"]
		vv.WeekOpen = row["weekOpen"]
		vv.MonthOpen = row["monthOpen"]
		vv.Duration = row["duration"]
		vv.EventOpen = row["eventOpen"]
		vv.EventEnd = row["eventEnd"]
		vv.Cd = ParseInt(row["cd"])
		vv.SettleTime = ParseInt(row["settleTime"])
		vv.NextId = ParseInt(row["nextId"])
		vv.IfFirst = ParseInt(row["ifFirst"])
		vv.OpenLoopMax = ParseInt(row["openLoopMax"])
		vv.IfBlockServer = row["ifBlockServer"]
		vv.IfBlock = ParseInt(row["ifBlock"])
		s.temp2[vv.Id] = &vv
	}

	return nil
}

func (s *ActivityCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid ID:%d", id))
		}
		if !enum.IsValidActivityServerType(v.ServerType) {
			return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid ServerType:%d,configId:%d", v.ServerType, id))
		}
		for _, unlockId := range v.UnlockId {
			unlockConfig := GetUnlockCfg(unlockId)
			if unlockConfig == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid unlockId:%d,configId:%d", unlockId, id))
			}
			if !enum.IsServerUnlock(unlockConfig.GetUnlockType()) {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid unlockId:%d is not server unlock,configId:%d", unlockConfig.GetUnlockType(), id))
			}
		}
		for _, unlockId := range v.UnlockAttendId {
			if GetUnlockCfg(unlockId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid unlockAttendId:%d,configId:%d", unlockId, id))
			}
		}
		if v.EventOpen != "" {
			err := errors.New("")
			count := strings.Count(v.EventOpen, "|")
			if count == 2 {
				_, err = ParseTimeWithYMD(v.EventOpen)
			} else {
				_, err = ParseTime(v.EventOpen)
			}
			if err != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid EventOpen:%s,configId:%d", v.EventOpen, id))
			}
		}
		if v.EventEnd != "" {
			err := errors.New("")
			count := strings.Count(v.EventEnd, "|")
			if count == 2 {
				_, err = ParseTimeWithYMD(v.EventEnd)
			} else {
				_, err = ParseTime(v.EventEnd)
			}
			if err != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid EventEnd:%s,configId:%d", v.EventEnd, id))
			}
		}
		for _, weekOpen := range v.WeekOpen {
			if weekOpen < 1 || weekOpen > 7 {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid WeekOpen:%d,configId:%d", weekOpen, id))
			}
		}
		for _, monthOpen := range v.MonthOpen {
			if monthOpen < 1 || monthOpen > 28 {
				return errors.New(fmt.Sprintf("[gameConfig] load activity error invalid MonthOpen:%d,configId:%d", monthOpen, id))
			}
		}
	}
	return nil
}

func (s *ActivityCfgLoader) apply() {
	activity.Store(s.temp1)
	originalActivity.Store(s.temp2)
}

var activity atomic.Value
var originalActivity atomic.Value

type ActivityCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 跨服匹配类型
	ServerType int32 `json:"serverType"`
	// 跨服范围
	ServerUnit [][]int32 `json:"serverUnit"`
	// 开启条件
	UnlockId []int32 `json:"unlockId"`
	// 参加条件
	UnlockAttendId []int32 `json:"unlockAttendId"`
	// 活动开启时间
	EventOpen string `json:"eventOpen"`
	// 活动结束时间
	EventEnd string `json:"eventEnd"`
	// 周活动开启时间
	WeekOpen []int32 `json:"weekOpen"`
	// 月活动开启时间
	MonthOpen []int32 `json:"monthOpen"`
	// 活动持续时间/h
	Duration []int32 `json:"duration"`
	// 活动结算时间/h
	SettleTime int32 `json:"settleTime"`
	// 下个活动id
	NextId int32 `json:"nextId"`
	// 下个活动间隔时间/h
	Cd int32 `json:"cd"`
	// 是否是首个活动
	IfFirst int32 `json:"ifFirst"`
	// 循环开启上限次数
	OpenLoopMax int32 `json:"openLoopMax"`
	// 是否部分服务器屏蔽
	IfBlockServer []int32 `json:"ifBlockServer"`
	// 是否全屏蔽
	IfBlock int32 `json:"ifBlock"`
}

type ActivityOriginalCfg struct {
	// 序号
	Id int32
	// 跨服匹配类型
	ServerType int32
	// 跨服范围
	ServerUnit string
	// 开启条件
	UnlockId string
	// 参加条件
	UnlockAttendId string
	// 活动开启时间
	EventOpen string
	// 活动结束时间
	EventEnd string
	// 周活动开启时间
	WeekOpen string
	// 月活动开启时间
	MonthOpen string
	// 活动持续时间/h
	Duration string
	// 活动结算时间/h
	SettleTime int32
	// 下个活动id
	NextId int32
	// 下个活动间隔时间/h
	Cd int32
	// 是否是首个活动
	IfFirst int32
	// 循环开启上限次数
	OpenLoopMax int32
	// 是否部分服务器屏蔽
	IfBlockServer string
	// 是否全屏蔽
	IfBlock int32
}

func GetAllOriginalActivityCfg() map[int32]*ActivityOriginalCfg {
	cfgMap := originalActivity.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ActivityOriginalCfg)
}
