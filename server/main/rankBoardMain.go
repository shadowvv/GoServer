package main

import (
	"github.com/drop/GoServer/server/logic/gameController"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/rankBoardPlatform"
	"github.com/drop/GoServer/server/logic/rankboardService"
	"github.com/drop/GoServer/server/service/logger"
)

func main() {
	rankBoardPlatform.BootRankBoardPlatform()
	platform.StartPprofByEnv()
	gameController.InitRankBoardController(rankBoardPlatform.GetRouter(), rankBoardPlatform.GetSessionManager(), rankBoardPlatform.GetDispatcher(), rankBoardPlatform.GetMessageSender(), rankBoardPlatform.GetUnlockService(), rankBoardPlatform.GetActivityService())
	gameController.RegisterRankBoardMessage()
	rankboardService.InitService()
	gloryArenaService.InitService()
	gloryArenaService.StartService()

	platform.ListenSignal(&rankBoardSignalHooker{})

	logger.InfoWithSprintf("Server Start!!!!!")
	select {}
}

type rankBoardSignalHooker struct {
}

var _ logicCommon.SignalHooker = (*rankBoardSignalHooker)(nil)

func (g rankBoardSignalHooker) KickAllPlayer() {

}

func (r rankBoardSignalHooker) AfterAllConfigReload() {

}
