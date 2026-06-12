package gloryArenaService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/rankBoardPlatform"
	"github.com/drop/GoServer/server/logic/rankboardService"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
)

var (
	ErrGloryArenaPoolRankNotConfigured    = errors.New("glory arena battle-power rank config not found")
	ErrGloryArenaQualifyRankNotConfigured = errors.New("glory arena arena-rank config not found")
	ErrGloryArenaPoolServerInfoMissing    = errors.New("glory arena server info service not initialized")
	ErrGloryArenaPoolInvalidRankID        = errors.New("invalid rank id for glory arena pool")
)

const (
	gloryArenaRoundDays   = 3
	gloryArenaPoolDataTTL = 8 * 24 * time.Hour
	gloryArenaRoundOffset = 30 * time.Minute
	gloryArenaMergeHourLo = 3
	gloryArenaMergeHourHi = 4
)

type GloryArenaRoundInfo struct {
	IsRoundOpen        bool
	IsChallengeWindow  bool
	SeasonType         enum.GloryArenaSeasonType
	RoundIndexInSeason int32 // season mode: 1~4, preseason mode: 1~2
	SeasonSeq          int32 // season mode: 1...
	IsFinalRound       bool
	RoundStart         int64
	RoundEnd           int64
}

type GloryArenaQualifiedRankPlayer struct {
	ServerID int32
	PlayerID int64
	Rank     int32
	Score    int64
}

type GloryArenaPoolBuildResult struct {
	PoolKey        string
	SourceServerID int32
	GroupServerIDs []int32
	TopN           int32
	MemberCount    int
}

type RankBoardGloryArenaPoolService struct {
	lastDailyMergeDate int32
}

var service *RankBoardGloryArenaPoolService

func InitService() {
	service = &RankBoardGloryArenaPoolService{lastDailyMergeDate: 0}
}

func GetService() *RankBoardGloryArenaPoolService {
	if service == nil {
		InitService()
	}
	return service
}

func StartService() {
	GetService().StartService()
}

func (s *RankBoardGloryArenaPoolService) StartService() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := tool.UnixNowMilli()
			logger.InfoWithSprintf("[gloryArenaPoolService] tick start now:%d local:%s", now, time.UnixMilli(now).Format(time.RFC3339))
			if err := s.buildRoundPoolForAllOpenServers(now); err != nil {
				logger.ErrorBySprintf("[gloryArenaPoolService] build round pool failed err:%v", err)
			} else {
				logger.InfoWithSprintf("[gloryArenaPoolService] build round pool finished now:%d", now)
			}
			if err := s.tryDailyMergePools(now); err != nil {
				logger.ErrorBySprintf("[gloryArenaPoolService] daily merge pools failed err:%v", err)
			} else {
				logger.InfoWithSprintf("[gloryArenaPoolService] daily merge pools finished now:%d", now)
			}
		}
	}()
}

func (s *RankBoardGloryArenaPoolService) buildRoundPoolForAllOpenServers(currentTime int64) error {
	logger.InfoWithSprintf("[gloryArenaPoolService] build round pool begin currentTime:%d", currentTime)
	sortedServers, crossStateMap, err := s.loadCrossServerStateMap(currentTime)
	if err != nil {
		return err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build round pool loaded servers:%d crossStates:%d", len(sortedServers), len(crossStateMap))
	if len(sortedServers) == 0 || len(crossStateMap) == 0 {
		return nil
	}

	handledVersion := make(map[string]bool)
	for _, server := range sortedServers {
		if server == nil {
			continue
		}
		serverID := server.GetServerId()
		crossState := crossStateMap[serverID]
		if crossState == nil || !crossState.IsRoundOpen || crossState.GroupVersion == "" {
			continue
		}
		if handledVersion[crossState.GroupVersion] {
			continue
		}
		logger.InfoWithSprintf("[gloryArenaPoolService] build round pool processing serverId:%d groupVersion:%s roundOpen:%t seasonType:%d roundStart:%d roundEnd:%d groupServerIds:%v",
			serverID, crossState.GroupVersion, crossState.IsRoundOpen, crossState.SeasonType, crossState.RoundStart, crossState.RoundEnd, crossState.GroupServerIDs)
		if err = s.ensureRoundPoolByCrossState(serverID, crossState, currentTime); err != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] build round pools failed serverId:%d groupVersion:%s err:%v", serverID, crossState.GroupVersion, err)
		}
		handledVersion[crossState.GroupVersion] = true
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build round pool done handledVersions:%d", len(handledVersion))
	return nil
}

func (s *RankBoardGloryArenaPoolService) loadCrossServerStateMap(currentTime int64) ([]logicCommon.ServerInfoInterface, map[int32]*logicCommon.GloryArenaOpsServerState, error) {
	servers, err := s.loadSortedOpenServers()
	if err != nil {
		return nil, nil, err
	}

	// Gateway owns ops_state and writes the authoritative groupVersion.
	// Rank should consume this first to avoid split-brain with local server-info cache.
	opsStateMap, opsErr := s.loadCrossServerStateMapFromRedis()
	if opsErr != nil {
		logger.ErrorBySprintf("[gloryArenaPoolService] load ops_state from redis failed, fallback calculate by local server info, err:%v", opsErr)
	} else if len(opsStateMap) > 0 {
		return servers, opsStateMap, nil
	}

	if len(servers) == 0 {
		return nil, map[int32]*logicCommon.GloryArenaOpsServerState{}, nil
	}

	stateMap, err := logicCommon.GetGloryArenaCrossServerResultByTime(servers, currentTime)
	if err != nil {
		return nil, nil, err
	}
	return servers, stateMap, nil
}

