package logicCommon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
)

var (
	ErrGloryArenaServerNotFound  = errors.New("glory arena server not found")
	ErrGloryArenaServerListEmpty = errors.New("glory arena open server list empty")
)

const gloryArenaRoundOpenOffset = 30 * time.Minute

type gloryArenaRoundState struct {
	SeasonType         enum.GloryArenaSeasonType
	RoundIndexInSeason int32
	IsRoundOpen        bool
	RoundStart         int64
	RoundEnd           int64
}

type GloryArenaOpsServerState struct {
	ServerID           int32                     `json:"serverId"`
	SeasonType         enum.GloryArenaSeasonType `json:"seasonType"`
	RoundIndexInSeason int32                     `json:"roundIndexInSeason"`
	IsRoundOpen        bool                      `json:"isRoundOpen"`
	RoundStart         int64                     `json:"roundStart"`
	RoundEnd           int64                     `json:"roundEnd"`
	GroupVersion       string                    `json:"groupVersion"`
	SeasonVersion      string                    `json:"seasonVersion"`
	GroupServerIDs     []int32                   `json:"groupServerIds"`
}

func GetGloryArenaCrossServerResultByTime(sortedServerInfo []ServerInfoInterface, currentTime int64) (map[int32]*GloryArenaOpsServerState, error) {
	allResults := make(map[int32]*GloryArenaOpsServerState)
	for _, serverInfo := range sortedServerInfo {
		result, err := CalculateGloryArenaCrossServerResult(sortedServerInfo, serverInfo, currentTime)
		if err != nil {
			return nil, err
		}
		allResults[serverInfo.GetServerId()] = result
	}
	return allResults, nil
}

func CalculateGloryArenaCrossServerResult(allServers []ServerInfoInterface, serverInfo ServerInfoInterface, currentTime int64) (*GloryArenaOpsServerState, error) {
	if len(allServers) == 0 {
		return nil, ErrGloryArenaServerListEmpty
	}

	targetIndex := -1
	for i, info := range allServers {
		if info.GetServerId() == serverInfo.GetServerId() {
			targetIndex = i
			break
		}
	}
	if targetIndex < 0 {
		return nil, ErrGloryArenaServerNotFound
	}

	// 赛季推进规则（业务口径）：
	// PRE(开服第2天起) -> FIRST -> SECOND -> POST
	// 若 FIRST 开不起来，则继续 PRE。
	firstSeasonStart := getFirstSeasonStartTime(serverInfo.GetServerOpenTime())
	if currentTime < firstSeasonStart {
		return buildPreSeasonState(serverInfo, currentTime)
	}
	return buildCommonSeasonState(serverInfo, allServers, currentTime)
}

func buildPreSeasonState(serverInfo ServerInfoInterface, currentTime int64) (*GloryArenaOpsServerState, error) {
	round := preseasonRoundState(serverInfo.GetServerOpenTime(), currentTime)
	roundIndex := int32(1)
	isRoundOpen := false
	if round.RoundIndexInSeason > 0 {
		roundIndex = round.RoundIndexInSeason
	}
	isRoundOpen = round.IsRoundOpen
	effectiveSize := int32(1)
	groupStartServerID := serverInfo.GetServerId()

	result := &GloryArenaOpsServerState{
		ServerID:           serverInfo.GetServerId(),
		SeasonType:         enum.GLORY_ARENA_SEASON_TYPE_PRE,
		RoundIndexInSeason: roundIndex,
		IsRoundOpen:        isRoundOpen,
		RoundStart:         round.RoundStart,
		RoundEnd:           round.RoundEnd,
		GroupVersion:       getGloryArenaVersion(int32(enum.GLORY_ARENA_SEASON_TYPE_PRE), groupStartServerID, effectiveSize, round.RoundStart, roundIndex),
		SeasonVersion:      getGloryArenaSeasonVersionByState(int32(enum.GLORY_ARENA_SEASON_TYPE_PRE), groupStartServerID, effectiveSize, round.RoundStart, roundIndex),
		GroupServerIDs:     []int32{serverInfo.GetServerId()},
	}
	return result, nil
}

