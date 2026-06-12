package activityService

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
)

type GameActivityService struct {
	unlockService             logicCommon.UnlockServiceInterface
	gameServerInfoService     *gameServerInfoService.GameServerInfoService
	serverActivityConfigModel *model.ServerActivityConfigModel
	serverOpenActivityModel   *model.ServerOpenActivityModel
	ActivityConfigChanged     atomic.Bool
}

var _ logicCommon.GameActivityServiceInterface = (*GameActivityService)(nil)

func NewGameActivityService(gameServerInfoService *gameServerInfoService.GameServerInfoService, unlockService logicCommon.UnlockServiceInterface) *GameActivityService {
	serverActivityConfigEntities, err := LoadActivityConfigFromRedis()
	if err != nil {
		logger.ErrorBySprintf("[platform] get all activity config from redis error: %v", err)
	}

	activityMap, err := LoadServerOpenActivityFromRedis()
	if err != nil {
		logger.ErrorBySprintf("[platform] load server open activity from redis error: %v", err)
	}

	service := &GameActivityService{
		serverActivityConfigModel: model.NewServerActivityConfigModel(serverActivityConfigEntities),
		serverOpenActivityModel:   model.NewServerOpenActivityModel(activityMap),
		gameServerInfoService:     gameServerInfoService,
		unlockService:             unlockService,
	}
	service.ActivityConfigChanged.Store(false)
	return service
}

func LoadActivityConfigFromRedis() ([]*model.ServerActivityConfigEntity, error) {
	var serverActivityConfigEntities []*model.ServerActivityConfigEntity
	configString, err := dbService.RDB.Get(context.Background(), enum.REDIS_ACTIVITY_ALL_CONFIG).Result()
	if err != nil {
		logger.ErrorBySprintf("[platform] get all activity config from redis error: %v", err)
		return nil, err
	}
	err = json.Unmarshal([]byte(configString), &serverActivityConfigEntities)
	if err != nil {
		logger.ErrorBySprintf("[platform] json activity config unmarshal error: %v", err)
		return nil, err
	}
	return serverActivityConfigEntities, nil
}

func LoadServerOpenActivityFromRedis() (map[int32][]*model.ServerOpenActivityEntity, error) {
	ctx := context.Background()
	serverOpenActivity := make(map[int32][]*model.ServerOpenActivityEntity)

	// 扫描所有 activity_open key
	iter := dbService.RDB.Scan(ctx, 0, enum.REDIS_ACTIVITY_OPEN+"*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val() // activity_open:1
		// 解析 serverId
		serverIdStr := strings.TrimPrefix(key, enum.REDIS_ACTIVITY_OPEN)
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			continue
		}
		// 读取 hash
		result, err := dbService.RDB.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		activityList := make([]*model.ServerOpenActivityEntity, 0, len(result))
		for _, value := range result {
			var entity model.ServerOpenActivityEntity
			err := json.Unmarshal([]byte(value), &entity)
			if err != nil {
				continue
			}
			activityList = append(activityList, &entity)
		}
		serverOpenActivity[int32(serverId)] = activityList
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return serverOpenActivity, nil
}

func (s *GameActivityService) Reload() {
	serverActivityConfigEntities, err := LoadActivityConfigFromRedis()
	if err != nil {
		logger.ErrorBySprintf("[platform] get all activity config from redis error: %v", err)
	} else {
		s.serverActivityConfigModel.ReloadServerActivityConfig(serverActivityConfigEntities)
	}

	activityMap, err := LoadServerOpenActivityFromRedis()
	if err != nil {
		logger.ErrorBySprintf("[platform] load server open activity from redis error: %v", err)
	} else {
		s.serverOpenActivityModel.Reload(activityMap)
	}
}

func (s *GameActivityService) IsActivitySettled(serverId int32, activityId int32, version string) bool {
	return s.serverOpenActivityModel.IsActivitySettled(serverId, activityId, version)
}

func (s *GameActivityService) GetAllOpenActivityByServerId(serverId int32) []logicCommon.GameActivityInterface {
	openAll := make([]logicCommon.GameActivityInterface, 0)
	all := s.serverOpenActivityModel.GetAllOpenActivityByServerId(serverId)
	for _, activity := range all {
		cfg := s.serverActivityConfigModel.GetActivityConfig(activity.GetActivityId())
		if cfg == nil {
			continue
		}
		realConfig, ok := cfg.(*model.ServerActivityConfigEntity)
		if !ok {
			continue
		}
		if isActivityBlocked(realConfig, serverId) {
			continue
		}
		openAll = append(openAll, activity)
	}
	return openAll
}

func (s *GameActivityService) IsActivityOpen(serverId int32, activityId int32) logicCommon.GameActivityInterface {
	cfg := s.serverActivityConfigModel.GetActivityConfig(activityId)
	if cfg == nil {
		return nil
	}
	realConfig, ok := cfg.(*model.ServerActivityConfigEntity)
	if !ok {
		return nil
	}
	if isActivityBlocked(realConfig, serverId) {
		return nil
	}
	return s.serverOpenActivityModel.IsActivityOpen(serverId, activityId)
}

func isActivityBlocked(config *model.ServerActivityConfigEntity, serverId int32) bool {
	if config.IfBlock == 1 {
		return true
	}
	for _, blockedServerId := range config.IfBlockServers {
		if serverId == blockedServerId {
			return true
		}
	}
	return false
}

func (s *GameActivityService) GetActivityConfig(activityId int32) logicCommon.GameActivityConfigInterface {
	activity := s.serverActivityConfigModel.GetActivityConfig(activityId)
	if activity == nil {
		return nil
	}
	return activity
}
