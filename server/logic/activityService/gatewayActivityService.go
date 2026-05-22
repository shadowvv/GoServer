package activityService

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

var TICK_INTERVAL = 5 * time.Second

type GatewayActivityService struct {
	env                       enum.Environment
	unlockService             logicCommon.UnlockServiceInterface
	gameServerInfoService     *gameServerInfoService.GameServerInfoService
	serverActivityConfigModel *model.ServerActivityConfigModel
	serverOpenActivityModel   *model.ServerOpenActivityModel
	activityConfigChanged     atomic.Bool
}

var _ logicCommon.GameActivityServiceInterface = (*GatewayActivityService)(nil)

func NewGatewayActivityService(env enum.Environment, gameServerInfoService *gameServerInfoService.GameServerInfoService, unlockService logicCommon.UnlockServiceInterface) *GatewayActivityService {

	serverActivityConfigEntities, err := loadActivityConfig(env)
	if err != nil {
		panic(fmt.Sprintf("[platform] get all activity error: %v", err))
	}
	serverOpenActivity, err := loadOpenActivity()
	if err != nil {
		panic(fmt.Sprintf("[platform] get all open activity error: %v", err))
	}

	service := &GatewayActivityService{
		serverActivityConfigModel: model.NewServerActivityConfigModel(serverActivityConfigEntities),
		serverOpenActivityModel:   model.NewServerOpenActivityModel(serverOpenActivity),
		gameServerInfoService:     gameServerInfoService,
		unlockService:             unlockService,
		env:                       env,
	}
	service.activityConfigChanged.Store(false)
	return service
}

func loadActivityConfig(env enum.Environment) ([]*model.ServerActivityConfigEntity, error) {
	var serverActivityConfigEntities []*model.ServerActivityConfigEntity
	if env != enum.ENV_LOCAL && env != enum.ENV_DEVELOP && env != enum.ENV_TEST {
		var err error
		serverActivityConfigEntities, err = easyDB.GetServerAllEntities[model.ServerActivityConfigEntity]()
		if err != nil {
			logger.ErrorBySprintf("[platform] get all activity error: %v", err)
			return nil, err
		}
	} else {
		gameAllActivity := gameConfig.GetAllOriginalActivityCfg()
		for _, cfg := range gameAllActivity {
			activityConfig := &model.ServerActivityConfigEntity{
				Id:             cfg.Id,
				ServerType:     cfg.ServerType,
				ServerUnit:     cfg.ServerUnit,
				UnlockId:       cfg.UnlockId,
				AttendUnlockId: cfg.UnlockAttendId,
				EventOpen:      cfg.EventOpen,
				EventEnd:       cfg.EventEnd,
				WeekOpen:       cfg.WeekOpen,
				MonthOpen:      cfg.MonthOpen,
				Duration:       cfg.Duration,
				SettleTime:     cfg.SettleTime,
				IfFirst:        cfg.IfFirst,
				NextId:         cfg.NextId,
				Cd:             cfg.Cd,
				OpenLoopNum:    cfg.OpenLoopMax,
				IfBlockServer:  cfg.IfBlockServer,
				IfBlock:        cfg.IfBlock,

				ServerUnitInfo: &model.ServerUnitData{
					AllServer:        false,
					ServerUnitVector: make([]*model.ServerUnitVector, 0),
				},
				UnlockIds:       make([]int32, 0),
				AttendUnlockIds: make([]int32, 0),
				EventOpenTime:   int64(0),
				EventEndTime:    int64(0),
				WeekOpenDays:    make([]int32, 0),
				MonthOpenDays:   make([]int32, 0),
				DurationTimes:   make([]int32, 0),
				IfBlockServers:  make([]int32, 0),
				LoopActivity:    false,
			}
			serverActivityConfigEntities = append(serverActivityConfigEntities, activityConfig)
			err := easyDB.SaveSeverEntity(activityConfig)
			if err != nil {
				logger.ErrorBySprintf("[platform] save activity error: %v,activity:%v", err, activityConfig)
				return nil, err
			}
		}
	}

	// 保存所有活动配置到redis
	allConfigConfigString, err := json.Marshal(serverActivityConfigEntities)
	if err != nil {
		logger.ErrorBySprintf("[platform] activity config json marshal error: %v", err)
		return nil, err
	}
	err = dbService.RDB.Set(context.Background(), enum.REDIS_ACTIVITY_ALL_CONFIG, string(allConfigConfigString), 0).Err()
	if err != nil {
		logger.ErrorBySprintf("[platform] activity config set redis error: %v", err)
		return nil, err
	}

	return serverActivityConfigEntities, nil
}