func preseasonRoundState(serverOpenTime int64, currentTime int64) *gloryArenaRoundState {
	state := &gloryArenaRoundState{
		SeasonType: enum.GLORY_ARENA_SEASON_TYPE_PRE,
	}
	openDayZero := time.UnixMilli(tool.GetTodayZeroByTimeStamp(serverOpenTime))
	round1Start := openDayZero.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
	round1End := round1Start.Add(3*24*time.Hour - gloryArenaRoundOpenOffset)

	if currentTime < round1Start.UnixMilli() {
		state.IsRoundOpen = false
		state.RoundIndexInSeason = 1
		state.RoundStart = round1Start.UnixMilli()
		state.RoundEnd = round1End.UnixMilli()
		return state
	}

	if currentTime >= round1Start.UnixMilli() && currentTime < round1End.UnixMilli() {
		state.IsRoundOpen = true
		state.RoundIndexInSeason = 1
		state.RoundStart = round1Start.UnixMilli()
		state.RoundEnd = round1End.UnixMilli()
		return state
	}

	round2Start := round1End.Add(gloryArenaRoundOpenOffset)
	round2End := round2Start.Add(3*24*time.Hour - gloryArenaRoundOpenOffset)
	nextSeasonStartTuesday := nextTuesdayAfter(round1End)
	// Preseason round2 can only open when it fully ends before the upcoming Tuesday,
	// so season 1 can switch on Tuesday without overlap.
	if !round2End.After(nextSeasonStartTuesday) {
		if currentTime >= round1End.UnixMilli() && currentTime < round2Start.UnixMilli() {
			state.IsRoundOpen = false
			state.RoundIndexInSeason = 2
			state.RoundStart = round2Start.UnixMilli()
			state.RoundEnd = round2End.UnixMilli()
			return state
		}
		if currentTime >= round2Start.UnixMilli() && currentTime < round2End.UnixMilli() {
			state.IsRoundOpen = true
			state.RoundIndexInSeason = 2
			state.RoundStart = round2Start.UnixMilli()
			state.RoundEnd = round2End.UnixMilli()
			return state
		}
	}

	// 第一轮结束后（含第二轮不允许开启场景）返回下一次开启窗口，避免前端拿不到 start/end。
	state.IsRoundOpen = false
	state.RoundIndexInSeason = 1
	state.RoundStart = nextSeasonStartTuesday.UnixMilli()
	state.RoundEnd = nextSeasonStartTuesday.Add(3*24*time.Hour - gloryArenaRoundOpenOffset).UnixMilli()
	return state
}

func nextTuesdayAfter(base time.Time) time.Time {
	tuesday := tool.WeekStartByMilli(base.UnixMilli()).AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
	if !base.Before(tuesday) {
		tuesday = tuesday.AddDate(0, 0, 7)
	}
	return tuesday
}

func getFirstSeasonStartTime(serverOpenTime int64) int64 {
	openDayZero := time.UnixMilli(tool.GetTodayZeroByTimeStamp(serverOpenTime))
	round1Start := openDayZero.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
	round1End := round1Start.Add(3*24*time.Hour - gloryArenaRoundOpenOffset)
	return nextTuesdayAfter(round1End).UnixMilli()
}

