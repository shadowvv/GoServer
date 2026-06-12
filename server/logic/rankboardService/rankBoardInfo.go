package rankboardService

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type RankBoardInfo struct {
	mu           sync.RWMutex
	rankId       string
	rankInfo     []*model.RankBoardInfoEntity
	index        map[int64]*model.RankBoardInfoEntity
	dirty        bool
	lastActiveAt int64
	close        atomic.Bool
}

func NewRankBoard(rankId string, infos []*model.RankBoardInfoEntity) *RankBoardInfo {
	temp := &RankBoardInfo{
		rankId:       rankId,
		rankInfo:     make([]*model.RankBoardInfoEntity, 0),
		index:        make(map[int64]*model.RankBoardInfoEntity),
		lastActiveAt: tool.UnixNowMilli(),
	}
	needPersistBackfill := false
	for _, e := range infos {
		if e == nil {
			continue
		}
		// Backward compatibility for rows created before update_time was introduced.
		if e.UpdateTime <= 0 {
			e.UpdateTime = e.EnterTime
			needPersistBackfill = true
		}
		temp.rankInfo = append(temp.rankInfo, e)
		temp.index[e.Id] = e
	}
	temp.dirty = needPersistBackfill
	temp.close.Store(false)
	return temp
}

func (r *RankBoardInfo) UpdateScore(userId int64, score int64, incrementalUpdate bool, maxNum int32, resort bool) (isEnter bool, newRank int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := tool.UnixNowMilli()

	if e, ok := r.index[userId]; ok {
		if incrementalUpdate {
			e.Score += score
		} else {
			if e.Score == score {
				return false, e.Rank
			}
			e.Score = score
		}
		e.UpdateTime = now
		if resort {
			r.resort()
		}
		r.lastActiveAt = now
		r.dirty = true
		return false, e.Rank
	}

	newEntity := &model.RankBoardInfoEntity{
		Id:         userId,
		Score:      score,
		EnterTime:  now,
		UpdateTime: now,
		Rank:       int32(len(r.rankInfo) + 1),
	}

	if int32(len(r.rankInfo)) < maxNum {
		r.rankInfo = append(r.rankInfo, newEntity)
		r.index[userId] = newEntity
		if resort {
			r.resort()
		}
		r.lastActiveAt = now
		r.dirty = true
		return true, newEntity.Rank
	}

	if !resort {
		return false, newEntity.Rank
	}

	last := r.rankInfo[len(r.rankInfo)-1]
	if !betterThan(newEntity, last) {
		return false, newEntity.Rank
	}

	delete(r.index, last.Id)
	r.rankInfo[len(r.rankInfo)-1] = newEntity
	r.index[userId] = newEntity

	r.resort()
	r.lastActiveAt = now
	r.dirty = true
	return false, newEntity.Rank
}

func (r *RankBoardInfo) resort() {
	sort.Slice(r.rankInfo, func(i, j int) bool {
		a := r.rankInfo[i]
		b := r.rankInfo[j]

		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.UpdateTime < b.UpdateTime
	})

	for i, e := range r.rankInfo {
		e.Rank = int32(i + 1)
	}
}

func betterThan(a, b *model.RankBoardInfoEntity) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.EnterTime < b.EnterTime
}

func (r *RankBoardInfo) ThumbUp(userId int64, thumbUp int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.index[userId]; ok {
		e.ThumbUpCount += thumbUp
		e.UpdateTime = tool.UnixNowMilli()
		r.dirty = true
		r.lastActiveAt = e.UpdateTime
	}
}

func (r *RankBoardInfo) GetTopN(n int) []*model.RankBoardInfoEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n < 0 {
		n = 0
	}
	if n > len(r.rankInfo) {
		n = len(r.rankInfo)
	}
	return r.rankInfo[:n]
}

func (r *RankBoardInfo) GetUserRank(userId int64) *model.RankBoardInfoEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.index[userId]
}

func (r *RankBoardInfo) StartPersistLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if r.close.Load() {
				return
			}
			if ShouldSkipRankUpdate() {
				continue
			}

			r.mu.Lock()
			if !r.dirty {
				r.mu.Unlock()
				continue
			}
			snapshot := make([]*model.RankBoardInfoEntity, 0, len(r.rankInfo))
			for _, e := range r.rankInfo {
				clone := *e
				snapshot = append(snapshot, &clone)
			}
			r.dirty = false
			r.mu.Unlock()

			if err := easyDB.SaveRankBoardToDB(r.rankId, snapshot); err != nil {
				r.mu.Lock()
				r.dirty = true
				r.mu.Unlock()
			}
		}
	}()
}

// TODO:是否可以移除
func ShouldSkipRankUpdate() bool {
	now := time.Now()

	y, m, d := now.Date()
	loc := now.Location()

	start := time.Date(y, m, d, 0, 0, 0, 0, loc)
	end1 := start.Add(5 * time.Minute)
	end2 := time.Date(y, m, d, 23, 55, 0, 0, loc)

	if now.Before(end1) || now.After(end2) {
		return true
	}
	return false
}

func (r *RankBoardInfo) stopPersistLoop() {
	r.close.Store(true)
}

