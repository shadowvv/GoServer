package model

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

const (
	GloryArenaDefaultLife = 3
	GloryArenaMaxWin      = 12
)

const (
	GloryArenaRoundStatusNotEnrolled int32 = 0
	GloryArenaRoundStatusEnrolled    int32 = 1
	GloryArenaRoundStatusFinished    int32 = 2
)

// PlayerGloryArenaEntity 只保存玩家个人进度态，不保存全服/公共匹配池状态。
type PlayerGloryArenaEntity struct {
	UserId         int64  `gorm:"column:user_id;primaryKey"` // 玩家ID（主键）
	SeasonId       int32  `gorm:"column:season_id"`          // 当前赛季ID（玩家视角快照）
	RoundId        int32  `gorm:"column:round_id"`           // 当前轮次ID（玩家视角快照）
	PoolVersion    string `gorm:"column:pool_version"`       // 当前参与的匹配池版本号（与 ops_state.groupVersion 一致）
	EnrollCount    int32  `gorm:"column:enroll_count"`       // 本轮累计报名次数（首免后用于门票消耗）
	Status         int32  `gorm:"column:status"`             // 当前报名状态（0=未报名，1=已报名进行中，2=已结束）
	WinCount       int32  `gorm:"column:win_count"`          // 当前报名胜场数
	Life           int32  `gorm:"column:life"`               // 当前报名剩余生命值
	RoundBestWin   int32  `gorm:"column:round_best_win"`     // 本轮内历史最高胜场
	RoundWinCount  string `gorm:"column:round_win_count"`
	RoundGotBox    int32  `gorm:"column:round_got_box"`    // 本轮已领奖个数（按轮重置）
	SeasonTotalWin int32  `gorm:"column:season_total_win"` // 当前赛季累计胜场
	LastMatchGroup string `gorm:"column:last_match_group"` // 当前三选一对手缓存（JSON数组，元素为PlayerRedisInfo）
	DefeatedCache  string `gorm:"column:defeated_cache"`   // 3/6/9 胜时缓存的被击败玩家信息（JSON对象）
	DefeatedSet    string `gorm:"column:defeated_set"`     // 当前报名已战胜对手集合（JSON数组）

	DefeatedSetData    map[int64]bool                   `gorm:"-"` // DefeatedSet 的运行态结构
	LastMatchGroupData []*logicCommon.PlayerRedisInfo   `gorm:"-"` // LastMatchGroup 的运行态结构
	DefeatedCacheData  *GloryArenaDefeatedCacheSnapshot `gorm:"-"` // DefeatedCache 的运行态结构
	RoundWinCountData  []int32                          `gorm:"-"` // RoundWinCount 的运行态结构（固定 12 位）
}

func (e *PlayerGloryArenaEntity) TableName() string {
	return "player_glory_arena_data"
}

type GloryArenaDefeatedCacheSnapshot struct {
	OpponentId int64                        `json:"opponentId"`
	SlotId     int32                        `json:"slotId,omitempty"`
	Heroes     []*logicCommon.HeroBasicInfo `json:"heroes"`
}

// PlayerGloryArenaSelectedOpponentEntity 保存“已选择英雄”快照。
type PlayerGloryArenaSelectedOpponentEntity struct {
	UserId      int64  `gorm:"column:user_id;primaryKey"` // 玩家ID（联合主键）
	SlotId      int32  `gorm:"column:slot_id;primaryKey"` // 3/6/9 槽位ID（联合主键）
	PoolVersion string `gorm:"column:pool_version"`       // 选择时的匹配池版本
	LineupBlob  string `gorm:"column:lineup_blob"`        // 已选择英雄快照（HonorArenaHeroInfo JSON）
	SelectedAt  int64  `gorm:"column:selected_at"`        // 选择时间（毫秒时间戳）

	SelectedHero *logicCommon.HeroBasicInfo `gorm:"-"` // LineupBlob 反序列化后的运行态结构
}

func (e *PlayerGloryArenaSelectedOpponentEntity) TableName() string {
	return "player_glory_arena_selected_opponent"
}

type PlayerGloryArenaModel struct {
	Player *PlayerModel
	Entity *PlayerGloryArenaEntity

	SelectedHeroes map[int32]*PlayerGloryArenaSelectedOpponentEntity

	Changed             map[string]interface{}
	SelectedHeroDeleted map[int32]bool
	lastOpsState        *logicCommon.GloryArenaOpsServerState
	// TODO:unlock临时处理
	IsLose     bool
	EnterCount int
}

var _ logicCommon.PlayerModelInterface = (*PlayerGloryArenaModel)(nil)

func (p *PlayerGloryArenaModel) SaveModelToDB() {
	if len(p.Changed) > 0 {
		easyDB.UpdatePlayerEntity(p.Entity, p.Changed, p.Player.GetUserId())
		p.Changed = make(map[string]interface{})
	}

	for slot := range p.SelectedHeroDeleted {
		_ = easyDB.DeletePlayerEntityByWhere[PlayerGloryArenaSelectedOpponentEntity](map[string]interface{}{
			"user_id": p.Player.GetUserId(),
			"slot_id": slot,
		}, p.Player.GetUserId())
	}
	p.SelectedHeroDeleted = make(map[int32]bool)
}

func (p *PlayerGloryArenaModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if p == nil || p.Player == nil || p.Player.User == nil {
		return
	}
	p.syncAllianceRoundBestWinToRedis()
}

// ForceSyncRoundState 供关键请求路径调用，避免心跳限频导致轮次状态滞后。
func (p *PlayerGloryArenaModel) ForceSyncRoundState(currentTime int64) {
	if p == nil || p.Player == nil || p.Player.User == nil {
		return
	}
	if currentTime <= 0 {
		currentTime = tool.UnixNowMilli()
	}
	p.syncRoundStateByOps(currentTime)
}

func (p *PlayerGloryArenaModel) SetPoolVersion(version string) {
	if p.Entity.PoolVersion == version {
		return
	}
	p.Entity.PoolVersion = version
	p.Changed["pool_version"] = version
	if p.Player != nil {
		p.Player.UpdatePlayerBasicInfoToRedis()
	}
}