func buildCommonSeasonState(serverInfo ServerInfoInterface, allServers []ServerInfoInterface, currentTime int64) (*GloryArenaOpsServerState, error) {
	if len(allServers) == 0 {
		return nil, ErrGloryArenaServerListEmpty
	}

	targetIndex := -1
	for i, info := range allServers {
		if info.GetServerId() == serverInfo.GetServerId() {
			targetIndex = i
			break
		}
	}
	if targetIndex < 0 {
		return nil, ErrGloryArenaServerNotFound
	}

	// FIRST 入口门槛：必须满足 2 服配对，且配对内服务器都到达 FIRST 可开启时间。
	firstGroupStart := (targetIndex / 2) * 2
	firstGroupEnd := firstGroupStart + 2
	if firstGroupEnd > len(allServers) || !isSeasonGroupStructureValid(allServers[firstGroupStart:firstGroupEnd], enum.GLORY_ARENA_SEASON_TYPE_FIRST) {
		return buildContinuedPreSeasonState(serverInfo, currentTime)
	}
	firstSeasonStart := int64(0)
	for i := firstGroupStart; i < firstGroupEnd; i++ {
		start := getFirstSeasonStartTime(allServers[i].GetServerOpenTime())
		if currentTime < start {
			return buildContinuedPreSeasonState(serverInfo, currentTime)
		}
		if start > firstSeasonStart {
			firstSeasonStart = start
		}
	}
	if firstSeasonStart <= 0 {
		return buildContinuedPreSeasonState(serverInfo, currentTime)
	}

	round := seasonRoundState(currentTime, firstSeasonStart)
	if round == nil {
		return nil, ErrGloryArenaServerNotFound
	}
	globalRound := getGlobalRoundBySeasonStart(currentTime, firstSeasonStart)
	selectedSeason := getSeasonTypeByGlobalRound(globalRound)

	selectedStart := firstGroupStart
	selectedEnd := firstGroupEnd
	selectedSize := int32(2)
	foundGroup := false
	for _, candidateSize := range getSeasonCandidateGroupSizes(selectedSeason) {
		if candidateSize <= 0 {
			continue
		}
		start := (targetIndex / int(candidateSize)) * int(candidateSize)
		end := start + int(candidateSize)
		if end > len(allServers) {
			continue
		}
		if !isSeasonGroupStructureValid(allServers[start:end], selectedSeason) {
			continue
		}
		// FIRST 仍需保证同组都已到达 FIRST 可开启时间。
		if selectedSeason == enum.GLORY_ARENA_SEASON_TYPE_FIRST {
			ready := true
			for i := start; i < end; i++ {
				if currentTime < getFirstSeasonStartTime(allServers[i].GetServerOpenTime()) {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}
		}
		selectedStart = start
		selectedEnd = end
		selectedSize = candidateSize
		foundGroup = true
		break
	}
	if !foundGroup && selectedSeason == enum.GLORY_ARENA_SEASON_TYPE_FIRST {
		return buildContinuedPreSeasonState(serverInfo, currentTime)
	}

	groupServerIDs := make([]int32, 0, selectedEnd-selectedStart)
	for i := selectedStart; i < selectedEnd; i++ {
		groupServerIDs = append(groupServerIDs, allServers[i].GetServerId())
	}

	groupStartServerID := serverInfo.GetServerId()
	if len(groupServerIDs) > 0 {
		groupStartServerID = groupServerIDs[0]
	}

	result := &GloryArenaOpsServerState{
		ServerID:           serverInfo.GetServerId(),
		SeasonType:         selectedSeason,
		RoundIndexInSeason: round.RoundIndexInSeason,
		IsRoundOpen:        round.IsRoundOpen,
		RoundStart:         round.RoundStart,
		RoundEnd:           round.RoundEnd,
		GroupVersion:       getGloryArenaVersion(int32(selectedSeason), groupStartServerID, selectedSize, round.RoundStart, round.RoundIndexInSeason),
		SeasonVersion:      getGloryArenaSeasonVersionByState(int32(selectedSeason), groupStartServerID, selectedSize, round.RoundStart, round.RoundIndexInSeason),
		GroupServerIDs:     groupServerIDs,
	}
	return result, nil
}

func buildContinuedPreSeasonState(serverInfo ServerInfoInterface, currentTime int64) (*GloryArenaOpsServerState, error) {
	roundIndex, isRoundOpen, roundStart, roundEnd := getWeeklyTwoRoundWindowState(currentTime)
	groupStartServerID := serverInfo.GetServerId()
	effectiveSize := int32(1)
	return &GloryArenaOpsServerState{
		ServerID:           serverInfo.GetServerId(),
		SeasonType:         enum.GLORY_ARENA_SEASON_TYPE_PRE,
		RoundIndexInSeason: roundIndex,
		IsRoundOpen:        isRoundOpen,
		RoundStart:         roundStart,
		RoundEnd:           roundEnd,
		GroupVersion:       getGloryArenaVersion(int32(enum.GLORY_ARENA_SEASON_TYPE_PRE), groupStartServerID, effectiveSize, roundStart, roundIndex),
		SeasonVersion:      getGloryArenaSeasonVersionByState(int32(enum.GLORY_ARENA_SEASON_TYPE_PRE), groupStartServerID, effectiveSize, roundStart, roundIndex),
		GroupServerIDs:     []int32{serverInfo.GetServerId()},
	}, nil
}

func getGlobalRoundBySeasonStart(currentTime int64, seasonStart int64) int32 {
	if currentTime <= 0 || seasonStart <= 0 {
		return 1
	}
	seasonStartMonday := tool.WeekStartByMilli(seasonStart)
	nowWeekMonday := tool.WeekStartByMilli(currentTime)
	if nowWeekMonday.Before(seasonStartMonday) {
		return 1
	}
	weeks := int32(nowWeekMonday.Sub(seasonStartMonday) / (7 * 24 * time.Hour))
	if weeks < 0 {
		return 1
	}

	round1Start := nowWeekMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
	round1End := nowWeekMonday.AddDate(0, 0, 4)
	round2Start := round1End.Add(gloryArenaRoundOpenOffset)
	round2End := nowWeekMonday.AddDate(0, 0, 7)
	now := time.UnixMilli(currentTime)

	switch {
	case now.Before(round1Start):
		return weeks*2 + 1
	case now.Before(round1End):
		return weeks*2 + 1
	case now.Before(round2Start):
		return weeks*2 + 2
	case now.Before(round2End):
		return weeks*2 + 2
	default:
		return (weeks+1)*2 + 1
	}
}

func getSeasonTypeByGlobalRound(globalRound int32) enum.GloryArenaSeasonType {
	switch {
	case globalRound <= 4:
		return enum.GLORY_ARENA_SEASON_TYPE_FIRST
	case globalRound <= 8:
		return enum.GLORY_ARENA_SEASON_TYPE_SECOND
	default:
		return enum.GLORY_ARENA_SEASON_TYPE_POST
	}
}

func getOpenNatureWeekDistance(openTime int64, currentTime int64) int32 {
	if openTime <= 0 || currentTime <= openTime {
		return 1
	}
	openWeekStart := tool.WeekStartByMilli(openTime)
	currentWeekStart := tool.WeekStartByMilli(currentTime)
	diffWeeks := int(currentWeekStart.Sub(openWeekStart) / (7 * 24 * time.Hour))
	if diffWeeks < 0 {
		return 1
	}
	return int32(diffWeeks + 1)
}

func getTargetGroupSizeBySeasonType(seasonType enum.GloryArenaSeasonType) int32 {
	switch seasonType {
	case enum.GLORY_ARENA_SEASON_TYPE_POST:
		return 8
	case enum.GLORY_ARENA_SEASON_TYPE_SECOND:
		return 4
	case enum.GLORY_ARENA_SEASON_TYPE_FIRST:
		return 2
	default:
		return 1
	}
}

func getSeasonCandidateGroupSizes(seasonType enum.GloryArenaSeasonType) []int32 {
	switch seasonType {
	case enum.GLORY_ARENA_SEASON_TYPE_POST:
		return []int32{8, 4, 2}
	case enum.GLORY_ARENA_SEASON_TYPE_SECOND:
		return []int32{4, 2}
	case enum.GLORY_ARENA_SEASON_TYPE_FIRST:
		return []int32{2}
	default:
		return []int32{1}
	}
}

func isServerGroupEligibleForSeason(servers []ServerInfoInterface, weekByServerID map[int32]int32, seasonType enum.GloryArenaSeasonType) bool {
	for _, server := range servers {
		week := weekByServerID[server.GetServerId()]
		if !isWeekEligibleForSeason(week, seasonType) {
			return false
		}
	}
	return true
}

// Non-preseason grouping must follow odd-even adjacent pairing: 1-2, 3-4, ...
// This prevents unmatched odd servers from opening season mode alone.
func isSeasonGroupStructureValid(servers []ServerInfoInterface, seasonType enum.GloryArenaSeasonType) bool {
	if seasonType == enum.GLORY_ARENA_SEASON_TYPE_PRE {
		return len(servers) >= 1
	}
	if len(servers) < 2 || len(servers)%2 != 0 {
		return false
	}
	for i := 0; i < len(servers); i += 2 {
		left := servers[i]
		right := servers[i+1]
		if left == nil || right == nil {
			return false
		}
		leftID := left.GetServerId()
		rightID := right.GetServerId()
		if leftID%2 == 0 || rightID != leftID+1 {
			return false
		}
	}
	return true
}

func isWeekEligibleForSeason(week int32, seasonType enum.GloryArenaSeasonType) bool {
	switch seasonType {
	case enum.GLORY_ARENA_SEASON_TYPE_POST:
		return week >= 6
	case enum.GLORY_ARENA_SEASON_TYPE_SECOND:
		return week >= 4 && week <= 5
	case enum.GLORY_ARENA_SEASON_TYPE_FIRST:
		return week >= 2 && week <= 3
	default:
		// Pre season is the final rollback target.
		return week >= 1
	}
}

func getFallbackSeasonType(seasonType enum.GloryArenaSeasonType) (enum.GloryArenaSeasonType, bool) {
	switch seasonType {
	case enum.GLORY_ARENA_SEASON_TYPE_POST:
		return enum.GLORY_ARENA_SEASON_TYPE_SECOND, true
	case enum.GLORY_ARENA_SEASON_TYPE_SECOND:
		return enum.GLORY_ARENA_SEASON_TYPE_FIRST, true
	case enum.GLORY_ARENA_SEASON_TYPE_FIRST:
		return enum.GLORY_ARENA_SEASON_TYPE_PRE, true
	default:
		return enum.GLORY_ARENA_SEASON_TYPE_PRE, false
	}
}

func getWeeklyTwoRoundWindowState(currentTime int64) (int32, bool, int64, int64) {
	if currentTime <= 0 {
		return 1, false, 0, 0
	}

	nowWeekMonday := tool.WeekStartByMilli(currentTime)
	round1Start := nowWeekMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset) // Tue 00:30
	round1End := nowWeekMonday.AddDate(0, 0, 4)                                  // Fri 00:00
	round2Start := round1End.Add(gloryArenaRoundOpenOffset)                      // Fri 00:30
	round2End := nowWeekMonday.AddDate(0, 0, 7)                                  // Mon 00:00
	now := time.UnixMilli(currentTime)

	if now.Before(round1Start) {
		return 1, false, round1Start.UnixMilli(), round1End.UnixMilli()
	}
	if now.Before(round1End) {
		return 1, true, round1Start.UnixMilli(), round1End.UnixMilli()
	}
	if now.Before(round2Start) {
		// 轮次交接休赛窗口（00:00~00:30）
		return 2, false, round2Start.UnixMilli(), round2End.UnixMilli()
	}
	if now.Before(round2End) {
		return 2, true, round2Start.UnixMilli(), round2End.UnixMilli()
	}
	// 周维度休赛：返回下一轮（下周二）窗口。
	nextRound1Start := nowWeekMonday.AddDate(0, 0, 8).Add(gloryArenaRoundOpenOffset)
	nextRound1End := nowWeekMonday.AddDate(0, 0, 11)
	return 1, false, nextRound1Start.UnixMilli(), nextRound1End.UnixMilli()
}

