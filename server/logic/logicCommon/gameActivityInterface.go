package logicCommon

type GameActivityInterface interface {
	GetActivityId() int32
	GetVersion() string
	GetOpenTime() int64
	GetSettleTime() int64
	GetEndTime() int64
}

type GameActivityConfigInterface interface {
	GetActivityId() int32
	GetAttendUnlockId() []int32
}

type GameActivityServiceInterface interface {
	// 判断活动是否结算
	IsActivitySettled(serverId int32, activityId int32, version string) bool
	// 获取所有开启的活动
	GetAllOpenActivityByServerId(serverId int32) []GameActivityInterface
	// 判断活动是否开启
	IsActivityOpen(serverId int32, activityId int32) GameActivityInterface
	// 获得获得配置
	GetActivityConfig(id int32) GameActivityConfigInterface
	// 重新加载
	Reload()
}
