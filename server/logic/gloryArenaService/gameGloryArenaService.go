package gloryArenaService

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
)

const (
	DefaultGloryArenaMatchCount     = 3
	gloryArenaMinPoolMemberForMatch = 3
	highWinFallbackBattleLow        = 5000
	highWinFallbackBattleHigh       = 15000
)

var (
	ErrGloryArenaPoolDataEmpty    = errors.New("glory arena pool data empty")
	ErrGloryArenaPoolDataInvalid  = errors.New("glory arena pool data invalid")
	ErrGloryArenaPoolMemberTooFew = errors.New("glory arena pool member not enough")
)

type GloryArenaPoolMember struct {
	PlayerId int64 `json:"playerId"`
	Score    int64 `json:"score"`
}

type GloryArenaMatchRequest struct {
	PlayerId       int64
	PoolVersion    string
	WinCount       int32
	SelfPower      int64 // reserved
	DefeatedSet    map[int64]bool
	LastOpponents  []int64
	NeedCount      int
	ForceDifferent bool
}

type GameGloryArenaService struct{}

func NewGloryArenaService() *GameGloryArenaService {
	return &GameGloryArenaService{}
}

// GetChallengeList selects opponents by rank-percent window from pool zset.
// Rank percent is based on leaderboard order (high score first), so the
// underlying zset must be read in reverse score order.
func (s *GameGloryArenaService) GetChallengeList(req *GloryArenaMatchRequest) ([]*GloryArenaPoolMember, string, error) {
	if req == nil {
		return nil, "", ErrGloryArenaPoolDataInvalid
	}
	needCount := req.NeedCount
	if needCount <= 0 {
		needCount = DefaultGloryArenaMatchCount
	}
	if needCount > DefaultGloryArenaMatchCount {
		needCount = DefaultGloryArenaMatchCount
	}

	ctx := context.Background()
	poolVersion := req.PoolVersion
	if req.PoolVersion == "" {
		return nil, "", ErrGloryArenaPoolDataInvalid
	}
	key := enum.GetGloryArenaPoolOpponentRoundKey(req.PoolVersion)

	total, err := getRealPoolSize(ctx, key)
	if err != nil {
		logger.ErrorBySprintf("[gloryArenaService] load pool size failed version:%s err:%v", poolVersion, err)
		return nil, "", ErrGloryArenaPoolDataEmpty
	}
	if total < gloryArenaMinPoolMemberForMatch {
		return nil, poolVersion, ErrGloryArenaPoolMemberTooFew
	}

	low, high := getRankPercentByWinCount(req.WinCount)
	rangeMembers, err := loadPoolMembersByPercentRange(ctx, key, total, low, high)
	if err != nil {
		return nil, "", err
	}

	forceDifferent := req.ForceDifferent && len(req.LastOpponents) > 0
	battleLow, battleHigh, useBattleLimit := getBattlePercentByWinCount(req.WinCount)
	result := make([]*GloryArenaPoolMember, 0, needCount)

	result = s.appendByRule(result, rangeMembers, req, needCount, forceDifferent, useBattleLimit, battleLow, battleHigh, false)

	// 6~9胜：匹配不到时，回退到前10%（由配置给出）
	if len(result) < needCount && req.WinCount >= 5 && req.WinCount <= 8 {
		fallbackLow, fallbackHigh, ok := getTopRankPercentByConfig()
		if ok {
			fallbackMembers, e := loadPoolMembersByPercentRange(ctx, key, total, fallbackLow, fallbackHigh)
			if e != nil {
				return nil, "", e
			}
			result = s.appendByRule(result, fallbackMembers, req, needCount, forceDifferent, false, 0, 0, false)
		}
	}

	// 10~12胜：匹配不到时，按战力名次向上补齐（同区间去掉战力限制，按名次顺序取）
	if len(result) < needCount && req.WinCount >= 9 && req.WinCount <= 11 {
		result = s.appendByRule(result, rangeMembers, req, needCount, forceDifferent, true, highWinFallbackBattleLow, highWinFallbackBattleHigh, true)
	}

	// 兜底：扩到全池
	if len(result) < needCount {
		allMembers, e := loadPoolMembersByPercentRange(ctx, key, total, 0, 1)
		if e != nil {
			return nil, "", e
		}
		useAllPoolBattleLimit := false
		allPoolBattleLow := int32(0)
		allPoolBattleHigh := int32(0)
		if req.WinCount >= 9 && req.WinCount <= 11 {
			useAllPoolBattleLimit = true
			allPoolBattleLow = highWinFallbackBattleLow
			allPoolBattleHigh = highWinFallbackBattleHigh
		}
		result = s.appendByRule(result, allMembers, req, needCount, forceDifferent, useAllPoolBattleLimit, allPoolBattleLow, allPoolBattleHigh, false)
	}

	if len(result) < needCount {
		return nil, poolVersion, ErrGloryArenaPoolMemberTooFew
	}
	return result, poolVersion, nil
}