func getGloryArenaVersion(seasonType, groupStartServerID, effectiveSize int32, roundStart int64, roundIndexInSeason int32) string {
	// s: season, ss: startServerId, c: serverCount, rs: roundStart, ri: roundIndexInSeason
	return fmt.Sprintf("s%d:ss%d:c%d:rs%s:ri%d", seasonType, groupStartServerID, effectiveSize, tool.GetTodayDataStringByTimeStamp(roundStart), roundIndexInSeason)
}

func getGloryArenaSeasonVersionByState(seasonType, groupStartServerID, effectiveSize int32, roundStart int64, roundIndexInSeason int32) string {
	seasonStart := getGloryArenaSeasonStart(roundStart, roundIndexInSeason)
	// s: season, ss: startServerId, c: serverCount, st: seasonStart
	return fmt.Sprintf("s%d:ss%d:c%d:st%s", seasonType, groupStartServerID, effectiveSize, tool.GetTodayDataStringByTimeStamp(seasonStart))
}

func getGloryArenaSeasonStart(roundStart int64, roundIndexInSeason int32) int64 {
	if roundStart <= 0 || roundIndexInSeason <= 1 {
		return roundStart
	}
	// Tue->Fri->Mon are all 3-day rounds; rewind to round #1 start in this season.
	return roundStart - int64(roundIndexInSeason-1)*3*24*60*60*1000
}