func loadOpenActivity() (map[int32][]*model.ServerOpenActivityEntity, error) {
	openActivityEntities, err := easyDB.GetServerAllEntities[model.ServerOpenActivityEntity]()
	if err != nil {
		logger.ErrorBySprintf("[platform] get all open activity error: %v", err)
		return nil, err
	}
	serverOpenActivity := make(map[int32][]*model.ServerOpenActivityEntity)
	for _, entity := range openActivityEntities {
		if serverOpenActivity[entity.OpenServerId] == nil {
			serverOpenActivity[entity.OpenServerId] = make([]*model.ServerOpenActivityEntity, 0)
		}
		serverOpenActivity[entity.OpenServerId] = append(serverOpenActivity[entity.OpenServerId], entity)
	}
	return serverOpenActivity, nil
}

func (s *GatewayActivityService) Reload() {
	s.activityConfigChanged.Store(true)
}

func (s *GatewayActivityService) StartService() {
	s.initAllActivity()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorBySprintf("[platform] gateway activity service heartbeat panic: %v", r)
			}
		}()

		ticker := time.NewTicker(TICK_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				configChanged := false
				if s.activityConfigChanged.Load() {
					serverActivityConfigEntities, err := loadActivityConfig(s.env)
					if err != nil {
						logger.ErrorBySprintf("[platform] load all activity config error: %v", err)
						continue
					}
					s.serverActivityConfigModel.ReloadServerActivityConfig(serverActivityConfigEntities)
					s.activityConfigChanged.Store(false)
					configChanged = true
				}
				changeMap := s.checkActivityChange(configChanged)
				if len(changeMap) > 0 {
					s.saveActivityToRedis()
					rpcController.BroadcastOperationToGameNode(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY)
				}
			}
		}
	}()
}

func (s *GatewayActivityService) checkActivityChange(configChanged bool) map[int32]map[int32]*model.ServerOpenActivityEntity {
	allActivityConfig := s.serverActivityConfigModel.GetAllServerActivityConfig()
	allServer := s.gameServerInfoService.GetAllOpenServerInfo()
	allActivity := s.serverOpenActivityModel.GetAllFinalActivity()
	changedMap := s.refreshAllActivity(allActivityConfig, allServer, allActivity, configChanged)

	if len(changedMap) > 0 {
		s.serverOpenActivityModel.OpenActivity(changedMap)
	}
	return changedMap
}

func (s *GatewayActivityService) saveActivityToRedis() {
	pip := dbService.RDB.Pipeline()
	ctx := context.Background()
	for serverId, openActivity := range s.serverOpenActivityModel.GetAllFinalActivity() {
		key := enum.GetActivityOpenKey(serverId)
		fields := make(map[string]interface{})
		for _, entity := range openActivity {
			data, err := json.Marshal(entity)
			if err != nil {
				logger.ErrorBySprintf("[platform] json marshal serverOpenActivity error: %v,activity:%v", err, entity)
				continue
			}
			fields[strconv.Itoa(int(entity.ActivityId))] = data
		}
		if len(fields) > 0 {
			pip.HSet(ctx, key, fields)
		}
	}
	_, err := pip.Exec(ctx)
	if err != nil {
		logger.ErrorBySprintf("[platform] set serverOpenActivity redis error: %v", err)
	}
}

func (s *GatewayActivityService) initAllActivity() {
	s.checkActivityChange(true)
	s.saveActivityToRedis()
	rpcController.BroadcastOperationToGameNode(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY)
}

