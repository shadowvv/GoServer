package main

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/platform"
)

func main() {
	platform.InitPlatform(enum.SERVER_TYPE_GAME, enum.ENV_DEVELOP)
}
