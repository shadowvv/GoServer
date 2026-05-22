package robotModuleController

import (
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"google.golang.org/protobuf/proto"
)

func RegisterRobotController(module string, controller robotRouter.RobotControllerInterface) {
	robotRouter.RegisterRobotController(module, controller)
}

func RegisterRobotReceiveMessageHandler(module string, messageID pb.MESSAGE_ID, response proto.Message, handler robotRouter.RobotReceiveMessageHandler) {
	robotRouter.RegisterRobotReceiveMessageHandler(module, messageID, response, handler)
}

func RegisterRobotMessageHandler(module string, messageID pb.MESSAGE_ID, request proto.Message) {
	robotRouter.RegisterRobotMessageHandler(module, messageID, request)
}
