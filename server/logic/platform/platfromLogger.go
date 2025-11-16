package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

func InfoWithFunction(enum enum.FuncEnum, msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSession().GetID()),
		zap.String("functionId", string(enum)),
	}
	logger.Info(msg, fields...)
}

func ErrorWithFunction(enum enum.FuncEnum, msg string, user logicInterface.UserBaseInterface, err error) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSession().GetID()),
		zap.String("function", string(enum)),
	}
	if err != nil {
		fields = append(fields, zap.String("error", err.Error()))
	}
	logger.Error(msg, fields...)
}

func Info(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSession().GetID()),
	}
	logger.Info(msg, fields...)
}

func Error(msg string, user logicInterface.UserBaseInterface, err error) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", user.GetSession().GetID()),
	}
	if err != nil {
		fields = append(fields, zap.String("error", err.Error()))
	}
	logger.Error(msg, fields...)
}