func (s *RankBoardGloryArenaPoolService) loadCrossServerStateMapFromRedis() (map[int32]*logicCommon.GloryArenaOpsServerState, error) {
	ctx := context.Background()
	rawStates, err := dbService.RDB.HGetAll(ctx, enum.GetGloryArenaOpsStateKey()).Result()
	if err != nil {
		return nil, err
	}
	if len(rawStates) == 0 {
		return map[int32]*logicCommon.GloryArenaOpsServerState{}, nil
	}

	stateMap := make(map[int32]*logicCommon.GloryArenaOpsServerState, len(rawStates))
	for serverIDStr, payload := range rawStates {
		if payload == "" {
			continue
		}
		serverID64, parseErr := strconv.ParseInt(serverIDStr, 10, 32)
		if parseErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] parse ops_state serverId failed serverId:%s err:%v", serverIDStr, parseErr)
			continue
		}
		state := &logicCommon.GloryArenaOpsServerState{}
		if unmarshalErr := json.Unmarshal([]byte(payload), state); unmarshalErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] unmarshal ops_state failed serverId:%s err:%v", serverIDStr, unmarshalErr)
			continue
		}
		if state.ServerID <= 0 {
			state.ServerID = int32(serverID64)
		}
		if state.ServerID <= 0 {
			continue
		}
		stateMap[state.ServerID] = state
	}
	return stateMap, nil
}

func (s *RankBoardGloryArenaPoolService) loadSortedOpenServers() ([]logicCommon.ServerInfoInterface, error) {
	serverInfoSvc := rankBoardPlatform.GetServerInfoService()
	if serverInfoSvc == nil {
		return nil, ErrGloryArenaPoolServerInfoMissing
	}
	allServerInfo := serverInfoSvc.GetAllServerInfo()
	if len(allServerInfo) == 0 {
		return nil, nil
	}
	servers := make([]logicCommon.ServerInfoInterface, 0, len(allServerInfo))
	for _, server := range allServerInfo {
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return nil, nil
	}
	// 排序服务器列表
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].GetServerId() < servers[j].GetServerId()
	})
	return servers, nil
}

func (s *RankBoardGloryArenaPoolService) ensureRoundPoolByCrossState(serverID int32, crossState *logicCommon.GloryArenaOpsServerState, currentTime int64) error {
	if crossState == nil || crossState.GroupVersion == "" || crossState.RoundStart <= 0 {
		return nil
	}
	ctx := context.Background()
	opponentKey := enum.GetGloryArenaPoolOpponentRoundKey(crossState.GroupVersion)
	qualifyKey := enum.GetGloryArenaPoolQualifyRoundKey(crossState.GroupVersion)
	logger.InfoWithSprintf("[gloryArenaPoolService] ensure round pool begin serverId:%d groupVersion:%s opponentKey:%s qualifyKey:%s", serverID, crossState.GroupVersion, opponentKey, qualifyKey)
	needBuildOpponent, err := needRebuildOpponentPool(ctx, opponentKey)
	if err != nil {
		return err
	}
	needBuildQualify, err := needRebuildQualifyPool(ctx, qualifyKey)
	if err != nil {
		return err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] ensure round pool decide serverId:%d groupVersion:%s needBuildOpponent:%t needBuildQualify:%t",
		serverID, crossState.GroupVersion, needBuildOpponent, needBuildQualify)

	if needBuildQualify {
		buildResult, buildErr := s.buildQualifyPoolByCrossState(serverID, crossState, currentTime)
		if buildErr != nil {
			err = buildErr
			return err
		}
		if buildResult != nil {
			logger.InfoWithSprintf("[gloryArenaPoolService] build qualify pool success serverId:%d groupVersion:%s poolKey:%s topN:%d members:%d groupServerIds:%v",
				serverID, crossState.GroupVersion, buildResult.PoolKey, buildResult.TopN, buildResult.MemberCount, buildResult.GroupServerIDs)
		}
	}
	if needBuildOpponent {
		buildResult, buildErr := s.buildChallengePoolByCrossState(serverID, crossState, currentTime)
		if buildErr != nil {
			err = buildErr
			return err
		}
		if buildResult != nil {
			logger.InfoWithSprintf("[gloryArenaPoolService] build opponent pool success serverId:%d groupVersion:%s poolKey:%s topN:%d members:%d groupServerIds:%v",
				serverID, crossState.GroupVersion, buildResult.PoolKey, buildResult.TopN, buildResult.MemberCount, buildResult.GroupServerIDs)
		}
	}
	return nil
}

