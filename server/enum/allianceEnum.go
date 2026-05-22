package enum

import (
	"time"

	"github.com/drop/GoServer/server/tool"
)

type AlliancePermit int32

const (
	CHANGE_ALLIANCE_INFO        AlliancePermit = iota + 1 // 修改联盟信息
	CHANGE_ALLIANCE_MEMBER_ROLE                           // 挑战成员职位
	APPROVE_ALLIANCE_APPLY                                // 管理申请列表
	LEAVE_ALLIANCE                                        // 退出联盟
)

func IsValidPermit(permit int32) bool {
	if AlliancePermit(permit) < CHANGE_ALLIANCE_INFO || AlliancePermit(permit) > LEAVE_ALLIANCE {
		return false
	}
	return true
}

const (
	AllianceEnterType_Apply               int32 = 0 // 申请加入
	AllianceEnterType_Free                int32 = 1 // 自由加入
	AllianceEnterType_AlreadyApply        int32 = 2 // 已申请
	AllianceEnterType_Condition_NOT_MATCH int32 = 3 // 条件不满足
)

const (
	GetServerAllianceInfoMaxCount                  = 20                   // 查询服务器联盟的最大个数
	ApplyListMaxLength                       int32 = 50                   // 联盟申请列表的最大长度
	AllianceLeaderTransferCoLeaderMaxOffline       = 3 * tool.DAY_MILLI   // 联盟盟主离线多长时间传位给副盟主
	AllianceLeaderTransferMemberMaxOffline         = 7 * tool.DAY_MILLI   // 联盟盟主离线多长时间传位个普通成员
	AllianceAutoDissolveOfflineThreshold           = 7 * tool.DAY_MILLI   // 联盟玩家离线超过多长时间解散联盟
	AllianceApplyExpireDurationMillis              = 24 * tool.HOUR_MILLI // 申请加入联盟超时时间

	AllianceHeartbeatCheckInterval = 30 * time.Minute // 联盟心跳时间
	AllianceUpdateFlushInterval    = 2 * time.Second  // 联盟入库时间间隔
)
