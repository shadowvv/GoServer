package main

import (
	"flag"
	"fmt"

	"github.com/drop/GoServer/server/robot/robotApi"
)

func main() {
	addr := flag.String("addr", ":18081", "robot api listen address")
	flag.Parse()

	server, err := robotApi.NewServer(*addr)
	if err != nil {
		panic(fmt.Sprintf("robot api start failed: %v", err))
	}
	if err = server.Start(); err != nil {
		panic(fmt.Sprintf("robot api listen failed: %v", err))
	}
}
