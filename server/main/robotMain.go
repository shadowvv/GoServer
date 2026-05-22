package main

import (
	"fmt"

	"github.com/drop/GoServer/server/robot"
)

func main() {
	if err := robot.Boot(); err != nil {
		panic(fmt.Sprintf("program start failed: %v", err))
	}
}
