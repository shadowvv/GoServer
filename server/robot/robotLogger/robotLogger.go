package robotLogger

import (
	"fmt"

	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/service/logger"
)

type RobotLogContext interface {
	GetAccount() string
	GetServerID() int32
	GetStateName() string
}

// 带用户信息的日志
func InfoWithRobot(user robotCommon.RobotInterface, msg string) {
	if user != nil {
		logger.InfoWithSprintf("[INFO] [%s]%s %s", user.GetName(), formatRobotContext(user), msg)
	} else {
		logger.InfoWithSprintf("[INFO] %s", msg)
	}
}

// 带用户信息的错误日志
func ErrorWithRobot(user robotCommon.RobotInterface, msg string) {
	if user != nil {
		logger.ErrorBySprintf("[ERROR] [%s]%s %s", user.GetName(), formatRobotContext(user), msg)
	} else {
		logger.ErrorBySprintf("[ERROR] %s", msg)
	}
}

func formatRobotContext(user robotCommon.RobotInterface) string {
	ctx, ok := user.(RobotLogContext)
	if !ok {
		return ""
	}
	return fmt.Sprintf("[account=%s][serverID=%d][state=%s]", ctx.GetAccount(), ctx.GetServerID(), ctx.GetStateName())
}
