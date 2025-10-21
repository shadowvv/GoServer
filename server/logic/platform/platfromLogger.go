package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

func InitLogger(env enum.Environment) {
	switch env {
	case enum.ENV_DEVELOP:
		logger.InitLogger("config/developLoggerConfig.yaml")
	case enum.ENV_TEST:
		logger.InitLogger("config/testLoggerConfig.yaml")
	case enum.ENV_PRODUCT:
		logger.InitLogger("config/prodLoggerConfig.yaml")
	}
}

func Info(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
	}
	logger.Info(msg, fields...)
}

func Debug(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
	}
	logger.Debug(msg, fields...)
}

func Error(msg string, user logicInterface.UserBaseInterface) {
	fields := []zap.Field{
		zap.Int32("serverId", user.GetUserServerId()),
		zap.Int64("userId", user.GetUserId()),
		zap.String("account", user.GetUserAccount()),
	}
	logger.Error(msg, fields...)
}