func (s *RankBoardGloryArenaPoolService) tryDailyMergePools(currentTime int64) error {
	now := time.UnixMilli(currentTime)
	hour := now.Hour()
	if hour < gloryArenaMergeHourLo || hour >= gloryArenaMergeHourHi {
		logger.InfoWithSprintf("[gloryArenaPoolService] daily merge skip by hour currentTime:%d hour:%d allowed:[%d,%d)", currentTime, hour, gloryArenaMergeHourLo, gloryArenaMergeHourHi)
		return nil
	}
	today := int32(tool.GetTodayDataIntByTimeStamp(currentTime))
	if s.lastDailyMergeDate == today {
		logger.InfoWithSprintf("[gloryArenaPoolService] daily merge skip by date today:%d lastMergedDate:%d", today, s.lastDailyMergeDate)
		return nil
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] daily merge begin currentTime:%d today:%d", currentTime, today)

	sortedServers, crossStateMap, err := s.loadCrossServerStateMap(currentTime)
	if err != nil {
		return err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] daily merge loaded servers:%d crossStates:%d", len(sortedServers), len(crossStateMap))
	if len(sortedServers) == 0 || len(crossStateMap) == 0 {
		return nil
	}

	handledVersion := make(map[string]bool)
	for _, server := range sortedServers {
		if server == nil {
			continue
		}
		serverID := server.GetServerId()
		crossState := crossStateMap[serverID]
		if crossState == nil || crossState.GroupVersion == "" || len(crossState.GroupServerIDs) == 0 {
			continue
		}
		if handledVersion[crossState.GroupVersion] {
			continue
		}
		logger.InfoWithSprintf("[gloryArenaPoolService] daily merge processing serverId:%d groupVersion:%s seasonType:%d roundStart:%d roundEnd:%d groupServerIds:%v",
			serverID, crossState.GroupVersion, crossState.SeasonType, crossState.RoundStart, crossState.RoundEnd, crossState.GroupServerIDs)
		if err = s.mergePoolsByCrossState(serverID, crossState, currentTime); err != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] daily merge pool failed serverId:%d groupVersion:%s err:%v", serverID, crossState.GroupVersion, err)
			continue
		}
		logger.InfoWithSprintf("[gloryArenaPoolService] daily merge success serverId:%d groupVersion:%s", serverID, crossState.GroupVersion)
		handledVersion[crossState.GroupVersion] = true
	}

	s.lastDailyMergeDate = today
	logger.InfoWithSprintf("[gloryArenaPoolService] daily merge done today:%d handledVersions:%d", today, len(handledVersion))
	return nil
}

func (s *RankBoardGloryArenaPoolService) mergePoolsByCrossState(serverID int32, crossState *logicCommon.GloryArenaOpsServerState, currentTime int64) error {
	logger.InfoWithSprintf("[gloryArenaPoolService] merge pools begin serverId:%d groupVersion:%s groupServerIds:%v", serverID, crossState.GroupVersion, crossState.GroupServerIDs)
	if err := s.mergeOpponentPoolByCrossState(crossState); err != nil {
		return err
	}
	if err := s.mergeQualifyPoolByCrossState(crossState, currentTime); err != nil {
		return err
	}
	_ = serverID
	logger.InfoWithSprintf("[gloryArenaPoolService] merge pools done serverId:%d groupVersion:%s", serverID, crossState.GroupVersion)
	return nil
}

func (s *RankBoardGloryArenaPoolService) mergeOpponentPoolByCrossState(crossState *logicCommon.GloryArenaOpsServerState) error {
	if crossState == nil || crossState.GroupVersion == "" {
		return nil
	}
	topN := gameConfig.GetGloryArenaOpponentRank()
	if topN <= 0 {
		return ErrGloryArenaPoolRankNotConfigured
	}

	poolKey := enum.GetGloryArenaPoolOpponentRoundKey(crossState.GroupVersion)
	merged, err := loadExistingOpponentPoolMembers(poolKey)
	if err != nil {
		return err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] merge opponent pool load existing groupVersion:%s poolKey:%s existingMembers:%d", crossState.GroupVersion, poolKey, len(merged))
	for _, sid := range crossState.GroupServerIDs {
		list, listErr := s.getBattlePowerTopPlayersByServer(sid, topN)
		if listErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] daily merge load topN failed serverId:%d err:%v", sid, listErr)
			continue
		}
		for _, p := range list {
			if p == nil || p.PlayerID <= 0 {
				continue
			}
			if old, ok := merged[p.PlayerID]; !ok || p.Score > old {
				merged[p.PlayerID] = p.Score
			}
		}
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] merge opponent pool write groupVersion:%s poolKey:%s mergedMembers:%d", crossState.GroupVersion, poolKey, len(merged))
	return writeMergedChallengePoolToRedis(poolKey, merged)
}

func (s *RankBoardGloryArenaPoolService) mergeQualifyPoolByCrossState(crossState *logicCommon.GloryArenaOpsServerState, currentTime int64) error {
	if crossState == nil || crossState.GroupVersion == "" {
		return nil
	}
	topN := gameConfig.GetGloryArenaEntryRequirement()
	if topN <= 0 {
		return ErrGloryArenaQualifyRankNotConfigured
	}

	poolKey := enum.GetGloryArenaPoolQualifyRoundKey(crossState.GroupVersion)
	merged, err := loadExistingQualifyPoolMembers(poolKey)
	if err != nil {
		return err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] merge qualify pool load existing groupVersion:%s poolKey:%s existingMembers:%d", crossState.GroupVersion, poolKey, len(merged))
	for _, sid := range crossState.GroupServerIDs {
		list, listErr := s.getArenaTopPlayersByServer(sid, topN, currentTime)
		if listErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] daily merge load arena topN failed serverId:%d err:%v", sid, listErr)
			continue
		}
		for _, p := range list {
			if p == nil || p.PlayerID <= 0 {
				continue
			}
			merged[p.PlayerID] = true
		}
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] merge qualify pool write groupVersion:%s poolKey:%s mergedMembers:%d", crossState.GroupVersion, poolKey, len(merged))
	return writeMergedQualifyPoolToRedis(poolKey, merged)
}

func needRebuildOpponentPool(ctx context.Context, key string) (bool, error) {
	exists, err := dbService.RDB.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists == 0 {
		return true, nil
	}
	total, err := dbService.RDB.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	_, err = dbService.RDB.ZScore(ctx, key, gloryArenaPoolInitMember).Result()
	if err == nil {
		return total <= 1, nil
	}
	if err == redis.Nil {
		return total == 0, nil
	}
	return false, err
}

