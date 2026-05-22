package gloryArenaService

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
)

const (
	gloryArenaStateTickInterval = time.Minute
	gloryArenaStateRedisTTL     = 3 * 24 * time.Hour
	gloryArenaPoolRedisTTL      = 8 * 24 * time.Hour
	gloryArenaPoolInitMember    = "__pool_init__"
)

type GatewayGloryArenaService struct {
	serverInfoService *gameServerInfoService.GameServerInfoService
}

func NewGatewayGloryArenaStateService(serverInfoService *gameServerInfoService.GameServerInfoService) *GatewayGloryArenaService {
	return &GatewayGloryArenaService{
		serverInfoService: serverInfoService,
	}
}

func (s *GatewayGloryArenaService) StartService() {
	s.syncNow(true)
	go func() {
		ticker := time.NewTicker(gloryArenaStateTickInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.syncNow(false)
		}
	}()
}

func (s *GatewayGloryArenaService) syncNow(force bool) {
	if s == nil || s.serverInfoService == nil {
		return
	}
	_ = force

	now := tool.UnixNowMilli()
	openServerMap := s.serverInfoService.GetAllServerInfo()
	openServers := make([]logicCommon.ServerInfoInterface, 0, len(openServerMap))
	for _, info := range openServerMap {
		openServers = append(openServers, info)
	}
	if len(openServers) == 0 {
		return
	}
	sort.Slice(openServers, func(i, j int) bool {
		return openServers[i].GetServerId() < openServers[j].GetServerId()
	})

	allServerState, err := logicCommon.GetGloryArenaCrossServerResultByTime(openServers, now)
	if err != nil {
		logger.ErrorBySprintf("[gatewayGloryArenaStateService] build cross server state failed err:%v", err)
		return
	}

	fields := make(map[string]interface{}, len(allServerState))
	for serverID, state := range allServerState {
		data, marshalErr := json.Marshal(state)
		if marshalErr != nil {
			logger.ErrorBySprintf("[gatewayGloryArenaStateService] marshal state failed serverId:%d err:%v", serverID, marshalErr)
			continue
		}
		fields[strconv.Itoa(int(serverID))] = data
	}
	if len(fields) == 0 {
		return
	}

	ctx := context.Background()
	opsKey := enum.GetGloryArenaOpsStateKey()
	pipe := dbService.RDB.Pipeline()
	pipe.HSet(ctx, opsKey, fields)
	pipe.Expire(ctx, opsKey, gloryArenaStateRedisTTL)
	if _, err = pipe.Exec(ctx); err != nil {
		logger.ErrorBySprintf("[gatewayGloryArenaStateService] sync redis failed err:%v", err)
		return
	}

	s.ensurePoolKeysByOpsState(ctx, fields)
}

func (s *GatewayGloryArenaService) ensurePoolKeysByOpsState(ctx context.Context, opsFields map[string]interface{}) {
	if len(opsFields) == 0 {
		return
	}

	handledVersion := make(map[string]bool)
	for serverID, raw := range opsFields {
		var payload []byte
		switch v := raw.(type) {
		case []byte:
			payload = v
		case string:
			payload = []byte(v)
		default:
			continue
		}
		if len(payload) == 0 {
			continue
		}

		state := &logicCommon.GloryArenaOpsServerState{}
		if err := json.Unmarshal(payload, state); err != nil {
			logger.ErrorBySprintf("[gatewayGloryArenaStateService] unmarshal ops state failed serverId:%s err:%v", serverID, err)
			continue
		}
		if state.GroupVersion == "" || handledVersion[state.GroupVersion] {
			continue
		}
		handledVersion[state.GroupVersion] = true

		opponentKey := enum.GetGloryArenaPoolOpponentRoundKey(state.GroupVersion)
		qualifyKey := enum.GetGloryArenaPoolQualifyRoundKey(state.GroupVersion)

		if err := ensurePoolZSetKey(ctx, opponentKey); err != nil {
			logger.ErrorBySprintf("[gatewayGloryArenaStateService] ensure opponent pool key failed groupVersion:%s key:%s err:%v", state.GroupVersion, opponentKey, err)
		}
		if err := ensurePoolSetKey(ctx, qualifyKey); err != nil {
			logger.ErrorBySprintf("[gatewayGloryArenaStateService] ensure qualify pool key failed groupVersion:%s key:%s err:%v", state.GroupVersion, qualifyKey, err)
		}
	}
}

func ensurePoolZSetKey(ctx context.Context, key string) error {
	exists, err := dbService.RDB.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		_ = dbService.RDB.Expire(ctx, key, gloryArenaPoolRedisTTL).Err()
		return nil
	}

	pipe := dbService.RDB.Pipeline()
	pipe.ZAdd(ctx, key, &redis.Z{Score: -1, Member: gloryArenaPoolInitMember})
	pipe.Expire(ctx, key, gloryArenaPoolRedisTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func ensurePoolSetKey(ctx context.Context, key string) error {
	exists, err := dbService.RDB.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		_ = dbService.RDB.Expire(ctx, key, gloryArenaPoolRedisTTL).Err()
		return nil
	}

	pipe := dbService.RDB.Pipeline()
	pipe.SAdd(ctx, key, gloryArenaPoolInitMember)
	pipe.Expire(ctx, key, gloryArenaPoolRedisTTL)
	_, err = pipe.Exec(ctx)
	return err
}
