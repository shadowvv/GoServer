package main

import (
	"github.com/drop/GoServer/server/logic/gameController"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/raid"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/wordFilter"
)

func main() {
	gamePlatform.BootGamePlatform()
	platform.StartPprofByEnv()
	logger.InfoWithSprintf("[platform] init model")
	gameController.InitGameController(gamePlatform.GetEventBus(), gamePlatform.GetRouter(), gamePlatform.GetSessionManager(), gamePlatform.GetDispatcher(), gamePlatform.GetMessageSender(), gamePlatform.GetUnlockService(), gamePlatform.GetActivityService())
	model.InitModel(gamePlatform.GetUnlockService(), gameController.GetMailService(), gamePlatform.GetEventBus(), gamePlatform.GetMessageSender(), itemService.GetItemService(), gamePlatform.GetActivityService(), gamePlatform.GetRpcMessageSender())
	raid.InitRaid(gamePlatform.GetEventBus(), gamePlatform.GetUnlockService(), gamePlatform.GetMessageSender(), gamePlatform.GetSceneService(), gamePlatform.GetPassService(), gameController.GetMailService(), nodeConfig.NodeId)
	gameController.RegisterAllMessage()

	platform.ListenSignal(&gameSignalHandler{})

	logger.InfoWithSprintf("Server Start!!!!!")
	select {}
}

type gameSignalHandler struct {
}

var _ logicCommon.SignalHooker = (*gameSignalHandler)(nil)

func (g gameSignalHandler) KickAllPlayer() {

}

func (g gameSignalHandler) AfterAllConfigReload() {
	err := wordFilter.Reload()
	if err != nil {
		logger.ErrorBySprintf("wordFilter reload error: %v", err)
		return
	}
	gamePlatform.GetActivityService().Reload()
}
