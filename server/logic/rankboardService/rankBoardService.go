package rankboardService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/platform/rankBoardPlatform"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

var rankBoardService *RankBoardService

const UPDATE_INTERVAL = time.Minute * 1
const HEARTBEAT_INTERVAL = time.Second * 30
const IDLE_TIMEOUT = time.Hour * 24 * 7

func InitService() {
	rankBoardService = newRankBoardService()
	rankBoardService.heartBeat(HEARTBEAT_INTERVAL, IDLE_TIMEOUT)
}

func newRankBoardService() *RankBoardService {
	var serverData *model.RankBoardServerDataEntity
	infos, err := easyDB.GetRankBoardData[model.RankBoardServerDataEntity]("rank_board_server_data")
	if err != nil || len(infos) == 0 {
		data := &model.RankBoardServerDataEntity{
			NodeId:       nodeConfig.NodeId,
			LastTimeTime: tool.UnixNowMilli(),
		}
		err := easyDB.SaveRankBoardData(data)
		if err != nil {
			panic(fmt.Sprintf("save rankboard server data error: %s", err))
		}
		serverData = data
	} else {
		serverData = infos[0]
	}
	logger.InfoWithSprintf("init rankBoard service data:%v", serverData)

	service := &RankBoardService{
		rankBoardInfoMap: make(map[string]*RankBoardInfo),
		serverData:       serverData,
	}

	commonRankMap := gameConfig.GetAllRankCfg()
	now := tool.UnixNowMilli()
	for rankID, cfg := range commonRankMap[0] {
		if cfg == nil {
			continue
		}
		// TODO:活动排行榜也需要想办法判断是否加载
		tableNames, err := easyDB.GetRankTableNames(fmt.Sprintf("common_%d", rankID))
		if err != nil {
			logger.ErrorBySprintf("[rankBoardService] scan rank tables failed prefix:common_%d err:%v", rankID, err)
			continue
		}
		for _, tableName := range tableNames {
			if !shouldLoadCommonTable(tableName, cfg, now) {
				continue
			}
			info, getErr := easyDB.GetRankBoardData[model.RankBoardInfoEntity](tableName)
			if getErr != nil {
				logger.ErrorBySprintf("[rankBoardService] load rank table data failed table:%s err:%v", tableName, getErr)
				continue
			}
			temp := NewRankBoard(tableName, info)
			service.rankBoardInfoMap[tableName] = temp
			temp.StartPersistLoop(UPDATE_INTERVAL)
		}
	}
	return service
}

func shouldLoadCommonTable(rankTable string, cfg *gameConfig.CommonRankCfg, currentTime int64) bool {
	if cfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) || cfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) {
		return shouldLoadCommonArenaRankTable(rankTable, cfg, currentTime)
	}
	if cfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT) || cfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT) || cfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT) {
		return shouldLoadCommonGloryArenaRankTable(rankTable, cfg, currentTime)
	}
	return true
}

func shouldLoadCommonArenaRankTable(rankTable string, cfg *gameConfig.CommonRankCfg, currentTime int64) bool {
	_, _, _, version := logicCommon.GetRankRealIdFromUniqueId(rankTable)
	_, startMilli, weekEndMilli, ok := logicCommon.ParseArenaRankVersionDateInt(version)
	if !ok {
		return true
	}
	if currentTime <= weekEndMilli {
		return true
	}
	cappedCheckTime := currentTime
	if weekEndMilli < cappedCheckTime {
		cappedCheckTime = weekEndMilli
	}

	for _, settleType := range cfg.SettlementType {
		settleDates := logicCommon.GetArenaRankSettleTaskSettleDates(settleType, cfg.SettlementType, startMilli, weekEndMilli, cappedCheckTime)
		for _, settleDate := range settleDates {
			taskVersion := fmt.Sprintf("%08d", settleDate)
			if isSettleTaskRewardDone(rankTable, int8(settleType), taskVersion) {
				return true
			}
		}
	}
	return false
}

