package model

import (
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/logic/logicCommon"

	"github.com/drop/GoServer/server/logic/gameConfig"
)

type ServerActivityConfigEntity struct {
	Id             int32  `gorm:"column:id;primaryKey"`
	ServerType     int32  `gorm:"column:server_type"`
	ServerUnit     string `gorm:"column:server_unit"`
	UnlockId       string `gorm:"column:unlock_id"`
	AttendUnlockId string `gorm:"column:attend_unlock_id"`
	EventOpen      string `gorm:"column:event_open"`
	EventEnd       string `gorm:"column:event_end"`
	WeekOpen       string `gorm:"column:week_open"`
	MonthOpen      string `gorm:"column:month_open"`
	Duration       string `gorm:"column:duration"`
	SettleTime     int32  `gorm:"column:settle_time"`
	IfFirst        int32  `gorm:"column:if_first"`
	NextId         int32  `gorm:"column:next_id"`
	Cd             int32  `gorm:"column:cd"`
	OpenLoopNum    int32  `gorm:"column:open_loop_num"`
	IfBlockServer  string `gorm:"column:if_block_server"`
	IfBlock        int32  `gorm:"column:if_block"`

	ServerUnitInfo  *ServerUnitData `gorm:"-"`
	UnlockIds       []int32         `gorm:"-"`
	AttendUnlockIds []int32         `gorm:"-"`
	EventOpenTime   int64           `gorm:"-"`
	EventEndTime    int64           `gorm:"-"`
	WeekOpenDays    []int32         `gorm:"-"`
	MonthOpenDays   []int32         `gorm:"-"`
	DurationTimes   []int32         `gorm:"-"`
	IfBlockServers  []int32         `gorm:"-"`
	HasPreActivity  bool            `gorm:"-"`
}

var _ logicCommon.GameActivityConfigInterface = (*ServerActivityConfigEntity)(nil)

func (s *ServerActivityConfigEntity) GetActivityId() int32 {
	return s.Id
}

func (s *ServerActivityConfigEntity) GetAttendUnlockId() []int32 {
	return s.AttendUnlockIds
}

func (s *ServerActivityConfigEntity) buildData() error {
	s.ServerUnitInfo = &ServerUnitData{
		AllServer:        false,
		ServerUnitVector: make([]*ServerUnitVector, 0),
	}
	if s.ServerUnit == "" {
		s.ServerUnitInfo.AllServer = true
	} else {
		units := strings.Split(s.ServerUnit, ";")
		for _, u := range units {
			vector := &ServerUnitVector{
				Units: make([]int32, 0),
			}
			elements := strings.Split(u, "|")
			for _, e := range elements {
				temp := strings.Split(e, "~")
				if len(temp) == 2 {
					vector.Left = gameConfig.ParseInt(temp[0])
					vector.Right = gameConfig.ParseInt(temp[1])
				} else if len(temp) == 1 {
					vector.Units = append(vector.Units, gameConfig.ParseInt(temp[0]))
				}
			}
			s.ServerUnitInfo.ServerUnitVector = append(s.ServerUnitInfo.ServerUnitVector, vector)
		}
	}

	s.UnlockIds = gameConfig.ParseIntArray(s.UnlockId)
	s.AttendUnlockIds = gameConfig.ParseIntArray(s.AttendUnlockId)
	if s.EventOpen == "" {
		s.EventOpenTime = 0
	} else {
		ot := int64(0)
		var err error = nil
		count := strings.Count(s.EventOpen, "|")
		if count == 2 {
			ot, err = gameConfig.ParseTimeWithYMD(s.EventOpen)
			if err != nil {
				return err
			}
		} else {
			ot, err = gameConfig.ParseTime(s.EventOpen)
			if err != nil {
				return err
			}
		}
		s.EventOpenTime = ot
	}
	if s.EventEnd == "" {
		s.EventEndTime = 0
	} else {
		et := int64(0)
		var err error = nil
		count := strings.Count(s.EventEnd, "|")
		if count == 2 {
			et, err = gameConfig.ParseTimeWithYMD(s.EventEnd)
			if err != nil {
				return err
			}
		} else {
			et, err = gameConfig.ParseTime(s.EventEnd)
			if err != nil {
				return err
			}
		}
		s.EventEndTime = et
	}
	s.WeekOpenDays = gameConfig.ParseIntArray(s.WeekOpen)
	s.MonthOpenDays = gameConfig.ParseIntArray(s.MonthOpen)
	s.DurationTimes = gameConfig.ParseIntArray(s.Duration)
	s.IfBlockServers = gameConfig.ParseIntArray(s.IfBlockServer)
	s.HasPreActivity = false
	return nil
}

func (s *ServerActivityConfigEntity) TableName() string {
	return "server_activity_config"
}

type ServerUnitData struct {
	AllServer        bool
	ServerUnitVector []*ServerUnitVector
}

type ServerUnitVector struct {
	Left  int32
	Right int32
	Units []int32
}

func NewServerActivityConfigModel(entity []*ServerActivityConfigEntity) *ServerActivityConfigModel {
	infoMap := make(map[int32]*ServerActivityConfigEntity)
	for _, e := range entity {
		err := e.buildData()
		if err != nil {
			panic(err)
		}
		infoMap[e.Id] = e
	}
	for _, e := range infoMap {
		if e.NextId != 0 {
			info := infoMap[e.NextId]
			if info != nil && info.IfFirst != 1 {
				info.HasPreActivity = true
			}
		}
	}
	model := &ServerActivityConfigModel{}
	model.configValues.Store(infoMap)
	return model
}

type ServerActivityConfigModel struct {
	configValues atomic.Value // map[int32]*ServerActivityConfigEntity
}

func (m *ServerActivityConfigModel) GetAllServerActivityConfig() map[int32]*ServerActivityConfigEntity {
	cfgRaw := m.configValues.Load()
	if cfgRaw == nil {
		return nil
	}
	return cfgRaw.(map[int32]*ServerActivityConfigEntity)
}

func (m *ServerActivityConfigModel) GetActivityConfig(id int32) logicCommon.GameActivityConfigInterface {
	cfg := m.configValues.Load().(map[int32]*ServerActivityConfigEntity)
	if cfg == nil {
		return nil
	}
	if cfg[id] == nil {
		return nil
	}
	return cfg[id]
}

func (m *ServerActivityConfigModel) ReloadServerActivityConfig(entity []*ServerActivityConfigEntity) {
	infoMap := make(map[int32]*ServerActivityConfigEntity)
	for _, e := range entity {
		err := e.buildData()
		if err != nil {
			panic(err)
		}
		infoMap[e.Id] = e
	}
	for _, e := range infoMap {
		if e.NextId != 0 {
			info := infoMap[e.NextId]
			if info != nil && info.IfFirst != 1 {
				info.HasPreActivity = true
			}
		}
	}

	m.configValues.Store(infoMap)
}
