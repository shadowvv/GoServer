package main

import (
	"GoServer/server/logic/enum"
	"GoServer/server/logic/platform"
)

func main() {

	platform.Init(enum.ENV_DEV)

}