func shouldLoadCommonGloryArenaRankTable(rankTable string, cfg *gameConfig.CommonRankCfg, currentTime int64) bool {
	_, _, _, version := logicCommon.GetRankRealIdFromUniqueId(rankTable)
	startMilli, weekEndMilli, ok := logicCommon.ParseGloryArenaRankVersionDateInt(version)
	if !ok {
		return true
	}
	if currentTime <= weekEndMilli {
		return true
	}
	cappedCheckTime := currentTime
	if weekEndMilli < cappedCheckTime {
		cappedCheckTime = weekEndMilli
	}

	for _, settleType := range cfg.SettlementType {
		settleDates := logicCommon.GetArenaRankSettleTaskSettleDates(settleType, cfg.SettlementType, startMilli, weekEndMilli, cappedCheckTime)
		for _, settleDate := range settleDates {
			taskVersion := fmt.Sprintf("%08d", settleDate)
			if isSettleTaskRewardDone(rankTable, int8(settleType), taskVersion) {
				return true
			}
		}
	}
	return false
}

func isSettleTaskRewardDone(rankTable string, settleType int8, taskVersion string) bool {
	rows, err := easyDB.GetRankDataByRaw[countResult](
		"SELECT COUNT(1) AS cnt FROM rank_settle_task WHERE rank_id=? AND settle_type=? AND version=? AND status=?",
		rankTable, settleType, taskVersion, enum.RankSettleTaskStatusRewardDone,
	)
	if err != nil || len(rows) == 0 || rows[0] == nil {
		return false
	}
	return rows[0].Cnt > 0
}

type RankBoardService struct {
	mu               sync.Mutex
	rankBoardInfoMap map[string]*RankBoardInfo
	serverData       *model.RankBoardServerDataEntity
}

func (s *RankBoardService) GetRankInfo(rankId string, num int, playerId int64) ([]*model.RankBoardInfoEntity, int32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rankInfo := s.getRankInfoByRankId(rankId)
	if rankInfo == nil {
		return nil, 0, errors.New("rank info is nil")
	}
	playerRank := int32(0)
	playerInfo := rankInfo.GetUserRank(playerId)
	if playerInfo != nil {
		playerRank = playerInfo.Rank
	}
	return rankInfo.GetTopN(num), playerRank, nil
}

func (s *RankBoardService) UpdatePlayerRankInfo(rankId string, userId int64, score int64, incrementalUpdate bool, maxNum int32, resort bool) (isEnter bool, newRank int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rankInfo := s.getRankInfoByRankId(rankId)
	if rankInfo == nil {
		return false, 0
	}
	return rankInfo.UpdateScore(userId, score, incrementalUpdate, maxNum, resort)
}

func (s *RankBoardService) UpdateRankInfoThumbUp(rankId string, userId int64, thumbUp int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rankInfo := s.getRankInfoByRankId(rankId)
	if rankInfo != nil {
		rankInfo.ThumbUp(userId, thumbUp)
	}
}

func (s *RankBoardService) GetPlayerRank(rankId string, userId int64) *model.RankBoardInfoEntity {
	s.mu.Lock()
	defer s.mu.Unlock()

	rankInfo := s.getRankInfoByRankId(rankId)
	if rankInfo == nil {
		return nil
	}
	return rankInfo.GetUserRank(userId)
}

func (s *RankBoardService) getRankInfoByRankId(rankId string) *RankBoardInfo {
	if rankInfo, ok := s.rankBoardInfoMap[rankId]; ok {
		return rankInfo
	} else {
		data, err := easyDB.GetRankBoardData[model.RankBoardInfoEntity](rankId)
		if err == nil {
			temp := NewRankBoard(rankId, data)
			s.rankBoardInfoMap[rankId] = temp
			temp.StartPersistLoop(UPDATE_INTERVAL)
			return temp
		} else {
			err = easyDB.CreateRankTable(rankId)
			if err != nil {
				logger.ErrorBySprintf("create rankboard table error: %s", rankId)
				return nil
			}
			temp := NewRankBoard(rankId, data)
			s.rankBoardInfoMap[rankId] = temp
			temp.dirty = true
			temp.StartPersistLoop(UPDATE_INTERVAL)
			return temp
		}
	}
}

