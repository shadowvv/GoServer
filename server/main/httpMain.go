package main

import (
	"github.com/drop/GoServer/server/logic/backend"
	"github.com/drop/GoServer/server/logic/gameController"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/httpPlatform"
	"github.com/drop/GoServer/server/service/logger"
)

func main() {
	httpPlatform.BootHttpPlatform()
	platform.StartPprofByEnv()
	gameController.RegisterWebMessage()
	gameController.RegisterWebGmMessage()
	gameController.RegisterRechargeMessage()
	httpPlatform.StartHttpService()

	backend.InitBackend()

	platform.ListenSignal(&httpSignalHandler{})

	logger.InfoWithSprintf("Server Start!!!!!")
	select {}
}

type httpSignalHandler struct {
}

var _ logicCommon.SignalHooker = (*httpSignalHandler)(nil)

func (g httpSignalHandler) KickAllPlayer() {

}

func (h httpSignalHandler) AfterAllConfigReload() {

}
