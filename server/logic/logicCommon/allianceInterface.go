package logicCommon

import (
	"context"
	"strconv"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
)

type AllianceInterface interface {
	GetAllianceId() int64
}

type AllianceItemServiceInterface interface {
	// 联盟道具操作
	AllianceItemOperation(itemId int32, num int32, operationType int32) error
}

func GetAllianceMemberId(allianceId int64) []int64 {
	if allianceId <= 0 || dbService.RDB == nil {
		return []int64{}
	}

	key := enum.GetAllianceMemberInfoKey(allianceId)
	memberIDs, err := dbService.RDB.SMembers(context.Background(), key).Result()
	if err != nil {
		logger.ErrorBySprintf("[redis] get alliance member ids failed allianceId=%d key=%s err=%v", allianceId, key, err)
		return []int64{}
	}

	result := make([]int64, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		userID, parseErr := strconv.ParseInt(memberID, 10, 64)
		if parseErr != nil {
			logger.ErrorBySprintf("[redis] parse alliance member id failed allianceId=%d key=%s memberId=%s err=%v", allianceId, key, memberID, parseErr)
			continue
		}
		if userID <= 0 {
			continue
		}
		result = append(result, userID)
	}
	return result
}
