package enum

// 活动服务器选择类型
type ActivityServerType int32

const (
	ActivityServerType_Single ActivityServerType = 1 // 单服活动
	ActivityServerType_Multi  ActivityServerType = 2 // 跨服活动
)

func IsValidActivityServerType(activityServerType int32) bool {
	switch activityServerType {
	case int32(ActivityServerType_Single):
		return true
	case int32(ActivityServerType_Multi):
		return true
	default:
		return false
	}
}