// GetGloryArenaSeasonVersion derives a season-level stable version from groupVersion.
// It removes round-specific parts (rs/ri) and keeps season/group identity.
func GetGloryArenaSeasonVersion(groupVersion string) string {
	if groupVersion == "" {
		return ""
	}
	parts := strings.Split(groupVersion, ":")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, "rs") || strings.HasPrefix(part, "ri") {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, ":")
}

func getGloryArenaSeasonType(week int32) enum.GloryArenaSeasonType {
	switch {
	case week <= 1:
		return enum.GLORY_ARENA_SEASON_TYPE_PRE
	case week <= 3:
		return enum.GLORY_ARENA_SEASON_TYPE_FIRST
	case week <= 5:
		return enum.GLORY_ARENA_SEASON_TYPE_SECOND
	default:
		return enum.GLORY_ARENA_SEASON_TYPE_POST
	}
}

func pickEffectiveGroupSize(total int, targetIndex int, targetSize int32) int32 {
	fallback := buildFallbackGroupSizes(targetSize)
	for _, size := range fallback {
		if size <= 1 {
			return 1
		}
		start := (targetIndex / int(size)) * int(size)
		end := start + int(size)
		if end <= total {
			return size
		}
	}
	return 1
}

func buildFallbackGroupSizes(targetSize int32) []int32 {
	switch targetSize {
	case 8:
		return []int32{8, 4, 2, 1}
	case 4:
		return []int32{4, 2, 1}
	case 2:
		return []int32{2, 1}
	default:
		return []int32{1}
	}
}