func (p *PlayerGloryArenaModel) SetSeasonTypeRound(seasonType enum.GloryArenaSeasonType, roundId int32) {
	seasonTypeValue := normalizeGloryArenaSeasonType(int32(seasonType))
	if p.Entity.SeasonId != seasonTypeValue {
		p.Entity.SeasonId = seasonTypeValue
		p.Changed["season_id"] = seasonTypeValue
	}
	if p.Entity.RoundId != roundId {
		p.Entity.RoundId = roundId
		p.Changed["round_id"] = roundId
	}
}

// SetSeasonRound keeps backward compatibility for legacy callers.
// legacy rule: odd season id => preseason, even season id => postseason.
func (p *PlayerGloryArenaModel) SetSeasonRound(seasonId int32, roundId int32) {
	p.SetSeasonTypeRound(enum.GetGloryArenaSeasonTypeBySeasonId(seasonId), roundId)
}

func (p *PlayerGloryArenaModel) SetEnrollStatus(enrolled bool) {
	if enrolled {
		p.resetCurrentEnrollProgress()
		p.setRoundStatus(GloryArenaRoundStatusEnrolled)
		return
	}
	p.setRoundStatus(GloryArenaRoundStatusNotEnrolled)
}

func (p *PlayerGloryArenaModel) AddEnrollCount(delta int32) {
	if delta == 0 {
		return
	}
	p.Entity.EnrollCount += delta
	if p.Entity.EnrollCount < 0 {
		p.Entity.EnrollCount = 0
	}
	p.Changed["enroll_count"] = p.Entity.EnrollCount
}

func (p *PlayerGloryArenaModel) SetLife(life int32) {
	if life < 0 {
		life = 0
	}
	if p.Entity.Life == life {
		return
	}
	p.Entity.Life = life
	p.Changed["life"] = life
	if life == 0 {
		p.setFinished(true)
	}
}

func (p *PlayerGloryArenaModel) AddWin(delta int32) {
	if delta == 0 {
		return
	}
	needNotifyRank := false
	oldWin := p.Entity.WinCount
	oldRoundBestWin := p.Entity.RoundBestWin
	p.Entity.WinCount += delta
	if p.Entity.WinCount < 0 {
		p.Entity.WinCount = 0
	}
	if p.Entity.WinCount > GloryArenaMaxWin {
		p.Entity.WinCount = GloryArenaMaxWin
	}
	p.Changed["win_count"] = p.Entity.WinCount
	if p.Entity.WinCount > p.Entity.RoundBestWin {
		p.Entity.RoundBestWin = p.Entity.WinCount
		p.Changed["round_best_win"] = p.Entity.RoundBestWin
		needNotifyRank = true
		if p.Player != nil {
			p.Player.UpdatePlayerBasicInfoToRedis()
		}
	}
	roundBestDelta := p.Entity.RoundBestWin - oldRoundBestWin
	realDelta := p.Entity.WinCount - oldWin
	if realDelta > 0 {
		p.addRoundWinCountByWinRange(oldWin+1, p.Entity.WinCount)
		p.Entity.SeasonTotalWin += realDelta
		if p.Entity.SeasonTotalWin < 0 {
			p.Entity.SeasonTotalWin = 0
		}
		p.Changed["season_total_win"] = p.Entity.SeasonTotalWin
		needNotifyRank = true
	}
	if needNotifyRank {
		p.notifyGloryArenaWinCountRankUpdate(roundBestDelta)
		if roundBestDelta != 0 {
			p.syncAllianceRoundBestWinToRedis()
		}
	}
	if p.Entity.WinCount >= GloryArenaMaxWin {
		p.setFinished(true)
	}
}

func (p *PlayerGloryArenaModel) addRoundWinCountByWinRange(startWin int32, endWin int32) {
	if p == nil || p.Entity == nil || startWin > endWin {
		return
	}
	if startWin < 1 {
		startWin = 1
	}
	if endWin > GloryArenaMaxWin {
		endWin = GloryArenaMaxWin
	}
	if startWin > endWin {
		return
	}
	if len(p.Entity.RoundWinCountData) != GloryArenaMaxWin {
		p.Entity.RoundWinCountData = normalizeGloryArenaRoundWinCountData(p.Entity.RoundWinCountData)
	}
	for win := startWin; win <= endWin; win++ {
		p.Entity.RoundWinCountData[win-1] += 1
	}
	p.Entity.RoundWinCount = marshalGloryArenaRoundWinCount(p.Entity.RoundWinCountData)
	p.Changed["round_win_count"] = p.Entity.RoundWinCount
}

func (p *PlayerGloryArenaModel) notifyGloryArenaWinCountRankUpdate(roundBestDelta int32) {
	if p == nil || p.Player == nil {
		return
	}
	roundVersion := p.Entity.PoolVersion
	seasonVersion := p.GetSeasonVersion()
	if roundVersion == "" && seasonVersion == "" {
		return
	}

	commonRankConfigs := gameConfig.GetAllRankCfg()
	for _, rankCfgMap := range commonRankConfigs {
		for _, rankCfg := range rankCfgMap {
			if rankCfg == nil || rankCfg.ActId != 0 {
				continue
			}

			var (
				score             int64
				version           string
				incrementalUpdate bool
			)

			switch rankCfg.PointType {
			case int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT):
				if roundVersion == "" {
					continue
				}
				score = int64(p.Entity.RoundBestWin)
				version = roundVersion
			case int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT):
				if seasonVersion == "" {
					continue
				}
				score = int64(p.Entity.SeasonTotalWin)
				version = seasonVersion
			case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT):
				if roundVersion == "" || roundBestDelta <= 0 {
					continue
				}
				score = int64(roundBestDelta)
				version = roundVersion
				incrementalUpdate = true
			default:
				continue
			}
			if score <= 0 {
				continue
			}

			rankId, err := logicCommon.GetRankUniqueId(rankCfg.Id, 0, 0, p.Player.GetUserServerId(), version)
			if err != nil {
				logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
			}
			updateRankReq := &rpcPb.NotifyUpdateRankInfo{
				Id:                p.Player.GetUserId(),
				Score:             score,
				IncrementalUpdate: incrementalUpdate,
			}
			if err := rpcMessageSender.SendMessageToRankBoard(p.Player.GetUserId(), rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, updateRankReq); err != nil {
				logger.ErrorBySprintf("[playerGloryArenaModel] update win count rank failed userId:%d rankId:%s pointType:%d err:%v", p.Player.GetUserId(), rankId, rankCfg.PointType, err)
			}
		}
	}
}

