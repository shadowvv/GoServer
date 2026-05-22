package logicCommon

import (
	"strconv"
	"strings"
	"time"
)

func ParseArenaRankVersionDateInt(version string) (serverId int32, startMilli int64, endMilli int64, ok bool) {
	serverId, monday, ok := ParseArenaRankVersion(version)
	if !ok {
		return 0, 0, 0, false
	}
	startTime, err := time.ParseInLocation("20060102", monday, time.Local)
	if err != nil {
		return 0, 0, 0, false
	}

	startMilli = startTime.UnixMilli()
	endMilli = startTime.AddDate(0, 0, 7).Add(-time.Millisecond).UnixMilli()
	return serverId, startMilli, endMilli, true
}

func ParseArenaRankVersion(version string) (serverId int32, startDate string, result bool) {
	if version == "" {
		return 0, "", false
	}
	if !strings.HasPrefix(version, "s") {
		return 0, "", false
	}

	// Strict format: s{sid}:t{yyyyMMdd}
	if idx := strings.Index(version, ":"); idx > 1 && idx < len(version)-1 {
		serverPart := version[1:idx]
		datePart := version[idx+1:]
		if !strings.HasPrefix(datePart, "t") || len(datePart) < 2 {
			return 0, "", false
		}
		datePart = datePart[1:]
		if _, ok := parseYYYYMMDDToDateInt(datePart); ok {
			serverID, err := strconv.Atoi(serverPart)
			if err == nil {
				return int32(serverID), datePart, true
			}
		}
	}
	return 0, "", false
}
