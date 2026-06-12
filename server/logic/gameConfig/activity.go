package gameConfig

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
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
		original := s.temp2[id]
		err := CheckActivityConfig(&ActivityConfigCheckData{
			Id:              v.Id,
			ServerType:      v.ServerType,
			ServerUnit:      original.ServerUnit,
			UnlockIds:       v.UnlockId,
			AttendUnlockIds: v.UnlockAttendId,
			EventOpen:       v.EventOpen,
			EventEnd:        v.EventEnd,
			WeekOpenDays:    v.WeekOpen,
			MonthOpenDays:   v.MonthOpen,
			Duration:        original.Duration,
			NextId:          v.NextId,
			OpenLoopMax:     v.OpenLoopMax,
		})
		if err != nil {
			return fmt.Errorf("[gameConfig] load activity error %w", err)
		}
	}
	return nil
}

func checkActivityUnlock(unlockId int32) (bool, bool) {
	unlockConfig := GetUnlockCfg(unlockId)
	if unlockConfig == nil {
		return false, false
	}
	return true, enum.IsServerUnlock(unlockConfig.GetUnlockType())
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

type ActivityConfigCheckData struct {
	Id              int32
	ServerType      int32
	ServerUnit      string
	UnlockIds       []int32
	AttendUnlockIds []int32
	EventOpen       string
	EventEnd        string
	WeekOpenDays    []int32
	MonthOpenDays   []int32
	Duration        string
	NextId          int32
	OpenLoopMax     int32
}

func checkActivityTime(value string) error {
	if value == "" {
		return nil
	}
	if strings.Count(value, "|") == 2 {
		_, err := ParseTimeWithYMD(value)
		return err
	}
	_, err := ParseTime(value)
	return err
}

func CheckActivityConfig(data *ActivityConfigCheckData) error {
	if data.Id <= 0 {
		return fmt.Errorf("invalid ID:%d", data.Id)
	}
	if !enum.IsValidActivityServerType(data.ServerType) {
		return fmt.Errorf("invalid ServerType:%d,configId:%d", data.ServerType, data.Id)
	}
	if data.ServerType == int32(enum.ActivityServerType_Multi) && data.ServerUnit == "" {
		return fmt.Errorf("empty ServerUnit for multi-server activity,configId:%d", data.Id)
	}
	for _, unlockId := range data.UnlockIds {
		unlockCfg := GetUnlockCfg(unlockId)
		if unlockCfg == nil {
			return fmt.Errorf("invalid unlockId:%d,configId:%d", unlockId, data.Id)
		}
		if !enum.IsServerUnlock(unlockCfg.GetUnlockType()) {
			return fmt.Errorf("invalid unlockId:%d is not server unlock,configId:%d", unlockId, data.Id)
		}
	}
	for _, unlockId := range data.AttendUnlockIds {
		if GetUnlockCfg(unlockId) == nil {
			return fmt.Errorf("invalid unlockAttendId:%d,configId:%d", unlockId, data.Id)
		}
	}
	if err := checkActivityTime(data.EventOpen); err != nil {
		return fmt.Errorf("invalid EventOpen:%s,configId:%d", data.EventOpen, data.Id)
	}
	if err := checkActivityTime(data.EventEnd); err != nil {
		return fmt.Errorf("invalid EventEnd:%s,configId:%d", data.EventEnd, data.Id)
	}
	for _, weekOpen := range data.WeekOpenDays {
		if weekOpen < 1 || weekOpen > 7 {
			return fmt.Errorf("invalid WeekOpen:%d,configId:%d", weekOpen, data.Id)
		}
	}
	for _, monthOpen := range data.MonthOpenDays {
		if monthOpen < 1 || monthOpen > 28 {
			return fmt.Errorf("invalid MonthOpen:%d,configId:%d", monthOpen, data.Id)
		}
	}
	if data.OpenLoopMax < -1 {
		return fmt.Errorf("invalid OpenLoopMax:%d,configId:%d", data.OpenLoopMax, data.Id)
	}
	if data.NextId != 0 {
		if strings.TrimSpace(data.Duration) == "" {
			return fmt.Errorf("empty Duration for activity with NextId:%d,configId:%d", data.NextId, data.Id)
		}
		if data.OpenLoopMax == 0 {
			return fmt.Errorf("invalid OpenLoopMax:0 for activity with NextId:%d,configId:%d", data.NextId, data.Id)
		}
	}
	return nil
}