func (p *PlayerGloryArenaModel) syncAllianceRoundBestWinToRedis() {
	if p == nil || p.Player == nil || p.Entity == nil {
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(p.Player.GetUserId())
	if allianceInfo == nil {
		return
	}
	if allianceInfo.RoundBestWin == p.Entity.RoundBestWin {
		return
	}
	allianceInfo.RoundBestWin = p.Entity.RoundBestWin
	logicCommon.UpdatePlayerAllianceInfo(allianceInfo)
}

func (p *PlayerGloryArenaModel) SetDefeatedSet(opponentIds []int64) {
	p.Entity.DefeatedSetData = make(map[int64]bool)
	for _, id := range opponentIds {
		if id <= 0 {
			continue
		}
		p.Entity.DefeatedSetData[id] = true
	}
	p.Entity.DefeatedSet = marshalInt64Set(p.Entity.DefeatedSetData)
	p.Changed["defeated_set"] = p.Entity.DefeatedSet
}

func (p *PlayerGloryArenaModel) AddDefeatedOpponent(opponentId int64) {
	if opponentId <= 0 {
		return
	}
	if p.Entity.DefeatedSetData == nil {
		p.Entity.DefeatedSetData = make(map[int64]bool)
	}
	if p.Entity.DefeatedSetData[opponentId] {
		return
	}
	p.Entity.DefeatedSetData[opponentId] = true
	p.Entity.DefeatedSet = marshalInt64Set(p.Entity.DefeatedSetData)
	p.Changed["defeated_set"] = p.Entity.DefeatedSet
}

func (p *PlayerGloryArenaModel) SetLastMatchGroup(opponentIds []int64) {
	p.Entity.LastMatchGroupData = buildGloryArenaCachedPlayers(opponentIds, nil)
	p.Entity.LastMatchGroup = marshalPlayerRedisInfoList(p.Entity.LastMatchGroupData)
	p.Changed["last_match_group"] = p.Entity.LastMatchGroup
}

func (p *PlayerGloryArenaModel) SetLastMatchGroupWithInfos(opponentIds []int64, infos map[int64]*logicCommon.PlayerRedisInfo) {
	p.Entity.LastMatchGroupData = buildGloryArenaCachedPlayers(opponentIds, infos)
	p.Entity.LastMatchGroup = marshalPlayerRedisInfoList(p.Entity.LastMatchGroupData)
	p.Changed["last_match_group"] = p.Entity.LastMatchGroup
}

// SetCurrentMatchCandidates 保存当前三选一候选对手（缓存基础信息 + 主阵容战斗信息）。
func (p *PlayerGloryArenaModel) SetCurrentMatchCandidates(opponentIds []int64, infos map[int64]*logicCommon.PlayerRedisInfo) {
	p.SetLastMatchGroupWithInfos(opponentIds, infos)
}

// GetCurrentMatchCandidates 返回当前三选一候选对手ID。
func (p *PlayerGloryArenaModel) GetCurrentMatchCandidates() []int64 {
	return p.GetLastMatchGroup()
}

func (p *PlayerGloryArenaModel) TrySettleBattle(opponentId int64, win bool) (bool, error) {
	if win {
		p.AddWin(1)
		p.AddDefeatedOpponent(opponentId)
		p.IsLose = false
	} else {
		p.SetLife(p.Entity.Life - 1)
		p.IsLose = true
	}
	if p.Entity.WinCount >= GloryArenaMaxWin || p.Entity.Life <= 0 {
		p.setFinished(true)
	}
	return !p.IsEnrolled(), nil
}

func (p *PlayerGloryArenaModel) ResetRoundProgress(poolVersion string) {
	p.IsLose = false
	p.Entity.PoolVersion = poolVersion
	p.Entity.EnrollCount = 0
	p.Entity.Status = GloryArenaRoundStatusNotEnrolled
	p.Entity.WinCount = 0
	p.Entity.Life = getGloryArenaInitLife()
	p.Entity.RoundBestWin = 0
	p.Entity.RoundWinCountData = normalizeGloryArenaRoundWinCountData(nil)
	p.Entity.RoundWinCount = marshalGloryArenaRoundWinCount(p.Entity.RoundWinCountData)
	p.Entity.RoundGotBox = 0
	p.Entity.DefeatedSetData = make(map[int64]bool)
	p.Entity.DefeatedSet = marshalInt64Set(p.Entity.DefeatedSetData)
	p.Entity.LastMatchGroupData = make([]*logicCommon.PlayerRedisInfo, 0)
	p.Entity.LastMatchGroup = marshalPlayerRedisInfoList(p.Entity.LastMatchGroupData)
	p.Entity.DefeatedCacheData = nil
	p.Entity.DefeatedCache = marshalGloryArenaDefeatedCache(p.Entity.DefeatedCacheData)
	p.Changed["pool_version"] = p.Entity.PoolVersion
	p.Changed["enroll_count"] = p.Entity.EnrollCount
	p.Changed["status"] = p.Entity.Status
	p.Changed["win_count"] = p.Entity.WinCount
	p.Changed["life"] = p.Entity.Life
	p.Changed["round_best_win"] = p.Entity.RoundBestWin
	p.Changed["round_win_count"] = p.Entity.RoundWinCount
	p.Changed["round_got_box"] = p.Entity.RoundGotBox
	p.Changed["defeated_set"] = p.Entity.DefeatedSet
	p.Changed["last_match_group"] = p.Entity.LastMatchGroup
	p.Changed["defeated_cache"] = p.Entity.DefeatedCache
	if p.Player != nil {
		p.Player.UpdatePlayerBasicInfoToRedis()
	}
	p.syncAllianceRoundBestWinToRedis()
	p.ClearSelectedHeroes()
}

func (p *PlayerGloryArenaModel) ResetSeasonProgress() {
	p.ResetRoundProgress(p.Entity.PoolVersion)
	p.Entity.SeasonTotalWin = 0
	p.Changed["season_total_win"] = p.Entity.SeasonTotalWin
}

func (p *PlayerGloryArenaModel) syncRoundStateByOps(currentTime int64) {
	state, err := p.loadOpsStateByServerID(p.Player.GetUserServerId())
	if err != nil {
		logger.ErrorBySprintf("[playerGloryArenaModel] load ops state failed userId:%d serverId:%d err:%v", p.Player.GetUserId(), p.Player.GetUserServerId(), err)
		return
	}
	if state == nil {
		return
	}

	poolVersion := state.GroupVersion
	if poolVersion == "" {
		return
	}
	oldSeason := p.Entity.SeasonId
	oldRound := p.Entity.RoundId
	oldVersion := p.Entity.PoolVersion
	newSeason := int32(state.SeasonType)
	newRound := state.RoundIndexInSeason

	if shouldIgnoreRoundStateRollback(p.lastOpsState, state, oldSeason, newSeason, oldRound, newRound, oldVersion, poolVersion) {
		logger.ErrorBySprintf("[playerGloryArenaModel] ignore stale round state rollback userId:%d serverId:%d oldSeason:%d oldRound:%d oldVersion:%s newSeason:%d newRound:%d newVersion:%s oldStart:%d newStart:%d now:%d",
			p.Player.GetUserId(),
			p.Player.GetUserServerId(),
			oldSeason,
			oldRound,
			oldVersion,
			newSeason,
			newRound,
			poolVersion,
			p.lastOpsState.RoundStart,
			state.RoundStart,
			currentTime,
		)
		return
	}

	p.lastOpsState = state

	if oldSeason != newSeason || oldRound != newRound || oldVersion != poolVersion {
		if oldSeason != newSeason || shouldResetSeasonOnRoundSwitch(oldSeason, oldRound, newRound) {
			p.ResetSeasonProgress()
		} else {
			p.ResetRoundProgress(poolVersion)
		}
		p.SetSeasonTypeRound(state.SeasonType, newRound)
		p.SetPoolVersion(poolVersion)
		return
	}

	// 同轮次内按天推进池版本。
	p.SetPoolVersion(poolVersion)
}

func (p *PlayerGloryArenaModel) loadOpsStateByServerID(serverID int32) (*logicCommon.GloryArenaOpsServerState, error) {
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

func shouldResetSeasonOnRoundSwitch(seasonType int32, oldRound int32, newRound int32) bool {
	// 季后赛同类型跨季时（4 -> 1/0）需要清赛季累计数据。
	return seasonType == int32(enum.GLORY_ARENA_SEASON_TYPE_POST) &&
		oldRound == 4 &&
		(newRound == 1 || newRound == 0)
}

func shouldIgnoreRoundStateRollback(prevState *logicCommon.GloryArenaOpsServerState, state *logicCommon.GloryArenaOpsServerState, oldSeason int32, newSeason int32, oldRound int32, newRound int32, oldVersion string, newVersion string) bool {
	if prevState == nil || state == nil {
		return false
	}
	if oldSeason != newSeason || oldRound == newRound {
		return false
	}
	if oldVersion != newVersion {
		return false
	}
	if int32(prevState.SeasonType) != oldSeason || prevState.GroupVersion != oldVersion {
		return false
	}
	if prevState.RoundStart <= 0 || state.RoundStart <= 0 {
		return false
	}
	return state.RoundStart < prevState.RoundStart
}

func (p *PlayerGloryArenaModel) IsFinished() bool {
	return p.Entity.Status == GloryArenaRoundStatusFinished
}

func (p *PlayerGloryArenaModel) GetLife() int32 {
	return p.Entity.Life
}

func (p *PlayerGloryArenaModel) GetWinCount() int32 {
	return p.Entity.WinCount
}

func (p *PlayerGloryArenaModel) GetRoundWinCount() int32 {
	if p == nil || p.Entity == nil {
		return 0
	}
	win := p.GetWinCount() // 0~12
	if win <= 0 {
		return 0
	}
	idx := win - 1 // 第N场 -> index N-1
	if idx < 0 || idx >= int32(len(p.Entity.RoundWinCountData)) {
		return 0
	}
	return p.Entity.RoundWinCountData[idx]
}

// ShouldSaveSelectedOpponentSnapshot 当前胜场是否命中 3/6/9 快照点。
func (p *PlayerGloryArenaModel) ShouldSaveSelectedOpponentSnapshot() bool {
	return IsGloryArenaSnapshotWin(p.Entity.WinCount)
}

func (p *PlayerGloryArenaModel) GetPoolVersion() string {
	return p.Entity.PoolVersion
}

func (p *PlayerGloryArenaModel) GetSeasonVersion() string {
	if p == nil || p.Entity == nil {
		return ""
	}
	// Preseason has no season-win leaderboard version.
	if p.Entity.SeasonId == int32(enum.GLORY_ARENA_SEASON_TYPE_PRE) {
		return ""
	}
	if p != nil && p.lastOpsState != nil && p.lastOpsState.SeasonVersion != "" {
		if p.lastOpsState.SeasonType == enum.GLORY_ARENA_SEASON_TYPE_PRE {
			return ""
		}
		return p.lastOpsState.SeasonVersion
	}
	return logicCommon.GetGloryArenaSeasonVersion(p.Entity.PoolVersion)
}

func (p *PlayerGloryArenaModel) GetSeasonTotalWin() int32 {
	return p.Entity.SeasonTotalWin
}

func (p *PlayerGloryArenaModel) GetRoundBestWinCount() int32 {
	return p.Entity.RoundBestWin
}

func (p *PlayerGloryArenaModel) GetRoundWinCountData() []int32 {
	return append([]int32(nil), normalizeGloryArenaRoundWinCountData(p.Entity.RoundWinCountData)...)
}

func (p *PlayerGloryArenaModel) GetRoundGotBoxCount() int32 {
	return p.Entity.RoundGotBox
}

func (p *PlayerGloryArenaModel) GetCurrentRoundTimeWindow() (int64, int64) {
	if p.lastOpsState == nil {
		return 0, 0
	}
	return p.lastOpsState.RoundStart, p.lastOpsState.RoundEnd
}

func (p *PlayerGloryArenaModel) SetRoundGotBoxCount(count int32) {
	if count < 0 {
		count = 0
	}
	if p.Entity.RoundGotBox == count {
		return
	}
	p.Entity.RoundGotBox = count
	p.Changed["round_got_box"] = p.Entity.RoundGotBox
}

func (p *PlayerGloryArenaModel) AddRoundGotBoxCount(delta int32) {
	if delta == 0 {
		return
	}
	p.SetRoundGotBoxCount(p.Entity.RoundGotBox + delta)
}

func (p *PlayerGloryArenaModel) GetSeasonType() int32 {
	return p.Entity.SeasonId
}

func (p *PlayerGloryArenaModel) GetDefeatedSet() map[int64]bool {
	result := make(map[int64]bool, len(p.Entity.DefeatedSetData))
	for id, ok := range p.Entity.DefeatedSetData {
		result[id] = ok
	}
	return result
}

func (p *PlayerGloryArenaModel) GetLastMatchGroup() []int64 {
	result := make([]int64, 0, len(p.Entity.LastMatchGroupData))
	for _, info := range p.Entity.LastMatchGroupData {
		if info == nil || info.BasicInfo == nil || info.BasicInfo.Id <= 0 {
			continue
		}
		result = append(result, info.BasicInfo.Id)
	}
	return result
}

func (p *PlayerGloryArenaModel) GetCurrentMatchCandidateInfos() map[int64]*logicCommon.PlayerRedisInfo {
	result := make(map[int64]*logicCommon.PlayerRedisInfo, len(p.Entity.LastMatchGroupData))
	for _, info := range p.Entity.LastMatchGroupData {
		if info == nil || info.BasicInfo == nil || info.BasicInfo.Id <= 0 {
			continue
		}
		result[info.BasicInfo.Id] = info
	}
	return result
}

func (p *PlayerGloryArenaModel) IsEnrolled() bool {
	return p.Entity.Status == GloryArenaRoundStatusEnrolled
}

func (p *PlayerGloryArenaModel) SaveDefeatedOpponentSnapshotFromPlayerInfo(opponentID int64, info *logicCommon.PlayerRedisInfo) {
	if opponentID <= 0 || info == nil || info.BattleInfo == nil {
		return
	}
	slotId := getGloryArenaSnapshotSlotByWinCount(p.Entity.WinCount)
	if slotId <= 0 {
		return
	}
	heroes := extractMainFormationHeroSnapshot(info.BattleInfo)
	if len(heroes) == 0 {
		return
	}
	p.Entity.DefeatedCacheData = normalizeGloryArenaDefeatedCache(&GloryArenaDefeatedCacheSnapshot{
		OpponentId: opponentID,
		SlotId:     slotId,
		Heroes:     heroes,
	})
	p.Entity.DefeatedCache = marshalGloryArenaDefeatedCache(p.Entity.DefeatedCacheData)
	p.Changed["defeated_cache"] = p.Entity.DefeatedCache
}

func (p *PlayerGloryArenaModel) ClearDefeatedOpponentCache() {
	p.Entity.DefeatedCacheData = nil
	p.Entity.DefeatedCache = marshalGloryArenaDefeatedCache(p.Entity.DefeatedCacheData)
	p.Changed["defeated_cache"] = p.Entity.DefeatedCache
}

func (p *PlayerGloryArenaModel) BuildSelectableHeroes() []*pb.HonorArenaHeroInfo {
	return buildSelectableHeroesForSnapshot(p.Entity.DefeatedCacheData)
}

func (p *PlayerGloryArenaModel) FindSelectableHero(heroSelectID int64) *logicCommon.HeroBasicInfo {
	if heroSelectID <= 0 {
		return nil
	}
	return findSelectableHeroFromSnapshot(p.Entity.DefeatedCacheData, heroSelectID)
}

func (p *PlayerGloryArenaModel) UpsertSelectedHero(heroInfo *logicCommon.HeroBasicInfo) error {
	if heroInfo == nil {
		return errors.New("selected hero is nil")
	}
	slotId := p.getCurrentSnapshotSlotId()
	poolVersion := p.Entity.PoolVersion
	heroBlob, err := json.Marshal(heroInfo)
	if err != nil {
		return err
	}

	entity := p.SelectedHeroes[slotId]
	if entity != nil {
		entity.PoolVersion = poolVersion
		entity.LineupBlob = string(heroBlob)
		entity.SelectedAt = tool.UnixNowMilli()
		entity.SelectedHero = cloneHeroBasicInfo(heroInfo)
		p.SelectedHeroes[slotId] = entity
		delete(p.SelectedHeroDeleted, slotId)
		easyDB.UpdatePlayerEntity(entity, map[string]interface{}{
			"pool_version": entity.PoolVersion,
			"lineup_blob":  entity.LineupBlob,
			"selected_at":  entity.SelectedAt,
		}, p.Player.GetUserId())
		return nil
	}

	entity = &PlayerGloryArenaSelectedOpponentEntity{
		UserId:       p.Player.GetUserId(),
		SlotId:       slotId,
		PoolVersion:  poolVersion,
		LineupBlob:   string(heroBlob),
		SelectedAt:   tool.UnixNowMilli(),
		SelectedHero: cloneHeroBasicInfo(heroInfo),
	}
	if err = easyDB.CreatePlayerEntity(entity); err != nil {
		return err
	}
	p.SelectedHeroes[slotId] = entity
	delete(p.SelectedHeroDeleted, slotId)
	return nil
}

func (p *PlayerGloryArenaModel) ClearSelectedHeroes() {
	for slotId := range p.SelectedHeroes {
		p.SelectedHeroDeleted[slotId] = true
	}
	p.SelectedHeroes = make(map[int32]*PlayerGloryArenaSelectedOpponentEntity)
}

func (p *PlayerGloryArenaModel) getCurrentSnapshotSlotId() int32 {
	if p == nil || p.Entity == nil {
		return 1
	}
	slotId := int32(0)
	if p.Entity.DefeatedCacheData != nil {
		slotId = p.Entity.DefeatedCacheData.SlotId
	}
	if !isGloryArenaSnapshotSlotIDValid(slotId) {
		slotId = getGloryArenaSnapshotSlotByWinCount(p.Entity.WinCount)
	}
	if !isGloryArenaSnapshotSlotIDValid(slotId) {
		return 1
	}
	return slotId
}

func (p *PlayerGloryArenaModel) setFinished(finished bool) {
	if finished {
		p.setRoundStatus(GloryArenaRoundStatusNotEnrolled)
		return
	}
	p.setRoundStatus(GloryArenaRoundStatusEnrolled)
}

func (p *PlayerGloryArenaModel) setRoundStatus(status int32) {
	if p.Entity.Status == status {
		return
	}
	p.Entity.Status = status
	p.Changed["status"] = status
}

func (p *PlayerGloryArenaModel) resetCurrentEnrollProgress() {
	p.IsLose = false
	p.Entity.WinCount = 0
	p.Entity.Life = getGloryArenaInitLife()
	p.Entity.DefeatedSetData = make(map[int64]bool)
	p.Entity.DefeatedSet = marshalInt64Set(p.Entity.DefeatedSetData)
	p.Entity.LastMatchGroupData = make([]*logicCommon.PlayerRedisInfo, 0)
	p.Entity.LastMatchGroup = marshalPlayerRedisInfoList(p.Entity.LastMatchGroupData)
	p.Entity.DefeatedCacheData = nil
	p.Entity.DefeatedCache = marshalGloryArenaDefeatedCache(p.Entity.DefeatedCacheData)

	p.Changed["win_count"] = p.Entity.WinCount
	p.Changed["life"] = p.Entity.Life
	p.Changed["defeated_set"] = p.Entity.DefeatedSet
	p.Changed["last_match_group"] = p.Entity.LastMatchGroup
	p.Changed["defeated_cache"] = p.Entity.DefeatedCache
}

func (p *PlayerGloryArenaModel) BuildSelectedHeroes() []*pb.HonorArenaHeroInfo {
	all := make([]*pb.HonorArenaHeroInfo, 0)
	slotIds := make([]int32, 0, len(p.SelectedHeroes))
	for slotId := range p.SelectedHeroes {
		slotIds = append(slotIds, slotId)
	}
	sort.Slice(slotIds, func(i, j int) bool {
		return slotIds[i] < slotIds[j]
	})
	for _, slotId := range slotIds {
		h := p.SelectedHeroes[slotId]
		if h == nil || h.SelectedHero == nil || h.SelectedHero.Id <= 0 {
			continue
		}
		heroResp := &pb.HonorArenaHeroInfo{
			HeroId: h.SelectedHero.Uid,
			Cid:    int32(h.SelectedHero.Id),
			Level:  h.SelectedHero.Level,
			Star:   h.SelectedHero.Star,
			Job:    h.SelectedHero.ClassId,
			Fight:  h.SelectedHero.Attr[enum.AttributeBasicCombatPower],
		}
		all = append(all, heroResp)
	}
	return all
}

func (p *PlayerGloryArenaModel) CanFreeCompete() int32 {
	if p.Entity.EnrollCount == 0 {
		return 1
	}
	return 0
}

func NewPlayerGloryArenaModel(player *PlayerModel) *PlayerGloryArenaModel {
	roundWinCountData := normalizeGloryArenaRoundWinCountData(nil)
	entity := &PlayerGloryArenaEntity{
		UserId:             player.GetUserId(),
		SeasonId:           int32(enum.GLORY_ARENA_SEASON_TYPE_PRE),
		RoundId:            0,
		PoolVersion:        "",
		EnrollCount:        0,
		Status:             GloryArenaRoundStatusNotEnrolled,
		WinCount:           0,
		Life:               getGloryArenaInitLife(),
		RoundBestWin:       0,
		RoundWinCount:      marshalGloryArenaRoundWinCount(roundWinCountData),
		RoundWinCountData:  roundWinCountData,
		RoundGotBox:        0,
		SeasonTotalWin:     0,
		DefeatedSetData:    make(map[int64]bool),
		LastMatchGroupData: make([]*logicCommon.PlayerRedisInfo, 0),
		DefeatedCache:      "{}",
		DefeatedCacheData:  nil,
	}
	entity.DefeatedSet = marshalInt64Set(entity.DefeatedSetData)
	entity.LastMatchGroup = marshalPlayerRedisInfoList(entity.LastMatchGroupData)
	entity.DefeatedCache = marshalGloryArenaDefeatedCache(entity.DefeatedCacheData)
	return &PlayerGloryArenaModel{
		Player:              player,
		Entity:              entity,
		SelectedHeroes:      make(map[int32]*PlayerGloryArenaSelectedOpponentEntity),
		Changed:             make(map[string]interface{}),
		SelectedHeroDeleted: make(map[int32]bool),
	}
}

func LoadPlayerGloryArenaModel(player *PlayerModel) (*PlayerGloryArenaModel, error) {
	entity, err := easyDB.GetPlayerEntityByID[PlayerGloryArenaEntity](player.GetUserId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	entity.SeasonId = normalizeGloryArenaSeasonType(entity.SeasonId)
	entity.DefeatedSetData = unmarshalInt64Set(entity.DefeatedSet)
	entity.LastMatchGroupData = unmarshalPlayerRedisInfoList(entity.LastMatchGroup)
	entity.DefeatedCacheData = unmarshalGloryArenaDefeatedCache(entity.DefeatedCache)
	entity.RoundWinCountData = unmarshalGloryArenaRoundWinCount(entity.RoundWinCount)
	entity.RoundWinCount = marshalGloryArenaRoundWinCount(entity.RoundWinCountData)

	opponents, err := easyDB.GetPlayerEntitiesByWhere[PlayerGloryArenaSelectedOpponentEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	selectedHeroMap := make(map[int32]*PlayerGloryArenaSelectedOpponentEntity)
	for _, item := range opponents {
		if item == nil {
			continue
		}
		if len(item.LineupBlob) > 0 {
			_ = json.Unmarshal([]byte(item.LineupBlob), &item.SelectedHero)
		}
		selectedHeroMap[item.SlotId] = item
	}

	return &PlayerGloryArenaModel{
		Player:              player,
		Entity:              entity,
		SelectedHeroes:      selectedHeroMap,
		Changed:             make(map[string]interface{}),
		SelectedHeroDeleted: make(map[int32]bool),
	}, nil
}

func CreatePlayerGloryArenaModel(player *PlayerModel) (*PlayerGloryArenaModel, error) {
	model := NewPlayerGloryArenaModel(player)
	if err := easyDB.CreatePlayerEntity(model.Entity); err != nil {
		return nil, err
	}
	return model, nil
}

func marshalInt64Set(value map[int64]bool) string {
	list := make([]int64, 0, len(value))
	for id, ok := range value {
		if !ok || id <= 0 {
			continue
		}
		list = append(list, id)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return marshalInt64List(list)
}

func marshalInt64List(value []int64) string {
	data, err := json.Marshal(compactInt64List(value))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func unmarshalInt64Set(raw string) map[int64]bool {
	result := make(map[int64]bool)
	for _, id := range unmarshalInt64List(raw) {
		result[id] = true
	}
	return result
}

func unmarshalInt64List(raw string) []int64 {
	if raw == "" {
		return make([]int64, 0)
	}
	var list []int64
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return make([]int64, 0)
	}
	return compactInt64List(list)
}

func normalizeGloryArenaRoundWinCountData(value []int32) []int32 {
	result := make([]int32, GloryArenaMaxWin)
	for i := 0; i < len(value) && i < GloryArenaMaxWin; i++ {
		if value[i] > 0 {
			result[i] = value[i]
		}
	}
	return result
}

func marshalGloryArenaRoundWinCount(value []int32) string {
	normalized := normalizeGloryArenaRoundWinCountData(value)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func unmarshalGloryArenaRoundWinCount(raw string) []int32 {
	if raw == "" {
		return normalizeGloryArenaRoundWinCountData(nil)
	}
	var list []int32
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return normalizeGloryArenaRoundWinCountData(list)
	}

	parts := strings.Split(raw, "|")
	if len(parts) == 0 {
		return normalizeGloryArenaRoundWinCountData(nil)
	}
	list = make([]int32, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			list = append(list, 0)
			continue
		}
		count, err := strconv.ParseInt(part, 10, 32)
		if err != nil {
			return normalizeGloryArenaRoundWinCountData(nil)
		}
		list = append(list, int32(count))
	}
	return normalizeGloryArenaRoundWinCountData(list)
}

func marshalGloryArenaDefeatedCache(value *GloryArenaDefeatedCacheSnapshot) string {
	normalized := normalizeGloryArenaDefeatedCache(value)
	if normalized == nil {
		return "{}"
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func unmarshalGloryArenaDefeatedCache(raw string) *GloryArenaDefeatedCacheSnapshot {
	if raw == "" {
		return nil
	}
	var result GloryArenaDefeatedCacheSnapshot
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return normalizeGloryArenaDefeatedCache(&result)
}

func normalizeGloryArenaDefeatedCache(value *GloryArenaDefeatedCacheSnapshot) *GloryArenaDefeatedCacheSnapshot {
	if value == nil || value.OpponentId <= 0 {
		return nil
	}
	return &GloryArenaDefeatedCacheSnapshot{
		OpponentId: value.OpponentId,
		SlotId:     normalizeGloryArenaSnapshotSlotID(value.SlotId),
		Heroes:     normalizeHeroSnapshotList(value.Heroes),
	}
}

func extractMainFormationHeroSnapshot(battleInfo *logicCommon.PlayerBattleInfo) []*logicCommon.HeroBasicInfo {
	if battleInfo == nil {
		return make([]*logicCommon.HeroBasicInfo, 0)
	}
	mainFormation := battleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]
	if mainFormation == nil || len(mainFormation.Heroes) == 0 {
		return make([]*logicCommon.HeroBasicInfo, 0)
	}

	result := make([]*logicCommon.HeroBasicInfo, 0, len(mainFormation.Heroes))
	for _, heroOwnID := range mainFormation.Heroes {
		heroInfo := battleInfo.FormationHeroes[heroOwnID]
		if heroInfo == nil {
			continue
		}
		result = append(result, cloneHeroBasicInfo(heroInfo))
	}
	return result
}

func buildSelectableHeroesForSnapshot(snapshot *GloryArenaDefeatedCacheSnapshot) []*pb.HonorArenaHeroInfo {
	if snapshot == nil || snapshot.OpponentId <= 0 {
		return make([]*pb.HonorArenaHeroInfo, 0)
	}
	heroes := snapshot.Heroes
	normalized := normalizeHeroSnapshotList(heroes)
	result := make([]*pb.HonorArenaHeroInfo, 0, len(normalized))
	for _, hero := range normalized {
		if hero == nil || hero.Id <= 0 {
			continue
		}
		if hero.Uid <= 0 {
			continue
		}
		fight := hero.Attr[enum.AttributeBasicCombatPower]
		result = append(result, &pb.HonorArenaHeroInfo{
			HeroId: hero.Uid,
			Cid:    int32(hero.Id),
			Level:  hero.Level,
			Star:   hero.Star,
			Job:    hero.ClassId,
			Fight:  fight,
		})
	}
	return result
}

func findSelectableHeroFromSnapshot(snapshot *GloryArenaDefeatedCacheSnapshot, heroSelectID int64) *logicCommon.HeroBasicInfo {
	if snapshot == nil || snapshot.OpponentId <= 0 || heroSelectID <= 0 {
		return nil
	}
	normalized := normalizeHeroSnapshotList(snapshot.Heroes)
	for _, hero := range normalized {
		if hero == nil || hero.Id <= 0 {
			continue
		}
		if hero.Uid == heroSelectID {
			return cloneHeroBasicInfo(hero)
		}
	}
	return nil
}

func normalizeHeroSnapshotList(heroes []*logicCommon.HeroBasicInfo) []*logicCommon.HeroBasicInfo {
	result := make([]*logicCommon.HeroBasicInfo, 0, len(heroes))
	for _, hero := range heroes {
		if hero == nil || hero.Id <= 0 {
			continue
		}
		result = append(result, cloneHeroBasicInfo(hero))
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.Id != right.Id {
			return left.Id < right.Id
		}
		if left.Level != right.Level {
			return left.Level > right.Level
		}
		if left.Star != right.Star {
			return left.Star > right.Star
		}
		if left.ClassId != right.ClassId {
			return left.ClassId < right.ClassId
		}
		return left.Units < right.Units
	})
	return result
}

func cloneHeroBasicInfo(src *logicCommon.HeroBasicInfo) *logicCommon.HeroBasicInfo {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Attr != nil {
		dst.Attr = make(map[int32]int64, len(src.Attr))
		for attrID, value := range src.Attr {
			dst.Attr[attrID] = value
		}
	}
	if src.Skill != nil {
		dst.Skill = append([]int32(nil), src.Skill...)
	}
	return &dst
}

func marshalPlayerRedisInfoList(value []*logicCommon.PlayerRedisInfo) string {
	data, err := json.Marshal(compactPlayerRedisInfoList(value))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func unmarshalPlayerRedisInfoList(raw string) []*logicCommon.PlayerRedisInfo {
	if raw == "" {
		return make([]*logicCommon.PlayerRedisInfo, 0)
	}

	var infos []*logicCommon.PlayerRedisInfo
	if err := json.Unmarshal([]byte(raw), &infos); err == nil {
		return compactPlayerRedisInfoList(infos)
	}

	// 兼容旧格式：[]int64
	ids := unmarshalInt64List(raw)
	return buildGloryArenaCachedPlayers(ids, nil)
}

func compactPlayerRedisInfoList(infos []*logicCommon.PlayerRedisInfo) []*logicCommon.PlayerRedisInfo {
	if len(infos) == 0 {
		return make([]*logicCommon.PlayerRedisInfo, 0)
	}

	uniq := make(map[int64]*logicCommon.PlayerRedisInfo, len(infos))
	for _, info := range infos {
		trimmed := trimPlayerRedisInfoForGloryArena(info)
		if trimmed == nil || trimmed.BasicInfo == nil || trimmed.BasicInfo.Id <= 0 {
			continue
		}
		uniq[trimmed.BasicInfo.Id] = trimmed
	}

	ids := make([]int64, 0, len(uniq))
	for id := range uniq {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	result := make([]*logicCommon.PlayerRedisInfo, 0, len(ids))
	for _, id := range ids {
		result = append(result, uniq[id])
	}
	return result
}

func buildGloryArenaCachedPlayers(opponentIds []int64, infos map[int64]*logicCommon.PlayerRedisInfo) []*logicCommon.PlayerRedisInfo {
	ids := compactInt64List(opponentIds)
	result := make([]*logicCommon.PlayerRedisInfo, 0, len(ids))
	for _, id := range ids {
		info := trimPlayerRedisInfoForGloryArena(infos[id])
		if info == nil {
			info = &logicCommon.PlayerRedisInfo{
				BasicInfo: &logicCommon.PlayerBasicInfo{Id: id},
			}
		}
		if info.BasicInfo == nil {
			info.BasicInfo = &logicCommon.PlayerBasicInfo{Id: id}
		}
		if info.BasicInfo.Id <= 0 {
			info.BasicInfo.Id = id
		}
		result = append(result, info)
	}
	return result
}

func trimPlayerRedisInfoForGloryArena(src *logicCommon.PlayerRedisInfo) *logicCommon.PlayerRedisInfo {
	if src == nil {
		return nil
	}

	dst := &logicCommon.PlayerRedisInfo{}
	if src.BasicInfo != nil {
		basicCopy := *src.BasicInfo
		dst.BasicInfo = &basicCopy
	}
	if src.BattleInfo == nil {
		return dst
	}

	dstBattle := &logicCommon.PlayerBattleInfo{
		UserId:          src.BattleInfo.UserId,
		FormationInfo:   make(map[int32]*logicCommon.FormationBasicInfo),
		FormationHeroes: make(map[int64]*logicCommon.HeroBasicInfo),
	}
	mainFormationType := int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)
	mainFormation := src.BattleInfo.FormationInfo[mainFormationType]
	if mainFormation != nil {
		heroIds := make([]int64, 0, len(mainFormation.Heroes))
		heroIds = append(heroIds, mainFormation.Heroes...)
		mainFormationCopy := &logicCommon.FormationBasicInfo{
			Heroes:      heroIds,
			BattlePower: mainFormation.BattlePower,
		}
		dstBattle.FormationInfo[mainFormationType] = mainFormationCopy
		for _, heroID := range heroIds {
			hero := src.BattleInfo.FormationHeroes[heroID]
			if hero == nil {
				continue
			}
			heroCopy := *hero
			if hero.Attr != nil {
				heroCopy.Attr = make(map[int32]int64, len(hero.Attr))
				for attrID, value := range hero.Attr {
					heroCopy.Attr[attrID] = value
				}
			}
			if hero.Skill != nil {
				heroCopy.Skill = append([]int32(nil), hero.Skill...)
			}
			dstBattle.FormationHeroes[heroID] = &heroCopy
		}
	}
	dst.BattleInfo = dstBattle
	return dst
}

func compactInt64List(ids []int64) []int64 {
	result := make([]int64, 0, len(ids))
	uniq := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || uniq[id] {
			continue
		}
		uniq[id] = true
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func IsGloryArenaSnapshotWin(winCount int32) bool {
	return winCount == 3 || winCount == 6 || winCount == 9
}

func getGloryArenaSnapshotSlotByWinCount(winCount int32) int32 {
	switch winCount {
	case 3:
		return 1
	case 6:
		return 2
	case 9:
		return 3
	default:
		return 0
	}
}

func isGloryArenaSnapshotSlotIDValid(slotID int32) bool {
	return slotID >= 1 && slotID <= 3
}

func normalizeGloryArenaSnapshotSlotID(slotID int32) int32 {
	if !isGloryArenaSnapshotSlotIDValid(slotID) {
		return 0
	}
	return slotID
}

func normalizeGloryArenaSeasonType(seasonType int32) int32 {
	if enum.IsValidGloryArenaSeasonType(seasonType) {
		return seasonType
	}
	return int32(enum.GLORY_ARENA_SEASON_TYPE_PRE)
}

func getGloryArenaInitLife() int32 {
	life := gameConfig.GetGloryArenaMaxHP()
	if life <= 0 {
		return GloryArenaDefaultLife
	}
	return life
}
