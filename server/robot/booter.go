package robot

import (
	"github.com/drop/GoServer/server/robot/robotPlatform"
	"github.com/drop/GoServer/server/tool"
)

func Boot() error {
	startTime := tool.Now()
	platform, err := robotPlatform.NewRobotPlatform()
	if err != nil {
		return err
	}
	platform.StartAllRobots()

	platform.WaitForExitSignal()
	platform.StopAllRobots()
	platform.PrintFinalSummary(startTime)
	return nil
}
