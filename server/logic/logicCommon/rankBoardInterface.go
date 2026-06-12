package logicCommon

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
)

type serverOpenTimeMeta struct {
	ServerID       int32 `gorm:"column:server_id"`
	ServerOpenTime int64 `gorm:"column:server_open_time"`
}

func (serverOpenTimeMeta) TableName() string {
	return "server_info"
}

// version不能是全数字，不然会被误读成serverId
func GetRankUniqueId(rankId, activityId, actRankId, ServerId int32, version string) (string, error) {
	if _, err := strconv.Atoi(version); err == nil {
		return "", errors.New("version is invalid,it must not be integer")
	}
	if activityId == 0 {
		rankCfg := gameConfig.GetRankCfg(rankId)
		if rankCfg == nil {
			return fmt.Sprintf("common_%d_%d", rankId, ServerId), nil
		}

		if version == "" {
			return fmt.Sprintf("common_%d_%d", rankId, ServerId), nil
		}
		return fmt.Sprintf("common_%d_%s", rankId, version), nil
	}

	return fmt.Sprintf("activity_%d_%d_%s", activityId, actRankId, version), nil
}

func GetRankRealIdFromUniqueId(uniqueId string) (commonId int32, activityRankId int32, actRankId int32, version string) {
	parts := strings.Split(uniqueId, "_")
	if len(parts) < 3 {
		return 0, 0, 0, ""
	}

	switch parts[0] {
	case "common":
		// common_{rankId}_{serverId}/{version}
		rid, err1 := strconv.Atoi(parts[1])
		if err1 != nil {
			return 0, 0, 0, ""
		}
		version = parts[2]
		return int32(rid), 0, 0, version

	case "activity":
		// activity_{activityId}_{actRankId}_{activityVersion}
		if len(parts) != 4 {
			return 0, 0, 0, ""
		}

		id, err1 := strconv.Atoi(parts[1])
		rankId, err2 := strconv.Atoi(parts[2])
		version = parts[3]

		if err1 != nil {
			return 0, 0, 0, ""
		}
		if err2 != nil {
			return 0, 0, 0, ""
		}
		return 0, int32(id), int32(rankId), version

	default:
		return 0, 0, 0, ""
	}
}

func UpdateAreanScoreRank(serverId int32, version string, userId int64, score int32) {
	redisKey := enum.GetArenaScoreInfoKey(serverId, version)
	ctx := context.Background()

	// 更新分数
	dbService.RDB.ZAdd(ctx, redisKey, &redis.Z{
		Score:  float64(score),
		Member: strconv.FormatInt(userId, 10),
	})

	maxRankingSize := int64(1000) // 1000
	// 只保留前N名
	dbService.RDB.ZRemRangeByRank(ctx, redisKey, maxRankingSize, -1)
	dbService.RDB.ExpireNX(ctx, redisKey, 7*24*time.Hour)
}

func GetCommonRankUniqueIdsByPointType(serverId int32, pointType enum.RankBoardScoreType, version string) []string {
	rankIds := make([]string, 0)
	for _, configMap := range gameConfig.GetAllRankCfg() {
		for _, cfg := range configMap {
			if cfg == nil || cfg.PointType != int32(pointType) {
				continue
			}
			rankId, err := GetRankUniqueId(cfg.Id, cfg.ActId, cfg.Id, serverId, version)
			if err != nil {
				continue
			}
			rankIds = append(rankIds, rankId)
		}

	}
	return rankIds
}

func BuildArenaRankVersion(serverId int32, mondayDate string) string {
	if mondayDate == "" {
		return ""
	}
	return fmt.Sprintf("s%d:t%s", serverId, mondayDate)
}

func GetArenaRankVersionByTime(serverId int32, currentTime int64) string {
	// Arena version switches at 00:30. During 00:00~00:29:59,
	// treat it as previous day to avoid early weekly version rollover.
	dayElapsed := currentTime - tool.GetTodayZeroByTimeStamp(currentTime)
	if dayElapsed < 30*tool.MINUTE_MILLI {
		currentTime -= 30 * tool.MINUTE_MILLI
	}
	return BuildArenaRankVersion(serverId, tool.GetMondayDataStringByTimeStamp(currentTime))
}

