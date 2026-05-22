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
	for _, e := range infos {
		temp.rankInfo = append(temp.rankInfo, e)
		temp.index[e.Id] = e
	}
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
		if resort {
			r.resort()
		}
		r.lastActiveAt = now
		r.dirty = true
		return false, e.Rank
	}

	newEntity := &model.RankBoardInfoEntity{
		Id:        userId,
		Score:     score,
		EnterTime: now,
		Rank:      int32(len(r.rankInfo) + 1),
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
		return a.EnterTime < b.EnterTime
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
		r.dirty = true
		r.lastActiveAt = tool.UnixNowMilli()
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
		return
	}
	if settleCfg.SendRewardType == int32(enum.RANK_BOARD_SEND_REWARD_TYPE_ENTER) {
		return
	}
	if len(settleCfg.SettleTypes) == 0 || len(settleCfg.RankRewardIDs) == 0 || len(settleCfg.MailIDs) == 0 {
		return
	}

	for _, settleType := range settleCfg.SettleTypes {
		settleDates := logicCommon.GetRankSettleTaskSettleDates(settleCfg.PointType, settleType, settleCfg.SettleTypes, version, currentTime)
		for _, settleDate := range settleDates {
			if settleDate <= 0 {
				continue
			}
			if settleDate == todayDate && !allowTodaySettle {
				continue
			}
			taskVersion := fmt.Sprintf("%08d", settleDate)
			if err := rankBoardService.ensureAndProcessSettleTask(r.rankId, int8(settleType), taskVersion, settleDate, serverID, settleCfg, currentTime); err != nil {
				logger.ErrorBySprintf("[rankBoardInfo] settle task process failed rankId:%s settleType:%d taskVersion:%s err:%v", r.rankId, settleType, taskVersion, err)
			}
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