func (s *GatewayActivityService) refreshAllActivity(configs map[int32]*model.ServerActivityConfigEntity, servers map[int32]*model.GameServerInfoEntity, activities map[int32]map[int32]*model.ServerOpenActivityEntity, configChanged bool) map[int32]map[int32]*model.ServerOpenActivityEntity {
	changedMap := make(map[int32]map[int32]*model.ServerOpenActivityEntity)
	currentTime := tool.UnixNowMilli()

	// 检测所有已经开过的活动有没有下一个活动
	for serverId, openActivity := range activities {
		for _, activity := range openActivity {
			cfg := s.serverActivityConfigModel.GetActivityConfig(activity.ActivityId)
			if cfg == nil {
				continue
			}
			realConfig, ok := cfg.(*model.ServerActivityConfigEntity)
			if !ok {
				continue
			}
			// 根据配置检测活动更改
			if configChanged && s.checkConfigChange(realConfig, activity, currentTime) {
				if changedMap[serverId] == nil {
					changedMap[serverId] = make(map[int32]*model.ServerOpenActivityEntity)
				}
				changedMap[serverId][activity.ActivityId] = activity
			}

			if realConfig.NextId == 0 {
				continue
			}
			if currentTime < activity.EndTime {
				continue
			}
			if realConfig.Cd > 0 && currentTime-activity.EndTime < int64(realConfig.Cd)*tool.HOUR_MILLI {
				continue
			}
			openInfo, changed := s.checkActivityOpen(realConfig, servers[serverId], activity.OpenCount, currentTime, false)
			if !changed {
				continue
			}
			if changedMap[serverId] == nil {
				changedMap[serverId] = make(map[int32]*model.ServerOpenActivityEntity)
			}
			changedMap[serverId][activity.ActivityId] = openInfo
		}
	}

	// 检测所有活动是否开启
	for _, cfg := range configs {
		for serverId, server := range servers {
			serverActivities, ok := activities[serverId]
			if ok {
				activity, ok := serverActivities[cfg.Id]
				if !ok {
					// 新增活动
					openInfo, changed := s.checkActivityOpen(cfg, server, 0, currentTime, true)
					if !changed {
						continue
					}
					if changedMap[serverId] == nil {
						changedMap[serverId] = make(map[int32]*model.ServerOpenActivityEntity)
					}
					changedMap[serverId][cfg.Id] = openInfo
				} else {
					// 检测活动是否结束
					if currentTime < activity.EndTime {
						continue
					}
					openInfo, changed := s.checkActivityOpen(cfg, server, activity.OpenCount, currentTime, true)
					if !changed {
						continue
					}
					if changedMap[serverId] == nil {
						changedMap[serverId] = make(map[int32]*model.ServerOpenActivityEntity)
					}
					changedMap[serverId][cfg.Id] = openInfo
				}
			} else {
				// 新增活动
				openInfo, changed := s.checkActivityOpen(cfg, server, 0, currentTime, true)
				if !changed {
					continue
				}
				if changedMap[serverId] == nil {
					changedMap[serverId] = make(map[int32]*model.ServerOpenActivityEntity)
				}
				changedMap[serverId][cfg.Id] = openInfo
			}
		}
	}
	return changedMap
}
func (s *GatewayActivityService) checkActivityOpen(activityConfig *model.ServerActivityConfigEntity, server *model.GameServerInfoEntity, openCount int32, currentTime int64, checkOpenCondition bool) (*model.ServerOpenActivityEntity, bool) {
	// 检测屏蔽
	if activityConfig.IfBlock == 1 {
		return nil, false
	}
	for _, serverId := range activityConfig.IfBlockServers {
		if serverId == server.ServerId {
			return nil, false
		}
	}
	if checkOpenCondition {
		// 检测解锁
		for _, unlockId := range activityConfig.UnlockIds {
			if !s.unlockService.CheckServerInfoUnlock(unlockId, server) {
				return nil, false
			}
		}
		// 检测时间
		if activityConfig.EventOpenTime != 0 && activityConfig.EventOpenTime > currentTime {
			return nil, false
		}
		if activityConfig.EventEndTime != 0 && activityConfig.EventEndTime < currentTime {
			return nil, false
		}
		if len(activityConfig.WeekOpenDays) > 0 {
			find := false
			for _, day := range activityConfig.WeekOpenDays {
				if day == int32(tool.WeekDayWithTimeStamp(currentTime)) {
					find = true
				}
			}
			if !find {
				return nil, false
			}
		} else if len(activityConfig.MonthOpenDays) > 0 {
			find := false
			for _, day := range activityConfig.MonthOpenDays {
				if day == int32(tool.MonthDayWithTimeStamp(currentTime)) {
					find = true
				}
			}
			if !find {
				return nil, false
			}
		}
	}
	version := ""
	if len(activityConfig.WeekOpenDays) != 0 || len(activityConfig.MonthOpenDays) != 0 || activityConfig.OpenLoopNum != 0 || isLoopActivity(activityConfig) {
		if openCount >= activityConfig.OpenLoopNum {
			return nil, false
		}
		openCount++
	} else {
		if openCount >= 1 {
			return nil, false
		}
		openCount = 1
	}
	// 活动开启服务器检测
	date := tool.GetTodayDataStringByTimeStamp(currentTime)
	switch activityConfig.ServerType {
	case int32(enum.ActivityServerType_Single):
		if !activityConfig.ServerUnitInfo.AllServer {
			for _, units := range activityConfig.ServerUnitInfo.ServerUnitVector {
				find := false
				if server.ServerId >= units.Left && server.ServerId <= units.Right {
					find = true
				} else {
					for _, id := range units.Units {
						if server.ServerId == id {
							find = true
						}
					}
				}
				if !find {
					return nil, false
				}
			}
		}
		version = getActivityVersion(date, server.ServerId, openCount)
	case int32(enum.ActivityServerType_Multi):
		if !activityConfig.ServerUnitInfo.AllServer {
			for i, units := range activityConfig.ServerUnitInfo.ServerUnitVector {
				find := false
				if server.ServerId >= units.Left && server.ServerId <= units.Right {
					find = true
				} else {
					for _, id := range units.Units {
						if server.ServerId == id {
							find = true
						}
					}
				}
				if !find {
					return nil, false
				}
				version = getActivityVersion(date, int32(i), openCount)
			}
		}
	}
	unlockIds := make([]int32, 0)
	for _, unlockId := range activityConfig.UnlockIds {
		unlockIds = append(unlockIds, unlockId)
	}
	openInfo := &model.ServerOpenActivityEntity{
		ActivityId:   activityConfig.Id,
		Version:      version,
		OpenServerId: server.ServerId,
		OpenTime:     currentTime,
	}
	if activityConfig.SettleTime == 0 {
		openInfo.SettleTime = math.MaxInt64
	} else {
		openInfo.SettleTime = currentTime + int64(activityConfig.SettleTime)*tool.HOUR_MILLI
	}
	duration := int32(0)
	if len(activityConfig.DurationTimes) > 1 {
		duration = activityConfig.DurationTimes[openCount-1]
	} else if len(activityConfig.DurationTimes) > 0 {
		duration = activityConfig.DurationTimes[0]
	}
	if duration == 0 {
		openInfo.EndTime = math.MaxInt64
	} else {
		openInfo.EndTime = currentTime + int64(duration)*tool.HOUR_MILLI
	}
	return openInfo, true
}