// ParseCommonArenaRankTableMeta parses common rank table names with compatible formats:
// common_{rankId}_{serverId}
// common_{rankId}_{version}
func ParseCommonArenaRankTableMeta(tableName string) (int32, string, int32, bool) {
	parts := strings.Split(tableName, "_")
	if len(parts) < 3 || parts[0] != "common" {
		return 0, "", 0, false
	}
	rankID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, "", 0, false
	}

	third := parts[2]
	if sid, sidErr := strconv.Atoi(third); sidErr == nil {
		if len(third) == 8 {
			return int32(rankID), third, 0, true
		}
		return int32(rankID), third, int32(sid), true
	}
	if parsedSID, _, ok := ParseArenaRankVersion(third); ok {
		return int32(rankID), third, parsedSID, true
	}
	return int32(rankID), third, 0, true
}

// GetRankSettleTaskSettleDates returns all settle_time(YYYYMMDD, date-only) that should be checked
// for the given settleType at currentTime.
func GetRankSettleTaskSettleDates(pointType int32, settleType int32, allSettleTypes []int32, version string, currentTime int64) []int64 {
	if currentTime <= 0 {
		currentTime = tool.UnixNowMilli()
	}
	result := make([]int64, 0)
	endTime := time.UnixMilli(currentTime)
	endDate := int64(tool.GetTodayDataIntByTimeStamp(currentTime))
	periodEndTime := endTime
	periodEndDate := endDate
	if prevDate, ok := addDaysToDateInt(endDate, -1); ok {
		periodEndDate = prevDate
		if prevTime, parsed := dateIntToTime(prevDate); parsed {
			periodEndTime = prevTime.AddDate(0, 0, 1).Add(-time.Millisecond)
		}
	}
	startDate := periodEndDate
	hasWeekSettle := containsSettleType(allSettleTypes, int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK))

	// Arena-like ranks use version(YYYYMMDD) as start date.
	if pointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) ||
		pointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) {
		if serverID, startMilli, weekEndMilli, ok := ParseArenaRankVersionDateInt(version); ok {
			startDate = int64(tool.GetTodayDataIntByTimeStamp(startMilli))
			if weekEndMilli < periodEndTime.UnixMilli() {
				periodEndTime = time.UnixMilli(weekEndMilli)
				periodEndDate = int64(tool.GetTodayDataIntByTimeStamp(weekEndMilli))
			}
			// If server open date is later than rank version date, settle dates start from open date.
			if openDate, hasOpenDate := getServerOpenDateInt(serverID); hasOpenDate && openDate > startDate {
				startDate = openDate
			}
		}
	}

	switch settleType {
	case int32(enum.RANK_BOARD_SETTLE_TYPE_DAY):
		for d := startDate; d <= periodEndDate; {
			// If day+week coexist, Sunday settles week only.
			appendDay := true
			if hasWeekSettle {
				if t, ok := dateIntToTime(d); ok && t.Weekday() == time.Sunday {
					appendDay = false
				}
			}
			if appendDay {
				result = append(result, d)
			}
			next, ok := nextDateInt(d)
			if !ok {
				break
			}
			d = next
		}
		return result

	case int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK):
		startTime, ok := dateIntToTime(startDate)
		if !ok {
			return result
		}
		// move to first Sunday on/after startTime
		for startTime.Weekday() != time.Sunday {
			startTime = startTime.AddDate(0, 0, 1)
		}
		for !startTime.After(periodEndTime) {
			result = append(result, int64(startTime.Year()*10000+int(startTime.Month())*100+startTime.Day()))
			startTime = startTime.AddDate(0, 0, 7)
		}
		return result

	case int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER),
		int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER),
		int32(enum.RANK_BOARD_SETTLE_TYPE_ACTIVITY_OVER):
		// Round-win score types: version carries round start date(rsYYYYMMDD), settle at roundStart+3d.
		if settleType == int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER) &&
			(pointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT) ||
				pointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT)) {
			if roundStartDate, ok := parseRoundStartDateFromVersion(version); ok {
				roundEndDate, ok := addDaysToDateInt(roundStartDate, 3)
				if ok && roundEndDate <= endDate {
					result = append(result, roundEndDate)
				}
				return result
			}
		}
		// Season-win score type: version carries season start date(stYYYYMMDD).
		if settleType == int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER) &&
			pointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT) {
			seasonStartDate, ok := parseSeasonStartDateFromVersion(version)
			if !ok {
				return result
			}
			seasonEndDate, ok := getGloryArenaSeasonEndDate(seasonStartDate, version)
			if ok && seasonEndDate <= endDate {
				result = append(result, seasonEndDate)
			}
			return result
		}
		// Triggered by caller-side condition; fallback to "today".
		if containsSettleType(allSettleTypes, settleType) {
			result = append(result, endDate)
		}
		return result
	}
	return result
}