func needRebuildQualifyPool(ctx context.Context, key string) (bool, error) {
	exists, err := dbService.RDB.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists == 0 {
		return true, nil
	}
	total, err := dbService.RDB.SCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	isSentinel, err := dbService.RDB.SIsMember(ctx, key, gloryArenaPoolInitMember).Result()
	if err != nil {
		return false, err
	}
	if isSentinel {
		return total <= 1, nil
	}
	return total == 0, nil
}

func (s *RankBoardGloryArenaPoolService) buildChallengePoolByCrossState(serverID int32, crossState *logicCommon.GloryArenaOpsServerState, currentTime int64) (*GloryArenaPoolBuildResult, error) {
	topN := gameConfig.GetGloryArenaOpponentRank()
	if topN <= 0 {
		return nil, ErrGloryArenaPoolRankNotConfigured
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build opponent pool begin serverId:%d groupVersion:%s topN:%d groupServerIds:%v",
		serverID, crossState.GroupVersion, topN, crossState.GroupServerIDs)

	uniq := make(map[int64]*GloryArenaQualifiedRankPlayer)
	failedServers := make([]int32, 0)
	for _, sid := range crossState.GroupServerIDs {
		list, listErr := s.getBattlePowerTopPlayersByServer(sid, topN)
		if listErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] load topN failed serverId:%d err:%v", sid, listErr)
			failedServers = append(failedServers, sid)
			continue
		}
		for _, p := range list {
			if p == nil || p.PlayerID <= 0 {
				continue
			}
			if old, ok := uniq[p.PlayerID]; ok {
				if p.Score > old.Score {
					old.Score = p.Score
					old.Rank = p.Rank
					old.ServerID = p.ServerID
				}
				continue
			}
			uniq[p.PlayerID] = p
		}
	}
	if len(failedServers) > 0 {
		return nil, fmt.Errorf("build challenge pool aborted, failed servers:%v", failedServers)
	}

	poolKey := enum.GetGloryArenaPoolOpponentRoundKey(crossState.GroupVersion)
	if err := writeChallengePoolToRedis(poolKey, uniq); err != nil {
		return nil, err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build opponent pool write done serverId:%d groupVersion:%s poolKey:%s members:%d",
		serverID, crossState.GroupVersion, poolKey, len(uniq))

	groupIDs := make([]int32, len(crossState.GroupServerIDs))
	copy(groupIDs, crossState.GroupServerIDs)
	return &GloryArenaPoolBuildResult{
		PoolKey:        poolKey,
		SourceServerID: serverID,
		GroupServerIDs: groupIDs,
		TopN:           topN,
		MemberCount:    len(uniq),
	}, nil
}

func (s *RankBoardGloryArenaPoolService) buildQualifyPoolByCrossState(serverID int32, crossState *logicCommon.GloryArenaOpsServerState, currentTime int64) (*GloryArenaPoolBuildResult, error) {
	topN := gameConfig.GetGloryArenaEntryRequirement()
	if topN <= 0 {
		return nil, ErrGloryArenaQualifyRankNotConfigured
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build qualify pool begin serverId:%d groupVersion:%s topN:%d groupServerIds:%v currentTime:%d",
		serverID, crossState.GroupVersion, topN, crossState.GroupServerIDs, currentTime)

	uniq := make(map[int64]*GloryArenaQualifiedRankPlayer)
	for _, sid := range crossState.GroupServerIDs {
		list, listErr := s.getArenaTopPlayersByServer(sid, topN, currentTime)
		if listErr != nil {
			logger.ErrorBySprintf("[gloryArenaPoolService] load arena topN failed serverId:%d err:%v", sid, listErr)
			continue
		}
		for _, p := range list {
			if p == nil || p.PlayerID <= 0 {
				continue
			}
			if old, ok := uniq[p.PlayerID]; ok {
				if p.Score > old.Score {
					old.Score = p.Score
					old.Rank = p.Rank
					old.ServerID = p.ServerID
				}
				continue
			}
			uniq[p.PlayerID] = p
		}
	}

	poolKey := enum.GetGloryArenaPoolQualifyRoundKey(crossState.GroupVersion)
	if err := writeQualifiedPoolToRedis(poolKey, uniq); err != nil {
		return nil, err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] build qualify pool write done serverId:%d groupVersion:%s poolKey:%s members:%d",
		serverID, crossState.GroupVersion, poolKey, len(uniq))

	groupIDs := make([]int32, len(crossState.GroupServerIDs))
	copy(groupIDs, crossState.GroupServerIDs)
	return &GloryArenaPoolBuildResult{
		PoolKey:        poolKey,
		SourceServerID: serverID,
		GroupServerIDs: groupIDs,
		TopN:           topN,
		MemberCount:    len(uniq),
	}, nil
}

func (s *RankBoardGloryArenaPoolService) getBattlePowerTopPlayersByServer(serverID int32, topN int32) ([]*GloryArenaQualifiedRankPlayer, error) {
	if topN <= 0 {
		return nil, nil
	}
	rankID := fmt.Sprintf("common_%d_%d", enum.GLORY_ARENA_BATTLE_POWER_RANK_ID, serverID)
	logger.InfoWithSprintf("[gloryArenaPoolService] query battle power rank table serverId:%d rankTable:%s topN:%d",
		serverID, rankID, topN)
	rankInfos, _, err := rankboardService.GetRankInfo(rankID, int(topN), 0)
	if err != nil {
		logger.ErrorBySprintf("[gloryArenaPoolService] query battle power rank failed serverId:%d rankTable:%s topN:%d err:%v",
			serverID, rankID, topN, err)
		return nil, err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] query battle power rank success serverId:%d rankTable:%s fetched:%d",
		serverID, rankID, len(rankInfos))
	return toQualifiedPlayers(serverID, topN, rankInfos), nil
}

