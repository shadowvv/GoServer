package main

import (
	"github.com/drop/GoServer/server/logic/gameController"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/gatewayPlatform"
	"github.com/drop/GoServer/server/service/logger"
)

func main() {
	gatewayPlatform.BootGatewayPlatform()
	platform.StartPprofByEnv()
	gameController.InitGatewayController(gatewayPlatform.GetRouter(), gatewayPlatform.GetSessionManager(), gatewayPlatform.GetDispatcher(), gatewayPlatform.GetMessageSender(), gatewayPlatform.GetUnlockService(), gatewayPlatform.GetActivityService())
	gameController.RegisterGatewayMessage()
	gameController.RegisterAllMessage()
	gameController.LoadRecentPlayerBasicInfoFromDB()

	platform.ListenSignal(&GatewaySignalHooker{})

	logger.InfoWithSprintf("Server Start!!!!!")
	select {}
}

type GatewaySignalHooker struct {
}

var _ logicCommon.SignalHooker = (*GatewaySignalHooker)(nil)

func (g GatewaySignalHooker) KickAllPlayer() {
	gatewayPlatform.KickOutAllPlayer()
}

func (g GatewaySignalHooker) AfterAllConfigReload() {
	gatewayPlatform.GetActivityService().Reload()
}
