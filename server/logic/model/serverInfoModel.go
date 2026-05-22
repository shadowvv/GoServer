package model

import (
	"context"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/tool"
)

type GameServerInfoEntity struct {
	ServerId         int32  `gorm:"column:server_id;primaryKey"`
	ServerName       string `gorm:"column:server_name"`
	ServerNameId     int32  `gorm:"column:server_name_id"`
	ServerOpenTime   int64  `gorm:"column:server_open_time"`
	ServerTime       int64  `gorm:"column:server_time"`
	ServerLogicId    int32  `gorm:"column:server_logic_id"`
	AreaId           int32  `gorm:"column:area_id"`
	AreaName         string `gorm:"column:area_name"`
	RegisterCount    int32  `gorm:"column:register_count"`
	MaxRegisterCount int32  `gorm:"column:max_register_count"`
	OpenToNewWeight  int32  `gorm:"column:open_to_new_weight"`
	OpenToNew        int32  `gorm:"column:open_to_new"`
	CanSeeGroupId    int32  `gorm:"column:can_see_group_id"`
	Status           int32  `gorm:"column:status"`
}

var _ logicCommon.ServerInfoInterface = (*GameServerInfoEntity)(nil)

func (s *GameServerInfoEntity) GetServerId() int32 {
	return s.ServerId
}

func (s *GameServerInfoEntity) GetServerOpenTime() int64 {
	return s.ServerOpenTime
}

func (s *GameServerInfoEntity) GetServerTime() int64 {
	return s.ServerTime
}

func (s *GameServerInfoEntity) GetRegisterCount() int32 {
	return s.RegisterCount
}

func (s *GameServerInfoEntity) GetActivePlayerCount() int32 {
	count, err := dbService.RDB.Get(context.Background(), enum.GetOnlinePlayerKey(s.ServerId)).Result()
	if err != nil {
		return 0
	}
	return gameConfig.ParseInt(count)
}

func (u *GameServerInfoEntity) TableName() string {
	return "server_info"
}

type GameServerInfoModel struct {
	mu           sync.RWMutex
	entities     map[int32]*GameServerInfoEntity
	MaxNewWeight int32
}

func (s *GameServerInfoModel) GetGameServerInfo(serverId int32) *GameServerInfoEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.entities[serverId]
}

func (s *GameServerInfoModel) GetAllServerInfo() map[int32]*GameServerInfoEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int32]*GameServerInfoEntity)
	for k, v := range s.entities {
		result[k] = v
	}
	return result
}

func (s *GameServerInfoModel) GetAllOpenServerInfo() map[int32]*GameServerInfoEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int32]*GameServerInfoEntity)
	for k, v := range s.entities {
		if v.Status != int32(enum.GAME_SERVER_STATUS_ONLINE) {
			continue
		}
		if v.ServerOpenTime > tool.UnixNowMilli() {
			continue
		}
		result[k] = v
	}
	return result
}

func (s *GameServerInfoModel) GetDefaultServerId() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	firstPriorityIds := make([]int32, 0)
	secondPriorityIds := make([]int32, 0)
	thirdPriorityIds := make([]int32, 0)

	var secondMaxWeight int32 = -1
	var thirdMaxWeight int32 = -1

	for _, v := range s.entities {
		serverRegisterCountKey := enum.GetRegisterConst(v.ServerId)
		serverRegisterCount, err := dbService.RDB.Get(context.Background(), serverRegisterCountKey).Int64()
		if err != nil {
			serverRegisterCount = 0
		}

		if v.ServerOpenTime > tool.UnixNowMilli() {
			continue
		}

		// 基础条件：OpenToNew开启且状态在线
		if v.OpenToNew != 1 || v.Status != int32(enum.GAME_SERVER_STATUS_ONLINE) {
			continue
		}

		// 第三优先级：只要基础条件满足就加入（即使注册量已满）
		thirdPriorityIds = append(thirdPriorityIds, v.ServerId)
		if v.OpenToNewWeight > thirdMaxWeight {
			thirdMaxWeight = v.OpenToNewWeight
		}

		// 第二优先级：注册量未满
		if serverRegisterCount+1 <= int64(v.MaxRegisterCount) {
			secondPriorityIds = append(secondPriorityIds, v.ServerId)
			if v.OpenToNewWeight > secondMaxWeight {
				secondMaxWeight = v.OpenToNewWeight
			}

			// 第一优先级：OpenToNewWeight最大且注册量未满
			if v.OpenToNewWeight == s.MaxNewWeight {
				firstPriorityIds = append(firstPriorityIds, v.ServerId)
			}
		}
	}

	// 第一优先级
	if len(firstPriorityIds) > 0 {
		return firstPriorityIds[tool.RandInt(0, len(firstPriorityIds)-1)]
	}

	// 第二优先级：返回权重最大的服务器
	if len(secondPriorityIds) > 0 {
		maxWeightIds := make([]int32, 0)
		for _, id := range secondPriorityIds {
			// 需要从 s.entities 中找到对应的权重
			for _, v := range s.entities {
				if v.ServerId == id && v.OpenToNewWeight == secondMaxWeight {
					maxWeightIds = append(maxWeightIds, id)
					break
				}
			}
		}
		return maxWeightIds[tool.RandInt(0, len(maxWeightIds)-1)]
	}

	// 第三优先级：返回权重最大的服务器（即使注册量已满）
	if len(thirdPriorityIds) > 0 {
		maxWeightIds := make([]int32, 0)
		for _, id := range thirdPriorityIds {
			for _, v := range s.entities {
				if v.ServerId == id && v.OpenToNewWeight == thirdMaxWeight {
					maxWeightIds = append(maxWeightIds, id)
					break
				}
			}
		}
		return maxWeightIds[tool.RandInt(0, len(maxWeightIds)-1)]
	}

	return 0
}