func (s *RankBoardGloryArenaPoolService) getArenaTopPlayersByServer(serverID int32, topN int32, currentTime int64) ([]*GloryArenaQualifiedRankPlayer, error) {
	if topN <= 0 {
		return nil, nil
	}
	version := logicCommon.GetArenaRankVersionByTime(serverID, currentTime)
	arenaRankID := gameConfig.GetArenaRankId()
	sourceRankTable := fmt.Sprintf("common_%d_%s", arenaRankID, version)
	versionRankID, err := logicCommon.GetRankUniqueId(gameConfig.GetArenaRankId(), 0, 0, serverID, version)
	if err != nil {
		logger.ErrorBySprintf("[gloryArenaPoolService] arena GetRankUniqueId failed serverId:%d arenaRankId:%d version:%s sourceRankTable:%s err:%+v",
			serverID, arenaRankID, version, sourceRankTable, err)
		return nil, err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] query arena rank table serverId:%d arenaRankId:%d version:%s sourceRankTable:%s queryRankID:%s topN:%d",
		serverID, arenaRankID, version, sourceRankTable, versionRankID, topN)
	rankInfos, _, err := rankboardService.GetRankInfo(versionRankID, int(topN), 0)
	if err != nil {
		logger.ErrorBySprintf("[gloryArenaPoolService] query arena rank failed serverId:%d version:%s queryRankID:%s topN:%d err:%v",
			serverID, version, versionRankID, topN, err)
		return nil, err
	}
	logger.InfoWithSprintf("[gloryArenaPoolService] query arena rank success serverId:%d version:%s queryRankID:%s fetched:%d",
		serverID, version, versionRankID, len(rankInfos))
	return toQualifiedPlayers(serverID, topN, rankInfos), nil
}

func toQualifiedPlayers(serverID int32, topN int32, rankInfos []*model.RankBoardInfoEntity) []*GloryArenaQualifiedRankPlayer {
	result := make([]*GloryArenaQualifiedRankPlayer, 0, len(rankInfos))
	for _, info := range rankInfos {
		if info == nil || info.Id <= 0 {
			continue
		}
		if info.Rank <= 0 || info.Rank > topN {
			continue
		}
		result = append(result, &GloryArenaQualifiedRankPlayer{
			ServerID: serverID,
			PlayerID: info.Id,
			Rank:     info.Rank,
			Score:    info.Score,
		})
	}
	return result
}

