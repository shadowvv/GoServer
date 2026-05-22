package platformLogger

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

func InfoWithUser(msg string, user logicCommon.UserBaseInterface) {
	if user == nil {
		logger.InfoWithSprintf(msg)
		return
	}
	sessionId := int64(0)
	if user.GetSession() != nil {
		sessionId = user.GetSession().GetID()
	}
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", sessionId),
	}
	logger.InfoWithZapFields(msg, fields...)
}

func ErrorWithUser(msg string, user logicCommon.UserBaseInterface, err error) {
	if user == nil {
		logger.ErrorWithZapFields(msg)
		return
	}
	sessionId := int64(0)
	if user.GetSession() != nil {
		sessionId = user.GetSession().GetID()
	}
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
		zap.Int64("sessionId", sessionId),
	}
	if err != nil {
		fields = append(fields, zap.String("error", err.Error()))
	}

	logger.ErrorWithZapFields(msg, fields...)
}
