package main

import (
	"github.com/drop/GoServer/server/logic/gameController"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/socialPlatform"
	"github.com/drop/GoServer/server/logic/socialService"
	"github.com/drop/GoServer/server/service/logger"
)

func main() {
	socialPlatform.BootSocialPlatform()
	platform.StartPprofByEnv()
	gameController.InitSocialController(socialPlatform.GetRouter(), socialPlatform.GetSessionManager(), socialPlatform.GetDispatcher(), socialPlatform.GetMessageSender(), socialPlatform.GetUnlockService(), socialPlatform.GetActivityService())
	gameController.RegisterAllianceMessage()
	socialService.InitService()

	platform.ListenSignal(&socialSignalHooker{})

	logger.InfoWithSprintf("Server Start!!!!!")
	select {}
}

type socialSignalHooker struct {
}

var _ logicCommon.SignalHooker = (*socialSignalHooker)(nil)

func (s socialSignalHooker) KickAllPlayer() {
}

func (s socialSignalHooker) AfterAllConfigReload() {
}