func writeChallengePoolToRedis(poolKey string, players map[int64]*GloryArenaQualifiedRankPlayer) error {
	ctx := context.Background()
	pipe := dbService.RDB.Pipeline()
	pipe.Del(ctx, poolKey)
	pipe.ZAdd(ctx, poolKey, &redis.Z{
		Score:  -1,
		Member: gloryArenaPoolInitMember,
	})
	for _, player := range players {
		pipe.ZAdd(ctx, poolKey, &redis.Z{
			Score:  float64(player.Score),
			Member: strconv.FormatInt(player.PlayerID, 10),
		})
	}
	pipe.Expire(ctx, poolKey, gloryArenaPoolDataTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func writeQualifiedPoolToRedis(poolKey string, players map[int64]*GloryArenaQualifiedRankPlayer) error {
	ctx := context.Background()
	pipe := dbService.RDB.Pipeline()
	pipe.Del(ctx, poolKey)
	pipe.SAdd(ctx, poolKey, gloryArenaPoolInitMember)
	for _, player := range players {
		pipe.SAdd(ctx, poolKey, strconv.FormatInt(player.PlayerID, 10))
	}
	pipe.Expire(ctx, poolKey, gloryArenaPoolDataTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func loadExistingOpponentPoolMembers(poolKey string) (map[int64]int64, error) {
	ctx := context.Background()
	result := make(map[int64]int64)
	rawMembers, err := dbService.RDB.ZRangeWithScores(ctx, poolKey, 0, -1).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	for _, raw := range rawMembers {
		member, ok := raw.Member.(string)
		if !ok || member == "" || member == gloryArenaPoolInitMember {
			continue
		}
		playerID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr != nil || playerID <= 0 {
			continue
		}
		score := int64(raw.Score)
		if old, exists := result[playerID]; !exists || score > old {
			result[playerID] = score
		}
	}
	return result, nil
}

func loadExistingQualifyPoolMembers(poolKey string) (map[int64]bool, error) {
	ctx := context.Background()
	result := make(map[int64]bool)
	members, err := dbService.RDB.SMembers(ctx, poolKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	for _, member := range members {
		if member == "" || member == gloryArenaPoolInitMember {
			continue
		}
		playerID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr != nil || playerID <= 0 {
			continue
		}
		result[playerID] = true
	}
	return result, nil
}

func writeMergedChallengePoolToRedis(poolKey string, players map[int64]int64) error {
	ctx := context.Background()
	pipe := dbService.RDB.Pipeline()
	pipe.Del(ctx, poolKey)
	pipe.ZAdd(ctx, poolKey, &redis.Z{
		Score:  -1,
		Member: gloryArenaPoolInitMember,
	})
	for playerID, score := range players {
		pipe.ZAdd(ctx, poolKey, &redis.Z{
			Score:  float64(score),
			Member: strconv.FormatInt(playerID, 10),
		})
	}
	pipe.Expire(ctx, poolKey, gloryArenaPoolDataTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func writeMergedQualifyPoolToRedis(poolKey string, players map[int64]bool) error {
	ctx := context.Background()
	pipe := dbService.RDB.Pipeline()
	pipe.Del(ctx, poolKey)
	pipe.SAdd(ctx, poolKey, gloryArenaPoolInitMember)
	for playerID := range players {
		pipe.SAdd(ctx, poolKey, strconv.FormatInt(playerID, 10))
	}
	pipe.Expire(ctx, poolKey, gloryArenaPoolDataTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// BuildChallengePoolOnRoundOpen rebuilds challenge list by pulling battle-power rank topN
// from all servers in the same cross-server group.
func BuildChallengePoolOnRoundOpen(serverID int32, currentTime int64) (*GloryArenaPoolBuildResult, error) {
	return GetService().BuildChallengePoolOnRoundOpen(serverID, currentTime)
}

// TryAppendByBattlePowerRankUpdate appends user into current challenge pool when rank update makes
// the user enter topN. It never removes users that dropped out of topN.
func TryAppendByBattlePowerRankUpdate(rankID string, userID int64, rank int32, score int64, currentTime int64) (bool, error) {
	return GetService().TryAppendByBattlePowerRankUpdate(rankID, userID, rank, score, currentTime)
}

// TryAppendByArenaRankUpdate appends user into current qualify pool when rank update makes
// the user enter topN. It never removes users that dropped out of topN.
func TryAppendByArenaRankUpdate(rankID string, userID int64, rank int32, score int64, currentTime int64) (bool, error) {
	return GetService().TryAppendByArenaRankUpdate(rankID, userID, rank, score, currentTime)
}

// IsQualifiedForCurrentRound returns whether the user can participate this round on the server.
// Caller should persist qualification into player glory-arena model once true.
func IsQualifiedForCurrentRound(serverID int32, userID int64, currentTime int64) (bool, error) {
	return GetService().IsQualifiedForCurrentRound(serverID, userID, currentTime)
}

func (s *RankBoardGloryArenaPoolService) BuildChallengePoolOnRoundOpen(serverID int32, currentTime int64) (*GloryArenaPoolBuildResult, error) {
	crossState, err := s.getCrossServerStateByServerID(serverID, currentTime)
	if err != nil {
		return nil, err
	}
	if crossState == nil || len(crossState.GroupServerIDs) == 0 || crossState.GroupVersion == "" {
		return nil, ErrGloryArenaPoolServerInfoMissing
	}
	if _, err = s.buildQualifyPoolByCrossState(serverID, crossState, currentTime); err != nil {
		return nil, err
	}
	return s.buildChallengePoolByCrossState(serverID, crossState, currentTime)
}

func (s *RankBoardGloryArenaPoolService) TryAppendByBattlePowerRankUpdate(rankID string, userID int64, rank int32, score int64, currentTime int64) (bool, error) {
	if userID <= 0 || rank <= 0 {
		return false, nil
	}
	commonRankID, serverID, err := parseCommonRankIDAndServerID(rankID)
	if err != nil {
		return false, err
	}
	if commonRankID != enum.GLORY_ARENA_BATTLE_POWER_RANK_ID {
		return false, nil
	}

	topN := gameConfig.GetGloryArenaOpponentRank()
	if topN <= 0 {
		return false, ErrGloryArenaPoolRankNotConfigured
	}
	if rank > topN {
		return false, nil
	}

	crossState, err := s.getCrossServerStateByServerID(serverID, currentTime)
	if err != nil {
		return false, err
	}
	if crossState == nil || crossState.GroupVersion == "" || !crossState.IsRoundOpen {
		return false, nil
	}

	poolKey := enum.GetGloryArenaPoolOpponentRoundKey(crossState.GroupVersion)
	member := strconv.FormatInt(userID, 10)
	ctx := context.Background()

	existingScore, err := dbService.RDB.ZScore(ctx, poolKey, member).Result()
	if err == nil {
		if float64(score) > existingScore {
			if _, zErr := dbService.RDB.ZAdd(ctx, poolKey, &redis.Z{Score: float64(score), Member: member}).Result(); zErr != nil {
				return false, zErr
			}
		}
		return false, nil
	}
	if err != redis.Nil {
		return false, err
	}

	if _, zErr := dbService.RDB.ZAdd(ctx, poolKey, &redis.Z{Score: float64(score), Member: member}).Result(); zErr != nil {
		return false, zErr
	}
	_ = dbService.RDB.Expire(ctx, poolKey, gloryArenaPoolDataTTL).Err()
	return true, nil
}

func (s *RankBoardGloryArenaPoolService) TryAppendByArenaRankUpdate(rankID string, userID int64, rank int32, score int64, currentTime int64) (bool, error) {
	_ = score
	if userID <= 0 || rank <= 0 {
		return false, nil
	}
	commonRankID, serverID, err := parseCommonRankIDAndServerID(rankID)
	if err != nil {
		return false, err
	}
	if commonRankID != gameConfig.GetArenaRankId() {
		return false, nil
	}

	topN := gameConfig.GetGloryArenaEntryRequirement()
	if topN <= 0 {
		return false, ErrGloryArenaQualifyRankNotConfigured
	}
	if rank > topN {
		return false, nil
	}

	crossState, err := s.getCrossServerStateByServerID(serverID, currentTime)
	if err != nil {
		return false, err
	}
	if crossState == nil || crossState.GroupVersion == "" || crossState.RoundStart <= 0 || !crossState.IsRoundOpen {
		return false, nil
	}

	qualifyKey := enum.GetGloryArenaPoolQualifyRoundKey(crossState.GroupVersion)
	member := strconv.FormatInt(userID, 10)
	ctx := context.Background()

	added, err := dbService.RDB.SAdd(ctx, qualifyKey, member).Result()
	if err != nil {
		return false, err
	}
	_ = dbService.RDB.Expire(ctx, qualifyKey, gloryArenaPoolDataTTL).Err()
	return added > 0, nil
}

func (s *RankBoardGloryArenaPoolService) IsQualifiedForCurrentRound(serverID int32, userID int64, currentTime int64) (bool, error) {
	if serverID <= 0 || userID <= 0 {
		return false, nil
	}

	topN := gameConfig.GetGloryArenaEntryRequirement()
	if topN <= 0 {
		return false, ErrGloryArenaQualifyRankNotConfigured
	}
	crossState, err := s.getCrossServerStateByServerID(serverID, currentTime)
	if err != nil {
		return false, err
	}
	if crossState == nil || !crossState.IsRoundOpen {
		return false, nil
	}

	if crossState.GroupVersion == "" || crossState.RoundStart <= 0 {
		return false, nil
	}
	qualifyKey := enum.GetGloryArenaPoolQualifyRoundKey(crossState.GroupVersion)
	member := strconv.FormatInt(userID, 10)
	return dbService.RDB.SIsMember(context.Background(), qualifyKey, member).Result()
}

func (s *RankBoardGloryArenaPoolService) getCrossServerStateByServerID(serverID int32, currentTime int64) (*logicCommon.GloryArenaOpsServerState, error) {
	if serverID <= 0 {
		return nil, ErrGloryArenaPoolServerInfoMissing
	}
	if state := logicCommon.LoadGloryArenaOpsStateByServerID(serverID); state != nil {
		return state, nil
	}

	sortedServers, err := s.loadSortedOpenServers()
	if err != nil {
		return nil, err
	}
	if len(sortedServers) == 0 {
		return nil, nil
	}

	var targetServer logicCommon.ServerInfoInterface
	for _, server := range sortedServers {
		if server != nil && server.GetServerId() == serverID {
			targetServer = server
			break
		}
	}
	if targetServer == nil {
		return nil, nil
	}

	state, err := logicCommon.CalculateGloryArenaCrossServerResult(sortedServers, targetServer, currentTime)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func isArenaRankQualified(rank int32, topN int32) bool {
	return rank > 0 && topN > 0 && rank <= topN
}

func GetGloryArenaRoundInfoByTime(serverOpenTime int64, currentTime int64) *GloryArenaRoundInfo {
	info := &GloryArenaRoundInfo{
		IsRoundOpen:       false,
		IsChallengeWindow: false,
		SeasonType:        enum.GLORY_ARENA_SEASON_TYPE_PRE,
	}
	if serverOpenTime <= 0 || currentTime <= 0 || currentTime < serverOpenTime {
		return info
	}

	// Season mode starts from the second week's Tuesday.
	seasonStart := secondWeekTuesdayStart(serverOpenTime)
	if currentTime >= seasonStart {
		return seasonRoundInfo(seasonStart, currentTime)
	}

	// Preseason mode.
	return preseasonRoundInfo(serverOpenTime, currentTime)
}

func preseasonRoundInfo(serverOpenTime int64, currentTime int64) *GloryArenaRoundInfo {
	info := &GloryArenaRoundInfo{
		SeasonType: enum.GLORY_ARENA_SEASON_TYPE_PRE,
	}
	openDayZero := dayZero(serverOpenTime)

	round1Start := openDayZero.AddDate(0, 0, 1).Add(gloryArenaRoundOffset) // day2 00:30
	round1End := round1Start.Add(gloryArenaRoundDays*24*time.Hour - gloryArenaRoundOffset)
	if currentTime < round1Start.UnixMilli() {
		info.IsRoundOpen = false
		info.RoundIndexInSeason = 1
		info.RoundStart = round1Start.UnixMilli()
		info.RoundEnd = round1End.UnixMilli()
		info.IsChallengeWindow = false
		return info
	}
	if currentTime >= round1Start.UnixMilli() && currentTime < round1End.UnixMilli() {
		info.IsRoundOpen = true
		info.RoundIndexInSeason = 1
		info.RoundStart = round1Start.UnixMilli()
		info.RoundEnd = round1End.UnixMilli()
		info.IsChallengeWindow = isChallengeWindow(currentTime)
		return info
	}

	round2Start := round1End.Add(gloryArenaRoundOffset)
	round2End := round2Start.Add(gloryArenaRoundDays*24*time.Hour - gloryArenaRoundOffset)
	nextSeasonStartTuesday := nextTuesdayAfter(round1End)
	// Keep Tuesday as the clean season-1 switch point.
	if !round2End.After(nextSeasonStartTuesday) {
		if currentTime >= round1End.UnixMilli() && currentTime < round2Start.UnixMilli() {
			info.IsRoundOpen = false
			info.RoundIndexInSeason = 2
			info.RoundStart = round2Start.UnixMilli()
			info.RoundEnd = round2End.UnixMilli()
			info.IsChallengeWindow = false
			return info
		}
		if currentTime >= round2Start.UnixMilli() && currentTime < round2End.UnixMilli() {
			info.IsRoundOpen = true
			info.RoundIndexInSeason = 2
			info.RoundStart = round2Start.UnixMilli()
			info.RoundEnd = round2End.UnixMilli()
			info.IsChallengeWindow = isChallengeWindow(currentTime)
			return info
		}
	}

	return info
}

func seasonRoundInfo(seasonStart int64, currentTime int64) *GloryArenaRoundInfo {
	info := &GloryArenaRoundInfo{
		SeasonType: enum.GLORY_ARENA_SEASON_TYPE_POST,
	}
	seasonStartMonday := weekStart(time.UnixMilli(seasonStart))
	nowWeekMonday := weekStart(time.UnixMilli(currentTime))
	if nowWeekMonday.Before(seasonStartMonday) {
		tuesday := seasonStartMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOffset)
		friday := seasonStartMonday.AddDate(0, 0, 4)
		info.RoundIndexInSeason = 1
		info.SeasonSeq = 1
		info.IsRoundOpen = false
		info.RoundStart = tuesday.UnixMilli()
		info.RoundEnd = friday.UnixMilli()
		return info
	}

	weeks := int(nowWeekMonday.Sub(seasonStartMonday) / (7 * 24 * time.Hour))
	if weeks < 0 {
		tuesday := seasonStartMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOffset)
		friday := seasonStartMonday.AddDate(0, 0, 4)
		info.RoundIndexInSeason = 1
		info.SeasonSeq = 1
		info.IsRoundOpen = false
		info.RoundStart = tuesday.UnixMilli()
		info.RoundEnd = friday.UnixMilli()
		return info
	}
	roundInWeek := int32(0)
	roundStart := time.Time{}
	roundEnd := time.Time{}
	round1Start := nowWeekMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOffset)
	round1End := nowWeekMonday.AddDate(0, 0, 4)
	round2Start := round1End.Add(gloryArenaRoundOffset)
	round2End := nowWeekMonday.AddDate(0, 0, 7)
	now := time.UnixMilli(currentTime)
	if now.Before(round1Start) {
		roundInWeek = 1
		roundStart = round1Start
		roundEnd = round1End
		info.IsRoundOpen = false
	} else if now.Before(round1End) {
		roundInWeek = 1
		roundStart = round1Start
		roundEnd = round1End
		info.IsRoundOpen = true
	} else if now.Before(round2Start) {
		roundInWeek = 2
		roundStart = round2Start
		roundEnd = round2End
		info.IsRoundOpen = false
	} else if now.Before(round2End) {
		roundInWeek = 2
		roundStart = round2Start
		roundEnd = round2End
		info.IsRoundOpen = true
	} else {
		// Off-season (Monday): expose next round window.
		roundInWeek = 1
		roundStart = nowWeekMonday.AddDate(0, 0, 8).Add(gloryArenaRoundOffset)
		roundEnd = nowWeekMonday.AddDate(0, 0, 11)
		info.IsRoundOpen = false
	}

	globalRound := int32(weeks)*2 + roundInWeek
	info.RoundIndexInSeason = ((globalRound - 1) % 4) + 1
	info.SeasonSeq = ((globalRound - 1) / 4) + 1
	info.IsFinalRound = info.RoundIndexInSeason == 4
	info.RoundStart = roundStart.UnixMilli()
	info.RoundEnd = roundEnd.UnixMilli()
	info.IsChallengeWindow = isChallengeWindow(currentTime)
	return info
}

func isChallengeWindow(currentTime int64) bool {
	startHour, endHour := gameConfig.GetGloryArenaChallengeTime()
	now := time.UnixMilli(currentTime)
	year, month, day := now.Date()
	location := now.Location()
	start := time.Date(year, month, day, int(startHour), 0, 0, 0, location)
	end := time.Date(year, month, day, int(endHour), 0, 0, 0, location)
	return !now.Before(start) && now.Before(end)
}

func dayZero(ts int64) time.Time {
	t := time.UnixMilli(ts)
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func weekStart(t time.Time) time.Time {
	y, m, d := t.Date()
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(y, m, d-wd+1, 0, 0, 0, 0, t.Location())
}

func nextTuesdayAfter(base time.Time) time.Time {
	tuesday := weekStart(base).AddDate(0, 0, 1).Add(gloryArenaRoundOffset)
	if !base.Before(tuesday) {
		tuesday = tuesday.AddDate(0, 0, 7)
	}
	return tuesday
}

func secondWeekTuesdayStart(serverOpenTime int64) int64 {
	openMonday := weekStart(time.UnixMilli(serverOpenTime))
	secondWeekTuesday := openMonday.AddDate(0, 0, 8).Add(gloryArenaRoundOffset)
	return secondWeekTuesday.UnixMilli()
}

// parseCommonRankIDAndServerID parses common rank id in the format:
// common_{rankId}_{serverId}
// common_{rankId}_{version}
// common_{rankId}_{serverId}_{version}
func parseCommonRankIDAndServerID(rankID string) (int32, int32, error) {
	parts := strings.Split(rankID, "_")
	if len(parts) < 3 || parts[0] != "common" {
		return 0, 0, ErrGloryArenaPoolInvalidRankID
	}
	rid, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0, 0, ErrGloryArenaPoolInvalidRankID
	}
	sid, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		parsedSID, _, ok := logicCommon.ParseArenaRankVersion(parts[2])
		if !ok || parsedSID <= 0 {
			return 0, 0, ErrGloryArenaPoolInvalidRankID
		}
		return int32(rid), parsedSID, nil
	}
	return int32(rid), int32(sid), nil
}