func (s *RankBoardService) heartBeat(checkInterval time.Duration, idleTTL time.Duration) {
	s.serverData.LastTimeTime = tool.UnixNowMilli()
	err := easyDB.SaveRankBoardData(s.serverData)
	if err != nil {
		logger.ErrorBySprintf("save rankboard server data error: %s", err)
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		lastSettleCheckAt := int64(0)

		for range ticker.C {
			now := tool.UnixNowMilli()
			if now-lastSettleCheckAt >= tool.MINUTE_MILLI {
				s.tryRecoverAndSettleRanks(now)
				lastSettleCheckAt = now
			}

			s.mu.Lock()
			for rankId, board := range s.rankBoardInfoMap {
				board.mu.RLock()
				idle := now - board.lastActiveAt
				isDirty := board.dirty
				board.mu.RUnlock()

				// ⭐ 超过 1 天无变化 && 已经落地
				if idle >= idleTTL.Milliseconds() && !isDirty {
					board.stopPersistLoop()
					delete(s.rankBoardInfoMap, rankId)
				}

			}
			s.mu.Unlock()

			s.serverData.LastTimeTime = now
			err := easyDB.SaveRankBoardData(s.serverData)
			if err != nil {
				logger.ErrorBySprintf("save rankboard server data error: %s", err)
			}
		}
	}()
}

type settleRuleCfg struct {
	PointType      int32
	SendRewardType int32
	SettleTypes    []int32
	RankRewardIDs  []int32
	MailIDs        []int32
}

type countResult struct {
	Cnt int64 `gorm:"column:cnt"`
}

func (s *RankBoardService) tryRecoverAndSettleRanks(currentTime int64) {
	s.mu.Lock()
	boards := make([]*RankBoardInfo, 0, len(s.rankBoardInfoMap))
	for _, rankInfo := range s.rankBoardInfoMap {
		if rankInfo != nil {
			boards = append(boards, rankInfo)
		}
	}
	s.mu.Unlock()
	for _, rankInfo := range boards {
		rankInfo.tryRecoverAndSettleRanks(currentTime)
	}
}

// 每天的00:00到00:15是留给其他系统结算的。
func isAfterDailySettleStart(currentTime int64) bool {
	t := time.UnixMilli(currentTime)
	if t.Hour() > 0 {
		return true
	}
	return t.Minute() >= 15
}

func (s *RankBoardService) scanAndProcessRankTables(prefix string, cfg *settleRuleCfg, currentTime int64, todayDate int64, allowTodaySettle bool) {
	if cfg == nil {
		return
	}
	tableNames, err := easyDB.GetRankTableNames(prefix)
	if err != nil {
		logger.ErrorBySprintf("[rankBoardService] scan rank tables failed prefix:%s err:%v", prefix, err)
		return
	}
	for _, tableName := range tableNames {
		_, version, serverID, ok := logicCommon.ParseCommonArenaRankTableMeta(tableName)
		if !ok {
			continue
		}
		s.processRankTableSettleTasks(tableName, version, serverID, cfg, currentTime, todayDate, allowTodaySettle)
	}
}

