package main

import (
	"github.com/drop/GoServer/server/logic"
	"github.com/drop/GoServer/server/logic/login"
	"github.com/drop/GoServer/server/logic/platform"
)

func main() {
	platform.BootPlatform()

	logic.RegisterProcessor()

	login.RegisterControllerMessage()

	select {}
}