func (s *GatewayActivityService) IsActivitySettled(serverId int32, activityId int32, version string) bool {
	return s.serverOpenActivityModel.IsActivitySettled(serverId, activityId, version)
}

func (s *GatewayActivityService) GetAllActivityByServerId(serverId int32) []logicCommon.GameActivityInterface {
	return s.serverOpenActivityModel.GetAllActivityByServerId(serverId)
}

func (s *GatewayActivityService) IsActivityOpen(serverId int32, activityId int32) logicCommon.GameActivityInterface {
	cfg := s.serverActivityConfigModel.GetActivityConfig(activityId)
	if cfg == nil {
		return nil
	}
	realConfig, ok := cfg.(*model.ServerActivityConfigEntity)
	if !ok {
		return nil
	}
	if realConfig.IfBlock == 1 {
		return nil
	}
	for _, s := range realConfig.IfBlockServers {
		if serverId == s {
			return nil
		}
	}
	return s.serverOpenActivityModel.IsActivityOpen(serverId, activityId)
}

func (s *GatewayActivityService) GetActivityConfig(activityId int32) logicCommon.GameActivityConfigInterface {
	return s.serverActivityConfigModel.GetActivityConfig(activityId)
}

func (s *GatewayActivityService) checkConfigChange(config *model.ServerActivityConfigEntity, activity *model.ServerOpenActivityEntity, currentTime int64) bool {
	changed := false
	// 活动已开启，则不检测活动开启时间
	if activity.OpenTime != config.EventOpenTime {
		if currentTime < activity.OpenTime {
			if config.EventOpenTime != 0 {
				activity.OpenTime = config.EventOpenTime
			} else {
				activity.OpenTime = currentTime
			}
			changed = true
		}
	}

	duration := int32(0)
	if len(config.DurationTimes) > 1 {
		duration = config.DurationTimes[0]
	} else if len(config.DurationTimes) > 0 {
		duration = config.DurationTimes[0]
	}
	endTime := int64(math.MaxInt64)
	if duration != 0 {
		endTime = activity.OpenTime + int64(duration)*tool.HOUR_MILLI
	}

	// 活动已结束，则不检测活动结束时间
	if activity.EndTime != endTime {
		if activity.EndTime > currentTime {
			activity.EndTime = endTime
			changed = true
		}
	}
	settleTime := int64(math.MaxInt64)
	if duration != 0 {
		settleTime = activity.OpenTime + int64(config.SettleTime)*tool.HOUR_MILLI
	}
	// 活动已结算，则不检测活动结算时间
	if activity.SettleTime != settleTime {
		if activity.SettleTime > currentTime {
			activity.SettleTime = settleTime
			changed = true
		}
	}
	return changed
}