func (s *RankBoardService) processRankTableSettleTasks(rankTable string, tableVersion string, serverID int32, cfg *settleRuleCfg, currentTime int64, todayDate int64, allowTodaySettle bool) {
	if cfg == nil {
		return
	}
	if cfg.SendRewardType == int32(enum.RANK_BOARD_SEND_REWARD_TYPE_ENTER) {
		return
	}
	if len(cfg.SettleTypes) == 0 || len(cfg.RankRewardIDs) == 0 || len(cfg.MailIDs) == 0 {
		return
	}
	for _, settleType := range cfg.SettleTypes {
		settleDates := logicCommon.GetRankSettleTaskSettleDates(cfg.PointType, settleType, cfg.SettleTypes, tableVersion, currentTime)
		for _, settleDate := range settleDates {
			if settleDate <= 0 {
				continue
			}
			if settleDate == todayDate && !allowTodaySettle {
				continue
			}
			taskVersion := fmt.Sprintf("%08d", settleDate)
			if err := s.ensureAndProcessSettleTask(rankTable, int8(settleType), taskVersion, settleDate, serverID, cfg, currentTime); err != nil {
				logger.ErrorBySprintf("[rankBoardService] settle task process failed table:%s settleType:%d taskVersion:%s err:%v", rankTable, settleType, taskVersion, err)
			}
		}
	}
}

func (s *RankBoardService) ensureAndProcessSettleTask(rankTable string, settleType int8, taskVersion string, settleDate int64, serverID int32, cfg *settleRuleCfg, currentTime int64) error {
	now := currentTime
	if now <= 0 {
		now = tool.UnixNowMilli()
	}
	if err := easyDB.RunRankRawSql(
		"INSERT IGNORE INTO rank_settle_task (rank_id, settle_type, version, settle_time, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		rankTable, settleType, taskVersion, settleDate, enum.RankSettleTaskStatusPending, now, now,
	); err != nil {
		return err
	}

	tasks, err := easyDB.GetRankDataByRaw[model.RankSettleTaskEntity](
		"SELECT * FROM rank_settle_task WHERE rank_id=? AND settle_type=? AND version=? LIMIT 1",
		rankTable, settleType, taskVersion,
	)
	if err != nil || len(tasks) == 0 {
		return err
	}
	task := tasks[0]
	if task == nil {
		logger.InfoWithSprintf("[rankSettle] skip rankId:%s settleType:%d taskVersion:%s reason:task_nil", rankTable, settleType, taskVersion)
		return nil
	}
	if task.Status == enum.RankSettleTaskStatusRewardDone {
		logger.InfoWithSprintf("[rankSettle] skip rankId:%s settleType:%d taskVersion:%s reason:task_reward_done taskId:%d", rankTable, settleType, taskVersion, task.Id)
		return nil
	}

	if task.Status != enum.RankSettleTaskStatusSnapshotDone {
		logger.InfoWithSprintf("[rankSettle] rebuild snapshot rankId:%s settleType:%d taskVersion:%s taskId:%d oldStatus:%d", rankTable, settleType, taskVersion, task.Id, task.Status)
		if err = s.rebuildSettleTaskSnapshot(task, now); err != nil {
			_ = easyDB.RunRankRawSql("UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?", enum.RankSettleTaskStatusFailed, now, task.Id)
			return err
		}
	}
	logger.InfoWithSprintf("[rankSettle] send rewards rankId:%s settleType:%d taskVersion:%s taskId:%d", rankTable, settleType, taskVersion, task.Id)
	if err = s.processSettleTaskRewards(task, cfg, serverID, now); err != nil {
		_ = easyDB.RunRankRawSql("UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?", enum.RankSettleTaskStatusFailed, now, task.Id)
		return err
	}
	return nil
}

func (s *RankBoardService) rebuildSettleTaskSnapshot(task *model.RankSettleTaskEntity, now int64) error {
	if task == nil {
		return nil
	}
	if err := easyDB.RunRankRawSql("UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?", enum.RankSettleTaskStatusRunning, now, task.Id); err != nil {
		return err
	}
	if err := easyDB.RunRankRawSql("DELETE FROM rank_snapshot_info WHERE task_id=?", task.Id); err != nil {
		return err
	}

	if rows, ok := s.buildSnapshotRowsFromMemory(task, now); ok {
		if err := easyDB.CreateRankRows("rank_snapshot_info", rows); err != nil {
			return err
		}
	} else {
		logger.ErrorBySprintf("rank board not in memory %v", task)
		return errors.New("rank snapshot not in memory")
	}
	return easyDB.RunRankRawSql("UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?", enum.RankSettleTaskStatusSnapshotDone, now, task.Id)
}

