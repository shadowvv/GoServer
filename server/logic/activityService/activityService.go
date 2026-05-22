package activityService

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/model"
)

// 活动服务器，分为网关和游戏服2个服务器，网关负责服务器活动的开启和关闭。将获得数据和活动配置写入redis和数据库,游戏服从redis中读取获得获得数据

// 获取活动版本 data_serverId/serverUnitIndex_openCount 日期_服务器id/服务器组id_开启次数
func getActivityVersion(date string, index int32, openCount int32) string {
	return fmt.Sprintf("d%ss%dc%d", date, index, openCount)
}

func isLoopActivity(activity *model.ServerActivityConfigEntity) bool {
	if len(activity.WeekOpenDays) != 0 || len(activity.MonthOpenDays) != 0 || activity.LoopActivity {
		return true
	}
	return false
}