func getCommonGloryArenaRoundState(serverOpenTime int64, currentTime int64) *gloryArenaRoundState {
	state := &gloryArenaRoundState{
		SeasonType: enum.GLORY_ARENA_SEASON_TYPE_PRE,
	}
	if serverOpenTime <= 0 || currentTime <= 0 || currentTime < serverOpenTime {
		return state
	}

	week := getOpenNatureWeekDistance(serverOpenTime, currentTime)
	if week <= 1 {
		return state
	}

	// Common season starts from even week Tuesday (2/4/6... weeks from server open).
	seasonStartWeek := week
	if seasonStartWeek%2 == 1 {
		seasonStartWeek--
	}
	if seasonStartWeek < 2 {
		seasonStartWeek = 2
	}

	seasonStart := getSeasonStartTuesdayByOpenWeek(serverOpenTime, seasonStartWeek)
	return seasonRoundState(currentTime, seasonStart)
}

func seasonRoundState(currentTime int64, seasonStart int64) *gloryArenaRoundState {
	state := &gloryArenaRoundState{
		SeasonType: enum.GLORY_ARENA_SEASON_TYPE_POST,
	}
	if currentTime <= 0 || seasonStart <= 0 {
		return state
	}

	seasonStartMonday := tool.WeekStartByMilli(seasonStart)
	nowWeekMonday := tool.WeekStartByMilli(currentTime)
	if nowWeekMonday.Before(seasonStartMonday) {
		tuesday := seasonStartMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
		friday := seasonStartMonday.AddDate(0, 0, 4)
		state.RoundIndexInSeason = 1
		state.IsRoundOpen = false
		state.RoundStart = tuesday.UnixMilli()
		state.RoundEnd = friday.UnixMilli()
		return state
	}

	weeks := int(nowWeekMonday.Sub(seasonStartMonday) / (7 * 24 * time.Hour))
	if weeks < 0 {
		tuesday := seasonStartMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
		friday := seasonStartMonday.AddDate(0, 0, 4)
		state.RoundIndexInSeason = 1
		state.IsRoundOpen = false
		state.RoundStart = tuesday.UnixMilli()
		state.RoundEnd = friday.UnixMilli()
		return state
	}

	round1Start := nowWeekMonday.AddDate(0, 0, 1).Add(gloryArenaRoundOpenOffset)
	round1End := nowWeekMonday.AddDate(0, 0, 4)
	round2Start := round1End.Add(gloryArenaRoundOpenOffset)
	round2End := nowWeekMonday.AddDate(0, 0, 7)
	now := time.UnixMilli(currentTime)

	roundInWeek := int32(0)
	if now.Before(round1Start) {
		roundInWeek = 1
		state.IsRoundOpen = false
		state.RoundStart = round1Start.UnixMilli()
		state.RoundEnd = round1End.UnixMilli()
	} else if now.Before(round1End) {
		roundInWeek = 1
		state.IsRoundOpen = true
		state.RoundStart = round1Start.UnixMilli()
		state.RoundEnd = round1End.UnixMilli()
	} else if now.Before(round2Start) {
		roundInWeek = 2
		state.IsRoundOpen = false
		state.RoundStart = round2Start.UnixMilli()
		state.RoundEnd = round2End.UnixMilli()
	} else if now.Before(round2End) {
		roundInWeek = 2
		state.IsRoundOpen = true
		state.RoundStart = round2Start.UnixMilli()
		state.RoundEnd = round2End.UnixMilli()
	} else {
		roundInWeek = 1
		state.IsRoundOpen = false
		nextTuesday := nowWeekMonday.AddDate(0, 0, 8).Add(gloryArenaRoundOpenOffset)
		nextFriday := nowWeekMonday.AddDate(0, 0, 11)
		state.RoundStart = nextTuesday.UnixMilli()
		state.RoundEnd = nextFriday.UnixMilli()
	}

	globalRound := int32(weeks)*2 + roundInWeek
	if now.Before(round1Start) {
		// Monday belongs to the upcoming Tuesday round.
		globalRound = int32(weeks)*2 + 1
	}
	if !now.Before(round2End) {
		// Safety branch (should rarely happen): move to next week's round1.
		globalRound = int32(weeks+1)*2 + 1
	}
	state.RoundIndexInSeason = ((globalRound - 1) % 4) + 1
	return state
}