// LoadChallengePlayerInfos batch-loads player basic/battle redis info by ids.
func (s *GameGloryArenaService) LoadChallengePlayerInfos(opponentIDs []int64) map[int64]*logicCommon.PlayerRedisInfo {
	return logicCommon.GetPlayerRedisInfos(opponentIDs)
}

func (s *GameGloryArenaService) GetSeasonTypeBySeasonID(seasonID int32) enum.GloryArenaSeasonType {
	return enum.GetGloryArenaSeasonTypeBySeasonId(seasonID)
}

func (s *GameGloryArenaService) GetOpsStateByServerID(serverID int32) (*logicCommon.GloryArenaOpsServerState, error) {
	if serverID <= 0 {
		return nil, nil
	}

	rawState, err := dbService.RDB.HGet(context.Background(), enum.GetGloryArenaOpsStateKey(), strconv.FormatInt(int64(serverID), 10)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if rawState == "" {
		return nil, nil
	}

	state := &logicCommon.GloryArenaOpsServerState{}
	if err = json.Unmarshal([]byte(rawState), state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *GameGloryArenaService) pickCandidates(members []*GloryArenaPoolMember, req *GloryArenaMatchRequest, needCount int, ignoreLastGroup bool, allowRepeat bool) []*GloryArenaPoolMember {
	return s.pickCandidatesByRule(members, req, needCount, ignoreLastGroup, allowRepeat, false, 0, 0, false)
}

func (s *GameGloryArenaService) pickCandidatesByRule(members []*GloryArenaPoolMember, req *GloryArenaMatchRequest, needCount int, ignoreLastGroup bool, allowRepeat bool, useBattleLimit bool, battleLow int32, battleHigh int32, keepRankOrder bool) []*GloryArenaPoolMember {
	filtered := s.filterCandidatesByRule(members, req, ignoreLastGroup, allowRepeat, useBattleLimit, battleLow, battleHigh)
	if len(filtered) <= needCount {
		return filtered
	}
	if keepRankOrder {
		return filtered[:needCount]
	}

	for i := len(filtered) - 1; i > 0; i-- {
		j := tool.RandInt(0, i)
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}
	return filtered[:needCount]
}

func (s *GameGloryArenaService) filterCandidatesByRule(members []*GloryArenaPoolMember, req *GloryArenaMatchRequest, ignoreLastGroup bool, allowRepeat bool, useBattleLimit bool, battleLow int32, battleHigh int32) []*GloryArenaPoolMember {
	if req == nil || len(members) == 0 {
		return nil
	}

	lastSet := make(map[int64]bool, len(req.LastOpponents))
	for _, id := range req.LastOpponents {
		if id > 0 {
			lastSet[id] = true
		}
	}

	filtered := make([]*GloryArenaPoolMember, 0, len(members))
	uniq := make(map[int64]bool, len(members))
	for _, item := range members {
		if item == nil || item.PlayerId <= 0 {
			continue
		}
		if item.PlayerId == req.PlayerId {
			continue
		}
		if uniq[item.PlayerId] {
			continue
		}
		if !matchBattleRange(req.SelfPower, item.Score, useBattleLimit, battleLow, battleHigh) {
			continue
		}
		if !allowRepeat {
			if req.DefeatedSet != nil && req.DefeatedSet[item.PlayerId] {
				continue
			}
			if !ignoreLastGroup && lastSet[item.PlayerId] {
				continue
			}
		}
		uniq[item.PlayerId] = true
		filtered = append(filtered, item)
	}
	return filtered
}

func (s *GameGloryArenaService) appendByRule(dst []*GloryArenaPoolMember, members []*GloryArenaPoolMember, req *GloryArenaMatchRequest, needCount int, forceDifferent bool, useBattleLimit bool, battleLow int32, battleHigh int32, keepRankOrder bool) []*GloryArenaPoolMember {
	if len(dst) >= needCount || len(members) == 0 {
		return dst
	}

	dst = appendUniqueMembers(dst, s.pickCandidatesByRule(members, req, needCount-len(dst), false, false, useBattleLimit, battleLow, battleHigh, keepRankOrder), needCount)
	if len(dst) >= needCount {
		return dst
	}

	allowIgnoreLast := true
	if forceDifferent {
		count := len(s.filterCandidatesByRule(members, req, true, false, useBattleLimit, battleLow, battleHigh))
		if count >= needCount*2 {
			allowIgnoreLast = false
		}
	}
	if allowIgnoreLast {
		dst = appendUniqueMembers(dst, s.pickCandidatesByRule(members, req, needCount-len(dst), true, false, useBattleLimit, battleLow, battleHigh, keepRankOrder), needCount)
		if len(dst) >= needCount {
			return dst
		}
	}

	allowRepeatWithIgnoreLast := true
	if forceDifferent {
		count := len(s.filterCandidatesByRule(members, req, true, true, useBattleLimit, battleLow, battleHigh))
		if count >= needCount*2 {
			allowRepeatWithIgnoreLast = false
		}
	}
	if allowRepeatWithIgnoreLast {
		dst = appendUniqueMembers(dst, s.pickCandidatesByRule(members, req, needCount-len(dst), true, true, useBattleLimit, battleLow, battleHigh, keepRankOrder), needCount)
	}
	return dst
}

func parsePoolMembersFromZSet(rawMembers []redis.Z) []*GloryArenaPoolMember {
	result := make([]*GloryArenaPoolMember, 0, len(rawMembers))
	dup := make(map[int64]bool, len(rawMembers))
	for _, raw := range rawMembers {
		member, ok := raw.Member.(string)
		if !ok || member == "" {
			continue
		}
		if member == gloryArenaPoolInitMember {
			continue
		}
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil || id <= 0 || dup[id] {
			continue
		}
		dup[id] = true
		result = append(result, &GloryArenaPoolMember{
			PlayerId: id,
			Score:    int64(raw.Score),
		})
	}
	return result
}

func loadPoolMembersByPercentRange(ctx context.Context, key string, total int64, low float64, high float64) ([]*GloryArenaPoolMember, error) {
	if total <= 0 {
		return nil, ErrGloryArenaPoolDataEmpty
	}
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	if low > high {
		low, high = high, low
	}

	start := int64(math.Floor(float64(total) * low))
	endExclusive := int64(math.Ceil(float64(total) * high))
	if endExclusive <= start {
		return nil, nil
	}
	if start < 0 {
		start = 0
	}
	if endExclusive > total {
		endExclusive = total
	}
	end := endExclusive - 1
	if end < start {
		return nil, nil
	}

	rawMembers, err := dbService.RDB.ZRevRangeWithScores(ctx, key, start, end).Result()
	if err != nil {
		logger.ErrorBySprintf("[gloryArenaService] zrevrange failed key:%s start:%d end:%d err:%v", key, start, end, err)
		return nil, ErrGloryArenaPoolDataEmpty
	}
	return parsePoolMembersFromZSet(rawMembers), nil
}

func getRealPoolSize(ctx context.Context, key string) (int64, error) {
	total, err := dbService.RDB.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	_, err = dbService.RDB.ZScore(ctx, key, gloryArenaPoolInitMember).Result()
	if err == nil {
		if total > 0 {
			total--
		}
		return total, nil
	}
	if err == redis.Nil {
		return total, nil
	}
	return 0, err
}

func getRankPercentByWinCount(winCount int32) (float64, float64) {
	cfg := gameConfig.GetGloryArenaBaseCfg(winCount)
	if cfg != nil && len(cfg.Rank) == 2 {
		low := cfg.Rank[0]
		high := cfg.Rank[1]
		if low >= 0 && low <= 10000 && high >= 0 && high <= 10000 && low <= high {
			return float64(low) / 10000.0, float64(high) / 10000.0
		}
	}
	return 0, 1.0
}

func getBattlePercentByWinCount(winCount int32) (int32, int32, bool) {
	cfg := gameConfig.GetGloryArenaBaseCfg(winCount)
	if cfg == nil || len(cfg.Battle) != 2 {
		return 0, 0, false
	}
	low := cfg.Battle[0]
	high := cfg.Battle[1]
	if low <= 0 || high <= 0 || low > high {
		return 0, 0, false
	}
	return low, high, true
}

func getTopRankPercentByConfig() (float64, float64, bool) {
	cfg := gameConfig.GetGloryArenaBaseCfg(9)
	if cfg == nil || len(cfg.Rank) != 2 {
		return 0, 0, false
	}
	low := cfg.Rank[0]
	high := cfg.Rank[1]
	if low < 0 || low > 10000 || high < 0 || high > 10000 || low > high {
		return 0, 0, false
	}
	return float64(low) / 10000.0, float64(high) / 10000.0, true
}

func matchBattleRange(selfPower int64, candidateScore int64, useBattleLimit bool, battleLow int32, battleHigh int32) bool {
	if !useBattleLimit {
		return true
	}
	if selfPower <= 0 || candidateScore <= 0 {
		return true
	}
	minPower := selfPower * int64(battleLow) / 10000
	maxPower := selfPower * int64(battleHigh) / 10000
	return candidateScore >= minPower && candidateScore <= maxPower
}

func appendUniqueMembers(dst []*GloryArenaPoolMember, src []*GloryArenaPoolMember, max int) []*GloryArenaPoolMember {
	if max <= 0 || len(src) == 0 {
		return dst
	}
	dup := make(map[int64]bool, len(dst))
	for _, item := range dst {
		if item != nil && item.PlayerId > 0 {
			dup[item.PlayerId] = true
		}
	}
	for _, item := range src {
		if len(dst) >= max {
			break
		}
		if item == nil || item.PlayerId <= 0 || dup[item.PlayerId] {
			continue
		}
		dup[item.PlayerId] = true
		dst = append(dst, item)
	}
	return dst
}
