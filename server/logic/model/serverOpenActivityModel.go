package model

import (
	"sync"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type ServerOpenActivityEntity struct {
	ActivityId   int32  `gorm:"column:activity_id;primaryKey"`
	Version      string `gorm:"column:version;primaryKey"`
	OpenServerId int32  `gorm:"column:open_server_id;primaryKey"`
	OpenTime     int64  `gorm:"column:open_time"`
	SettleTime   int64  `gorm:"column:settle_time"`
	EndTime      int64  `gorm:"column:end_time"`

	OpenCount int32 `gorm:"-"`
}

var _ logicCommon.GameActivityInterface = (*ServerOpenActivityEntity)(nil)

func (s *ServerOpenActivityEntity) GetActivityId() int32 {
	return s.ActivityId
}

func (s *ServerOpenActivityEntity) GetVersion() string {
	return s.Version
}

func (s *ServerOpenActivityEntity) GetOpenTime() int64 {
	return s.OpenTime
}

func (s *ServerOpenActivityEntity) GetSettleTime() int64 {
	return s.SettleTime
}

func (s *ServerOpenActivityEntity) GetEndTime() int64 {
	return s.EndTime
}

func (s *ServerOpenActivityEntity) TableName() string {
	return "server_open_activity"
}

func NewServerOpenActivityModel(entityMap map[int32][]*ServerOpenActivityEntity) *ServerOpenActivityModel {
	return &ServerOpenActivityModel{
		entities: entityMap,
	}
}

type ServerOpenActivityModel struct {
	mu       sync.RWMutex
	entities map[int32][]*ServerOpenActivityEntity
}

// 因为有的活动会多开，所以这里返回所有最后一次开启的活动
func (m *ServerOpenActivityModel) GetAllFinalActivity() map[int32]map[int32]*ServerOpenActivityEntity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	finalActivity := make(map[int32]map[int32]*ServerOpenActivityEntity)
	for k, allEntities := range m.entities {
		if finalActivity[k] == nil {
			finalActivity[k] = make(map[int32]*ServerOpenActivityEntity)
		}
		for _, v := range allEntities {
			if finalActivity[k][v.ActivityId] == nil {
				finalActivity[k][v.ActivityId] = v
				v.OpenCount = 1
			} else {
				v.OpenCount = finalActivity[k][v.ActivityId].OpenCount + 1
				if finalActivity[k][v.ActivityId].EndTime < v.EndTime {
					finalActivity[k][v.ActivityId] = v
				}
			}
		}
	}
	return finalActivity
}

func (m *ServerOpenActivityModel) OpenActivity(activities map[int32]map[int32]*ServerOpenActivityEntity) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serverId, serverActivities := range activities {
		serverActivity := m.entities[serverId]
		for _, activityEntity := range serverActivities {
			find := false
			for _, vvv := range serverActivity {
				if vvv.ActivityId == activityEntity.ActivityId {
					if vvv.Version == activityEntity.Version {
						err := easyDB.SaveSeverEntity(vvv)
						if err != nil {
							logger.ErrorBySprintf("save activity error: %v,activity:%+v", err, activityEntity)
							continue
						}
						find = true
						break
					}
				}
			}
			if !find {
				newActivity := &ServerOpenActivityEntity{
					ActivityId:   activityEntity.ActivityId,
					OpenServerId: serverId,
					Version:      activityEntity.Version,
					OpenTime:     activityEntity.OpenTime,
					SettleTime:   activityEntity.SettleTime,
					EndTime:      activityEntity.EndTime,
				}
				m.entities[serverId] = append(m.entities[serverId], newActivity)

				err := easyDB.CreateServerEntity[ServerOpenActivityEntity](newActivity)
				if err != nil {
					logger.ErrorBySprintf("create new activity error: %v,activity:%+v", err, newActivity)
					continue
				}
			}
		}
	}
}

func (m *ServerOpenActivityModel) Reload(entityMap map[int32][]*ServerOpenActivityEntity) {
	m.mu.Lock()
	m.entities = entityMap
	m.mu.Unlock()
}

func (m *ServerOpenActivityModel) IsActivitySettled(serverId int32, activityId int32, version string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := m.entities[serverId]
	if all == nil {
		return true
	}
	for _, v := range all {
		if v.ActivityId == activityId && v.Version == version {
			if tool.UnixNowMilli() >= v.SettleTime {
				return true
			} else if tool.UnixNowMilli() >= v.EndTime {
				return true
			} else {
				return false
			}
		}
	}
	return true
}

func (m *ServerOpenActivityModel) GetAllActivityByServerId(serverId int32) []logicCommon.GameActivityInterface {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := make([]logicCommon.GameActivityInterface, 0)
	for _, v := range m.entities[serverId] {
		all = append(all, v)
	}
	return all
}

func (m *ServerOpenActivityModel) IsActivityOpen(serverId int32, activityId int32) logicCommon.GameActivityInterface {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, v := range m.entities[serverId] {
		if v.ActivityId == activityId {
			if tool.UnixNowMilli() >= v.EndTime {
				continue
			}
			if tool.UnixNowMilli() < v.OpenTime {
				continue
			}
			return v
		}
	}
	return nil
}
