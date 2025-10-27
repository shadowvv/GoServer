package platform

import (
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

func Info(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSessionId()),
	}
	logger.Info(msg, fields...)
}

func Debug(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSessionId()),
	}
	logger.Debug(msg, fields...)
}

func Error(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSessionId()),
	}
	logger.Error(msg, fields...)
}