func getSeasonStartTuesdayByOpenWeek(serverOpenTime int64, seasonStartWeek int32) int64 {
	openMonday := tool.WeekStartByMilli(serverOpenTime)
	offsetDays := int(seasonStartWeek-1)*7 + 1
	return openMonday.AddDate(0, 0, offsetDays).Add(gloryArenaRoundOpenOffset).UnixMilli()
}

func LoadGloryArenaOpsStateByServerID(serverID int32) *GloryArenaOpsServerState {
	if serverID <= 0 || dbService.RDB == nil {
		return nil
	}
	rawState, err := dbService.RDB.HGet(context.Background(), enum.GetGloryArenaOpsStateKey(), strconv.FormatInt(int64(serverID), 10)).Result()
	if err != nil || rawState == "" {
		if err != nil && err != redis.Nil {
			logger.ErrorBySprintf("[allianceModel] load glory arena ops state failed serverId:%d err:%v", serverID, err)
		}
		return nil
	}
	state := &GloryArenaOpsServerState{}
	if err = json.Unmarshal([]byte(rawState), state); err != nil {
		logger.ErrorBySprintf("[allianceModel] unmarshal glory arena ops state failed serverId:%d err:%v", serverID, err)
		return nil
	}
	return state
}

func ParseGloryArenaRankVersionDateInt(version string) (startMilli int64, endMilli int64, ok bool) {
	if version == "" {
		return 0, 0, false
	}

	// Round rank version: ...:rsYYYYMMDD:...
	if roundStartDate, parsed := parseRoundStartDateFromVersion(version); parsed {
		roundStartTime, parsedTime := dateIntToTime(roundStartDate)
		if !parsedTime {
			return 0, 0, false
		}
		startMilli = roundStartTime.UnixMilli()
		// Round window settles at roundStart + 3 days (day-end inclusive).
		endMilli = roundStartTime.AddDate(0, 0, 3).Add(-time.Millisecond).UnixMilli()
		return startMilli, endMilli, true
	}

	// Season rank version: ...:stYYYYMMDD
	if seasonStartDate, parsed := parseSeasonStartDateFromVersion(version); parsed {
		seasonStartTime, parsedStart := dateIntToTime(seasonStartDate)
		if !parsedStart {
			return 0, 0, false
		}
		seasonEndDate, parsedEnd := getGloryArenaSeasonEndDate(seasonStartDate, version)
		if !parsedEnd {
			return 0, 0, false
		}
		seasonEndTime, parsedEndTime := dateIntToTime(seasonEndDate)
		if !parsedEndTime {
			return 0, 0, false
		}

		startMilli = seasonStartTime.UnixMilli()
		// Inclusive end of season-end day.
		endMilli = seasonEndTime.AddDate(0, 0, 1).Add(-time.Millisecond).UnixMilli()
		return startMilli, endMilli, true
	}

	return 0, 0, false
}