func nextDateInt(date int64) (int64, bool) {
	t, ok := dateIntToTime(date)
	if !ok {
		return 0, false
	}
	t = t.AddDate(0, 0, 1)
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day()), true
}

func containsSettleType(settleTypes []int32, target int32) bool {
	for _, v := range settleTypes {
		if v == target {
			return true
		}
	}
	return false
}

func addDaysToDateInt(date int64, days int) (int64, bool) {
	t, ok := dateIntToTime(date)
	if !ok {
		return 0, false
	}
	t = t.AddDate(0, 0, days)
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day()), true
}

func getServerOpenDateInt(serverID int32) (int64, bool) {
	if serverID <= 0 {
		return 0, false
	}
	info, err := easyDB.GetServerEntityByWhere[serverOpenTimeMeta](map[string]interface{}{"server_id": serverID})
	if err != nil || info == nil || info.ServerOpenTime <= 0 {
		return 0, false
	}
	openDate := int64(tool.GetTodayDataIntByTimeStamp(info.ServerOpenTime))
	return openDate, true
}

// startTimeMilli排行榜开始时间，endTimeMilli排行榜理论上的结束时间，cappedCheckTime这个是当前应该的结束时间，比如竞技场排行榜7天结束，当前是第3天，那cappedCheckTime是3天
func GetArenaRankSettleTaskSettleDates(settleType int32, allSettleTypes []int32, startTimeMilli, endTimeMilli, cappedCheckTime int64) []int64 {
	if cappedCheckTime <= 0 {
		cappedCheckTime = tool.UnixNowMilli()
	}
	result := make([]int64, 0)
	if startTimeMilli > cappedCheckTime {
		return result
	}

	start := time.UnixMilli(startTimeMilli)
	startDateTime := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	end := time.UnixMilli(cappedCheckTime)
	endDateTime := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.Local)
	hasWeekSettle := containsSettleType(allSettleTypes, int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK))

	switch settleType {
	case int32(enum.RANK_BOARD_SETTLE_TYPE_DAY):
		for d := startDateTime; !d.After(endDateTime); d = d.AddDate(0, 0, 1) {
			// If day+week coexist, Sunday settles week only.
			if hasWeekSettle && d.Weekday() == time.Sunday {
				continue
			}
			result = append(result, int64(d.Year()*10000+int(d.Month())*100+d.Day()))
		}
		return result

	case int32(enum.RANK_BOARD_SETTLE_TYPE_WEEK):
		// move to first Sunday on/after startTime
		firstSunday := startDateTime
		for firstSunday.Weekday() != time.Sunday {
			firstSunday = firstSunday.AddDate(0, 0, 1)
		}
		for d := firstSunday; !d.After(endDateTime); d = d.AddDate(0, 0, 7) {
			result = append(result, int64(d.Year()*10000+int(d.Month())*100+d.Day()))
		}
		return result

	case int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER):
	case int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER):
		endTime := time.UnixMilli(endTimeMilli)
		endTimeDate := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, time.Local)
		result = append(result, int64(endTimeDate.Year()*10000+int(endTimeDate.Month())*100+endTimeDate.Day()))
		return result
	}
	return result
}
