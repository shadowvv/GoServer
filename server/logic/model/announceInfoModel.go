package model

import (
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type AnnounceInfoEntity struct {
	Id           int32  `gorm:"column:id;primaryKey;autoIncrement"`
	AnnounceType int32  `gorm:"column:announce_type"`
	ShowType     int32  `gorm:"column:show_type"`
	Title        string `gorm:"column:title"`
	Content      string `gorm:"column:content"`
	PicAddress   string `gorm:"column:pic_address"`
	ServerIds    string `gorm:"column:server_Id"`
	Unlocks      string `gorm:"column:unlocks"`
	UnlockStop   string `gorm:"column:unlock_stop"`
	StartTime    int64  `gorm:"column:start_time"`
	EndTime      int64  `gorm:"column:end_time"`
	Valid        int32  `gorm:"column:valid"`
	ExtraInfo    string `gorm:"column:extrainfo"`

	ServerIntIds  []int32 `gorm:"-"`
	UnlockIds     []int32 `gorm:"-"`
	UnlockStopIds []int32 `gorm:"-"`
}

func (s *AnnounceInfoEntity) TableName() string {
	return "announce_info"
}

func NewAnnounceInfoModel(entity []*AnnounceInfoEntity) *AnnounceInfoModel {
	model := &AnnounceInfoModel{
		Entity: make(map[int32]*AnnounceInfoEntity),
	}
	for _, v := range entity {
		model.Entity[v.Id] = v
		v.ServerIntIds = gameConfig.ParseIntArray(v.ServerIds)
		v.UnlockIds = gameConfig.ParseIntArray(v.Unlocks)
		v.UnlockStopIds = gameConfig.ParseIntArray(v.UnlockStop)
	}
	return model
}

type AnnounceInfoModel struct {
	mu     sync.RWMutex
	Entity map[int32]*AnnounceInfoEntity
}

func (s *AnnounceInfoModel) GetBlockAnnounceInfo(serverId int32) *AnnounceInfoEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.Entity {
		for _, id := range v.ServerIntIds {
			if id == serverId || id == 0 {
				if v.Valid == 1 && v.StartTime <= tool.UnixNowMilli() && v.EndTime >= tool.UnixNowMilli() && v.AnnounceType == enum.ANNOUNCE_INFO_TYPE_INTERCEPT_ANNOUNCE {
					return v
				}
			}
		}
	}
	return nil
}

func (s *AnnounceInfoModel) GetAllAnnounceInfo(serverId int32) []*AnnounceInfoEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AnnounceInfoEntity, 0)
	for _, v := range s.Entity {
		for _, id := range v.ServerIntIds {
			if id == serverId || id == 0 {
				if v.Valid == 1 && v.StartTime <= tool.UnixNowMilli() && v.EndTime >= tool.UnixNowMilli() {
					result = append(result, v)
				}
			}
		}
	}
	return result
}

func (s *AnnounceInfoModel) createAnnounceInfo(entity *AnnounceInfoEntity) error {
	return easyDB.CreateServerEntity(entity)
}

func (s *AnnounceInfoModel) updateAnnounceInfo(entity *AnnounceInfoEntity) error {
	return easyDB.UpdateServerEntity(entity, map[string]interface{}{
		"server_Id":     entity.ServerIds,
		"unlocks":       entity.Unlocks,
		"unlock_stop":   entity.UnlockStop,
		"start_time":    entity.StartTime,
		"end_time":      entity.EndTime,
		"valid":         entity.Valid,
		"extrainfo":     entity.ExtraInfo,
		"announce_type": entity.AnnounceType,
		"show_type":     entity.ShowType,
		"title":         entity.Title,
		"content":       entity.Content,
		"pic_address":   entity.PicAddress,
	})
}

func (s *AnnounceInfoModel) AddAnnounceInfo(entity *AnnounceInfoEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Entity[entity.Id] = entity
	entity.ServerIntIds = gameConfig.ParseIntArray(entity.ServerIds)
	entity.UnlockIds = gameConfig.ParseIntArray(entity.Unlocks)
	entity.UnlockStopIds = gameConfig.ParseIntArray(entity.UnlockStop)

	return s.createAnnounceInfo(entity)
}

func (s *AnnounceInfoModel) UpdateAnnounceInfo(entity *AnnounceInfoEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Entity[entity.Id] = entity
	entity.ServerIntIds = gameConfig.ParseIntArray(entity.ServerIds)
	entity.UnlockIds = gameConfig.ParseIntArray(entity.Unlocks)
	entity.UnlockStopIds = gameConfig.ParseIntArray(entity.UnlockStop)

	return s.updateAnnounceInfo(entity)
}

func (s *AnnounceInfoModel) ReloadAnnounce(entity []*AnnounceInfoEntity) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.Entity)
	for _, v := range entity {
		s.Entity[v.Id] = v
		v.ServerIntIds = gameConfig.ParseIntArray(v.ServerIds)
		v.UnlockIds = gameConfig.ParseIntArray(v.Unlocks)
		v.UnlockStopIds = gameConfig.ParseIntArray(v.UnlockStop)
	}
}