func parseRoundStartDateFromVersion(version string) (startTime int64, result bool) {
	if version == "" {
		return 0, false
	}
	parts := strings.Split(version, ":")
	for _, part := range parts {
		if strings.HasPrefix(part, "rs") && len(part) > 2 {
			return parseYYYYMMDDToDateInt(part[2:])
		}
	}
	// compatibility: plain YYYYMMDD as version
	return parseYYYYMMDDToDateInt(version)
}

func parseSeasonStartDateFromVersion(version string) (startTime int64, result bool) {
	if version == "" {
		return 0, false
	}
	parts := strings.Split(version, ":")
	for _, part := range parts {
		if strings.HasPrefix(part, "st") && len(part) > 2 {
			return parseYYYYMMDDToDateInt(part[2:])
		}
	}
	// compatibility: plain YYYYMMDD as version
	return parseYYYYMMDDToDateInt(version)
}

func getGloryArenaSeasonEndDate(seasonStartDate int64, version string) (endTime int64, result bool) {
	seasonStartTime, ok := dateIntToTime(seasonStartDate)
	if !ok {
		return 0, false
	}
	seasonType, hasSeasonType := parseSeasonTypeFromVersion(version)
	if hasSeasonType && seasonType == int32(enum.GLORY_ARENA_SEASON_TYPE_PRE) {
		round1End := seasonStartTime.AddDate(0, 0, 3)
		weekSaturday := tool.WeekStartByMilli(round1End.UnixMilli()).AddDate(0, 0, 5)
		seasonEnd := round1End
		// Keep same rule as preseasonRoundState: only open round2 when round1End < weekSaturday.
		if round1End.Before(weekSaturday) {
			seasonEnd = round1End.AddDate(0, 0, 3)
		}
		return int64(seasonEnd.Year()*10000 + int(seasonEnd.Month())*100 + seasonEnd.Day()), true
	}
	// Non-pre seasons: 4 rounds in one season; settle at round4 end day.
	seasonEnd := seasonStartTime.AddDate(0, 0, 13)
	return int64(seasonEnd.Year()*10000 + int(seasonEnd.Month())*100 + seasonEnd.Day()), true
}

func parseSeasonTypeFromVersion(version string) (seasonType int32, result bool) {
	if version == "" {
		return 0, false
	}
	parts := strings.Split(version, ":")
	for _, part := range parts {
		if strings.HasPrefix(part, "s") && len(part) > 1 && !strings.HasPrefix(part, "ss") && !strings.HasPrefix(part, "st") {
			v, err := strconv.Atoi(part[1:])
			if err != nil {
				return 0, false
			}
			return int32(v), true
		}
	}
	return 0, false
}