func (r *RankBoardInfo) tryRecoverAndSettleRanks(currentTime int64) {
	todayDate := int64(tool.GetTodayDataIntByTimeStamp(currentTime))
	allowTodaySettle := isAfterDailySettleStart(currentTime)

	settleCfg, version, serverID, ok := r.buildSettleCfgByRankID()
	if !ok || settleCfg == nil || version == "" {
		logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:invalid_settle_cfg ok:%t cfgNil:%t version:%s currentDate:%d", r.rankId, ok, settleCfg == nil, version, todayDate)
		return
	}
	logger.InfoWithSprintf("[rankSettle] check rankId:%s pointType:%d version:%s serverID:%d settleTypes:%v rewardIds:%v mailIds:%v currentDate:%d allowToday:%t", r.rankId, settleCfg.PointType, version, serverID, settleCfg.SettleTypes, settleCfg.RankRewardIDs, settleCfg.MailIDs, todayDate, allowTodaySettle)
	if settleCfg.SendRewardType == int32(enum.RANK_BOARD_SEND_REWARD_TYPE_ENTER) {
		logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:enter_reward_type sendRewardType:%d", r.rankId, settleCfg.SendRewardType)
		return
	}
	if len(settleCfg.SettleTypes) == 0 || len(settleCfg.RankRewardIDs) == 0 || len(settleCfg.MailIDs) == 0 {
		logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:empty_settle_config settleTypes:%v rewardIds:%v mailIds:%v", r.rankId, settleCfg.SettleTypes, settleCfg.RankRewardIDs, settleCfg.MailIDs)
		return
	}

	for _, settleType := range settleCfg.SettleTypes {
		settleDates := logicCommon.GetRankSettleTaskSettleDates(settleCfg.PointType, settleType, settleCfg.SettleTypes, version, currentTime)
		if len(settleDates) == 0 {
			logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:empty_settle_dates pointType:%d settleType:%d allSettleTypes:%v version:%s serverID:%d currentDate:%d", r.rankId, settleCfg.PointType, settleType, settleCfg.SettleTypes, version, serverID, todayDate)
			continue
		}
		logger.InfoWithSprintf("[rankSettle] rankId:%s settleType:%d settleDates:%v", r.rankId, settleType, settleDates)
		for _, settleDate := range settleDates {
			if settleDate <= 0 {
				logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:invalid_settle_date settleType:%d settleDate:%d", r.rankId, settleType, settleDate)
				continue
			}
			if settleDate == todayDate && !allowTodaySettle {
				logger.InfoWithSprintf("[rankSettle] skip rankId:%s reason:before_daily_settle_window settleType:%d settleDate:%d currentDate:%d", r.rankId, settleType, settleDate, todayDate)
				continue
			}
			taskVersion := fmt.Sprintf("%08d", settleDate)
			logger.InfoWithSprintf("[rankSettle] process rankId:%s settleType:%d taskVersion:%s settleDate:%d", r.rankId, settleType, taskVersion, settleDate)
			if err := rankBoardService.ensureAndProcessSettleTask(r.rankId, int8(settleType), taskVersion, settleDate, serverID, settleCfg, currentTime); err != nil {
				logger.ErrorBySprintf("[rankBoardInfo] settle task process failed rankId:%s settleType:%d taskVersion:%s err:%v", r.rankId, settleType, taskVersion, err)
				continue
			}
			logger.InfoWithSprintf("[rankSettle] process done rankId:%s settleType:%d taskVersion:%s", r.rankId, settleType, taskVersion)
		}
	}
}

func (r *RankBoardInfo) buildSettleCfgByRankID() (*settleRuleCfg, string, int32, bool) {
	if r == nil || r.rankId == "" {
		return nil, "", 0, false
	}

	commonID, actID, actRankID, _ := logicCommon.GetRankRealIdFromUniqueId(r.rankId)
	if commonID != 0 {
		cfg := gameConfig.GetRankCfgByIds(0, commonID)
		if cfg == nil {
			return nil, "", 0, false
		}
		_, version, serverID, ok := logicCommon.ParseCommonArenaRankTableMeta(r.rankId)
		if !ok || version == "" {
			return nil, "", 0, false
		}
		return &settleRuleCfg{
			PointType:      cfg.PointType,
			SendRewardType: cfg.SendRewardType,
			SettleTypes:    cfg.SettlementType,
			RankRewardIDs:  cfg.RankRewardsId,
			MailIDs:        cfg.MailId,
		}, version, serverID, true
	}

	if actID != 0 && actRankID != 0 {
		cfg := gameConfig.GetRankActCfg(actRankID)
		if cfg == nil {
			return nil, "", 0, false
		}
		_, _, version, ok := parseActivityRankTableMeta(r.rankId)
		if !ok || version == "" {
			return nil, "", 0, false
		}
		return &settleRuleCfg{
			PointType:      cfg.PointType,
			SendRewardType: cfg.SendRewardType,
			SettleTypes:    cfg.SettlementType,
			RankRewardIDs:  cfg.RankRewardsId,
			MailIDs:        cfg.MailId,
		}, version, 0, true
	}

	return nil, "", 0, false
}