func (s *RankBoardService) buildSnapshotRowsFromMemory(task *model.RankSettleTaskEntity, now int64) ([]*model.RankSnapshotInfoEntity, bool) {
	if task == nil {
		return nil, false
	}

	s.mu.Lock()
	board, ok := s.rankBoardInfoMap[task.RankId]
	s.mu.Unlock()
	if !ok || board == nil {
		return nil, false
	}

	board.mu.RLock()
	if len(board.rankInfo) == 0 {
		board.mu.RUnlock()
		return []*model.RankSnapshotInfoEntity{}, true
	}
	rows := make([]*model.RankSnapshotInfoEntity, 0, len(board.rankInfo))
	for _, item := range board.rankInfo {
		if item == nil {
			continue
		}
		rows = append(rows, &model.RankSnapshotInfoEntity{
			TaskId:       task.Id,
			RankId:       task.RankId,
			SettleType:   task.SettleType,
			Version:      task.Version,
			SourceId:     item.Id,
			Rank:         item.Rank,
			Score:        item.Score,
			ThumbUpCount: int64(item.ThumbUpCount),
			EnterTime:    item.EnterTime,
			RewardStatus: enum.RankRewardStatusPending,
			CreatedAt:    now,
		})
	}
	board.mu.RUnlock()
	return rows, true
}

func (s *RankBoardService) processSettleTaskRewards(task *model.RankSettleTaskEntity, cfg *settleRuleCfg, serverID int32, now int64) error {
	if task == nil || cfg == nil {
		return nil
	}
	rewardIndex := getRewardIndexBySettleType(cfg.SettleTypes, int32(task.SettleType))
	if rewardIndex < 0 || rewardIndex >= len(cfg.RankRewardIDs) || rewardIndex >= len(cfg.MailIDs) {
		return easyDB.RunRankRawSql("UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?", enum.RankSettleTaskStatusRewardDone, now, task.Id)
	}

	rewardCfgID := cfg.RankRewardIDs[rewardIndex]
	mailID := cfg.MailIDs[rewardIndex]
	snapshots, err := easyDB.GetRankDataByRaw[model.RankSnapshotInfoEntity](
		"SELECT * FROM rank_snapshot_info WHERE task_id=? AND reward_status=? ORDER BY `rank` ASC",
		task.Id, enum.RankRewardStatusPending,
	)
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		if rewardCfgID == 0 {
			_ = easyDB.RunRankRawSql("UPDATE rank_snapshot_info SET reward_status=? WHERE id=?", enum.RankRewardStatusDone, snapshot.Id)
			continue
		}
		dropID := gameConfig.GetRankRewardCfgWithRank(rewardCfgID, snapshot.Rank)
		if dropID == 0 {
			_ = easyDB.RunRankRawSql("UPDATE rank_snapshot_info SET reward_status=? WHERE id=?", enum.RankRewardStatusDone, snapshot.Id)
			continue
		}
		items := gameConfig.Drop(dropID)
		if len(items) == 0 {
			_ = easyDB.RunRankRawSql("UPDATE rank_snapshot_info SET reward_status=? WHERE id=?", enum.RankRewardStatusDone, snapshot.Id)
			continue
		}

		rewardPayload, _ := json.Marshal(items)
		rows, insertErr := easyDB.RunRankRawSqlWithRowsAffected(
			"INSERT IGNORE INTO rank_reward_record (task_id, source_id, reward, `rank`, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			task.Id, snapshot.SourceId, string(rewardPayload), snapshot.Rank, enum.RankRewardStatusPending, now, now,
		)
		if insertErr != nil {
			return insertErr
		}
		if rows == 0 {
			records, queryErr := easyDB.GetRankDataByRaw[model.RankRewardRecordEntity](
				"SELECT * FROM rank_reward_record WHERE task_id=? AND source_id=? LIMIT 1",
				task.Id, snapshot.SourceId,
			)
			if queryErr == nil && len(records) > 0 && records[0] != nil && records[0].Status == enum.RankRewardStatusDone {
				_ = easyDB.RunRankRawSql("UPDATE rank_snapshot_info SET reward_status=? WHERE id=?", enum.RankRewardStatusDone, snapshot.Id)
				continue
			}
		}

		sendErr := s.sendSettleRewardByPointType(cfg.PointType, mailID, serverID, snapshot.SourceId, items, snapshot.Rank)
		if sendErr != nil {
			return sendErr
		}
		if err = easyDB.RunRankRawSql(
			"UPDATE rank_reward_record SET status=?, updated_at=? WHERE task_id=? AND source_id=?",
			enum.RankRewardStatusDone, now, task.Id, snapshot.SourceId,
		); err != nil {
			return err
		}
		if err = easyDB.RunRankRawSql(
			"UPDATE rank_snapshot_info SET reward_status=? WHERE id=?",
			enum.RankRewardStatusDone, snapshot.Id,
		); err != nil {
			return err
		}
	}

	pendingCount, err := easyDB.GetRankDataByRaw[countResult](
		"SELECT COUNT(1) AS cnt FROM rank_snapshot_info WHERE task_id=? AND reward_status=?",
		task.Id, enum.RankRewardStatusPending,
	)
	if err != nil {
		return err
	}
	if len(pendingCount) == 0 || pendingCount[0] == nil || pendingCount[0].Cnt == 0 {
		return easyDB.RunRankRawSql(
			"UPDATE rank_settle_task SET status=?, updated_at=? WHERE id=?",
			enum.RankSettleTaskStatusRewardDone, now, task.Id,
		)
	}
	return nil
}