func NewGameServerInfoModel(entity []*GameServerInfoEntity) *GameServerInfoModel {
	serverInfoModel := &GameServerInfoModel{
		entities:     make(map[int32]*GameServerInfoEntity),
		MaxNewWeight: 0,
	}
	for _, v := range entity {
		serverInfoModel.entities[v.ServerId] = v
		registerCountKey := enum.GetRegisterConst(v.ServerId)
		userInfo, err := easyDB.GetPlayerEntitiesByWhere[UserEntity](map[string]interface{}{"server_id": v.ServerId})
		if err == nil {
			registerCount := len(userInfo)
			dbService.RDB.Set(context.Background(), registerCountKey, registerCount, 0)
		}
		if v.Status == int32(enum.GAME_SERVER_STATUS_ONLINE) && serverInfoModel.MaxNewWeight < v.OpenToNewWeight {
			serverInfoModel.MaxNewWeight = v.OpenToNewWeight
		}
	}
	return serverInfoModel
}
func (s *GameServerInfoModel) createServerInfo(entity *GameServerInfoEntity) error {
	return easyDB.CreateServerEntity(entity)
}

func (s *GameServerInfoModel) AddServerInfo(entity *GameServerInfoEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := easyDB.CreateServerEntity(entity)
	if err != nil {
		return err
	}
	s.entities[entity.ServerId] = entity
	if entity.Status == int32(enum.GAME_SERVER_STATUS_ONLINE) && s.MaxNewWeight < entity.OpenToNewWeight {
		s.MaxNewWeight = entity.OpenToNewWeight
	}
	return nil
}

func (s *GameServerInfoModel) updateServerInfo(entity *GameServerInfoEntity) error {
	return easyDB.UpdateServerEntity(entity, map[string]interface{}{
		"server_name":        entity.ServerName,
		"server_open_time":   entity.ServerOpenTime,
		"server_time":        entity.ServerTime,
		"server_logic_id":    entity.ServerLogicId,
		"area_id":            entity.AreaId,
		"area_name":          entity.AreaName,
		"max_register_count": entity.MaxRegisterCount,
		"open_to_new_weight": entity.OpenToNewWeight,
		"open_to_new":        entity.OpenToNew,
		"can_see_group_id":   entity.CanSeeGroupId,
		"status":             entity.Status,
	})
}

func (s *GameServerInfoModel) UpdateServerInfo(entity *GameServerInfoEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entities[entity.ServerId] = entity
	if entity.Status == int32(enum.GAME_SERVER_STATUS_ONLINE) && s.MaxNewWeight < entity.OpenToNewWeight {
		s.MaxNewWeight = entity.OpenToNewWeight
	} else if entity.Status != int32(enum.GAME_SERVER_STATUS_ONLINE) {
		s.MaxNewWeight = 0
		for _, v := range s.entities {
			if v.Status == int32(enum.GAME_SERVER_STATUS_ONLINE) && v.OpenToNewWeight > s.MaxNewWeight {
				s.MaxNewWeight = v.OpenToNewWeight
			}
		}
	}
	return s.updateServerInfo(entity)
}

func (s *GameServerInfoModel) ReloadServerInfo(entity []*GameServerInfoEntity) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range entity {
		s.entities[v.ServerId] = v
		if v.Status == int32(enum.GAME_SERVER_STATUS_ONLINE) && s.MaxNewWeight < v.OpenToNewWeight {
			s.MaxNewWeight = v.OpenToNewWeight
		}
	}
}

func (s *GameServerInfoModel) ResetOnlinePlayerNum() {
	for _, v := range s.entities {
		_ = dbService.RDB.SetEX(context.Background(), enum.GetOnlinePlayerKey(v.ServerId), 0, 0)
	}
}
