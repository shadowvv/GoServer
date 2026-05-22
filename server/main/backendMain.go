package main

import (
	"github.com/drop/GoServer/server/logic/backend"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/backendPlatform"
)

func main() {
	backendPlatform.BootBackEndPlatform()
	platform.StartPprofByEnv()
	backend.InitBackend()
	backend.RegisterBackendMessage()
	backendPlatform.StartHttpService()
	select {}
}
