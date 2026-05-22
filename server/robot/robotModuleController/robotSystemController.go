package robotModuleController

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/robot/robotLogger"
	"github.com/drop/GoServer/server/robot/robotLogic"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterRobotController("system", &RobotSystemController{})
}

type RobotSystemController struct{}

var _ robotRouter.RobotControllerInterface = (*RobotSystemController)(nil)

func (c *RobotSystemController) RegisterRobotMessages() {
	RegisterRobotReceiveMessageHandler("system", pb.MESSAGE_ID_LOGIN_RESP, &pb.LoginResp{}, onLoginResp)
	RegisterRobotReceiveMessageHandler("system", pb.MESSAGE_ID_LOAD_SCENE_OVER_RESP, &pb.LoadSceneOverResp{}, onLoadSceneOverResp)
	RegisterRobotReceiveMessageHandler("system", pb.MESSAGE_ID_MESSAGE_ERROR, &pb.MessageError{}, onMessageError)
	RegisterRobotReceiveMessageHandler("system", pb.MESSAGE_ID_HEART_RESP, &pb.HeartResp{}, onHeartResp)
}

func onMessageError(message proto.Message, robot robotCommon.RobotInterface) bool {
	msgErr, ok := message.(*pb.MessageError)
	if !ok {
		return false
	}

	realRobot, ok := robot.(*robotLogic.Robot)
	if !ok {
		return false
	}

	robotLogger.ErrorWithRobot(realRobot, fmt.Sprintf("phase=server_message_error respMsgID=%d errorCode=%d", msgErr.GetMsgId(), msgErr.GetErrorCode()))
	realRobot.HandleWaitingResponseByRespID(uint32(msgErr.GetMsgId()))
	return true
}

func (c *RobotSystemController) GetManualProto(msgID pb.MESSAGE_ID) proto.Message {
	return nil
}

func onLoginResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	if _, ok := message.(*pb.LoginResp); !ok {
		return false
	}
	realRobot, ok := robot.(*robotLogic.Robot)
	if !ok {
		return false
	}
	robotLogger.InfoWithRobot(realRobot, fmt.Sprintf("phase=login_resp status=success costMs=%d action=send_load_scene_over", realRobot.CostSinceLoginRequest().Milliseconds()))
	realRobot.SendLoadSceneOverRequest()
	return true
}

func onLoadSceneOverResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	if _, ok := message.(*pb.LoadSceneOverResp); !ok {
		return false
	}
	realRobot, ok := robot.(*robotLogic.Robot)
	if !ok {
		return false
	}
	realRobot.SetSceneLoaded(true)
	realRobot.SetAuthed(true)
	robotLogger.InfoWithRobot(realRobot, fmt.Sprintf("phase=ready status=success loadSceneCostMs=%d", realRobot.CostSinceLoadSceneRequest().Milliseconds()))
	return true
}

func onHeartResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.HeartResp)
	return ok
}
