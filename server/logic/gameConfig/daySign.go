package gameConfig

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("daySign", &DaySignCfgLoader{})
}

type DaySignCfgLoader struct {
	cfgByID        map[int32]*DaySignCfg
	signIDsByActID map[int32][]int32
}

var _ configLoaderInterface = (*DaySignCfgLoader)(nil)

func (s *DaySignCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/daySign.json`, &rawData); err != nil {
		return err
	}

	s.cfgByID = make(map[int32]*DaySignCfg)
	s.signIDsByActID = make(map[int32][]int32)

	// 兼容：历史配置 key "sevenDaysSign" 已更名为 "daySign"。
	rows := rawData["daySign"]
	if rows == nil {
		rows = rawData["sevenDaysSign"]
	}
	for _, row := range rows {
		cfg := &DaySignCfg{
			Id:        ParseInt(row["id"]),
			ActID:     ParseInt(row["actID"]),
			Loop:      ParseInt(row["loop"]),
			Permanent: ParseInt(row["permanent"]),
			Duration:  ParseInt(row["duration"]),
			Case:      ParseInt(row["case"]),
			Sort:      ParseInt(row["sort"]),
			DropID:    ParseIntArray(row["dropID"]),
		}
		if cfg.Id <= 0 {
			continue
		}
		if s.cfgByID[cfg.Id] != nil {
			return fmt.Errorf("[gameConfig] load daySign error duplicate ID:%d", cfg.Id)
		}
		s.cfgByID[cfg.Id] = cfg
		s.signIDsByActID[cfg.ActID] = append(s.signIDsByActID[cfg.ActID], cfg.Id)
	}

	for actID, signIDs := range s.signIDsByActID {
		sort.Slice(signIDs, func(i, j int) bool { return signIDs[i] < signIDs[j] })
		s.signIDsByActID[actID] = signIDs
	}
	return nil
}

func (s *DaySignCfgLoader) checkData() error {
	allActivity := GetAllOriginalActivityCfg()
	for id, cfg := range s.cfgByID {
		if cfg.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error invalid ID:%d", id))
		}
		if cfg.ActID <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error invalid ActID:%d,configId:%d", cfg.ActID, id))
		}
		if allActivity == nil || allActivity[cfg.ActID] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error activity not found actID:%d,configId:%d", cfg.ActID, id))
		}
		if cfg.Duration < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error invalid duration:%d,configId:%d", cfg.Duration, id))
		}
		if cfg.Case <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error invalid case:%d,configId:%d", cfg.Case, id))
		}
		if len(cfg.DropID) == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error empty dropID,configId:%d", id))
		}
		if int32(len(cfg.DropID)) != cfg.Duration && cfg.Duration != 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load daySign error dropID size mismatch duration:%d,dropCount:%d,configId:%d", cfg.Duration, len(cfg.DropID), id))
		}
		for _, dropID := range cfg.DropID {
			if dropID <= 0 || GetDropCfg(dropID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load daySign error invalid dropID:%d,configId:%d", dropID, id))
			}
		}
	}
	return nil
}

func (s *DaySignCfgLoader) apply() {
	daySign.Store(s.cfgByID)
	daySignByAct.Store(s.signIDsByActID)
}

var daySign atomic.Value
var daySignByAct atomic.Value

type DaySignCfg struct {
	// 配置ID
	Id int32 `json:"id"`
	// 活动ID
	ActID int32 `json:"actID"`
	// 循环标记（1=循环）
	Loop int32 `json:"loop"`
	// 永久标记（1=永久）
	Permanent int32 `json:"permanent"`
	// 周期时长（自然日）
	Duration int32 `json:"duration"`
	// 覆盖分组，同一组内按 sort/id 互相覆盖，组间互不影响
	Case int32 `json:"case"`
	// 优先级排序键（越大优先级越高）
	Sort int32 `json:"sort"`
	// 每日掉落列表（按天序）
	DropID []int32 `json:"dropID"`
}

func GetDaySignCfg(id int32) *DaySignCfg {
	cfgMap := daySign.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DaySignCfg)[id]
}

func GetAllDaySignCfg() map[int32]*DaySignCfg {
	cfgMap := daySign.Load()
	if cfgMap == nil {
		return map[int32]*DaySignCfg{}
	}
	return cfgMap.(map[int32]*DaySignCfg)
}

func GetDaySignIdsByActID(actID int32) []int32 {
	index := daySignByAct.Load()
	if index == nil {
		return nil
	}
	signIDs := index.(map[int32][]int32)[actID]
	if len(signIDs) == 0 {
		return nil
	}
	// 返回副本，避免调用方误修改配置缓存切片。
	res := make([]int32, len(signIDs))
	copy(res, signIDs)
	return res
}