func (s *RankBoardService) sendSettleRewardByPointType(pointType int32, mailID int32, serverID int32, sourceID int64, items []*gameConfig.ItemConfig, rank int32) error {
	switch pointType {
	case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA),
		int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT),
		int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_TOTAL_POWER):
		if serverID <= 0 {
			serverID = 0
		}
		return rankBoardPlatform.SendRankBoardAllianceRewardMail(mailID, serverID, sourceID, items, rank)
	default:
		rankBoardPlatform.SendRankBoardRewardMail(mailID, sourceID, items, rank)
		return nil
	}
}

func parseActivityRankTableMeta(tableName string) (int32, int32, string, bool) {
	parts := strings.Split(tableName, "_")
	if len(parts) < 4 || parts[0] != "activity" {
		return 0, 0, "", false
	}
	actID, err1 := strconv.Atoi(parts[1])
	actRankID, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return 0, 0, "", false
	}
	version := strings.Join(parts[3:], "_")
	return int32(actID), int32(actRankID), version, true
}

func getRewardIndexBySettleType(settleTypes []int32, settleType int32) int {
	for idx, v := range settleTypes {
		if v == settleType {
			return idx
		}
	}
	return -1
}

func GetRankInfo(rankId string, maxNum int, playerId int64) ([]*model.RankBoardInfoEntity, int32, error) {
	return rankBoardService.GetRankInfo(rankId, maxNum, playerId)
}

func UpdatePlayerRankInfo(rankId string, userId int64, score int64, incrementalUpdate bool, maxNum int32, resort bool) (isEnter bool, newRank int32) {
	return rankBoardService.UpdatePlayerRankInfo(rankId, userId, score, incrementalUpdate, maxNum, resort)
}

func UpdateRankInfoThumbUp(rankId string, userId int64, thumbUp int32) {
	rankBoardService.UpdateRankInfoThumbUp(rankId, userId, thumbUp)
}

func GetPlayerRank(rankId string, playerId int64) *model.RankBoardInfoEntity {
	return rankBoardService.GetPlayerRank(rankId, playerId)
}
