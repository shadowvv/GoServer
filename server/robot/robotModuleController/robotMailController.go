package robotModuleController

import (
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterRobotController("mail", &RobotMailController{})
}

type RobotMailController struct{}

var _ robotRouter.RobotControllerInterface = (*RobotMailController)(nil)

func (c *RobotMailController) RegisterRobotMessages() {
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_LIST_REQ, &pb.MailListReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_DETAIL_REQ, &pb.MailDetailReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_READ_REQ, &pb.MailReadReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_CLAIM_REQ, &pb.MailClaimReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_CLAIM_ALL_REQ, &pb.MailClaimAllReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_DELETE_REQ, &pb.MailDeleteReq{})
	RegisterRobotMessageHandler("mail", pb.MESSAGE_ID_MAIL_DELETE_CLAIMED_REQ, &pb.MailDeleteClaimedReq{})

	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_LIST_RESP, &pb.MailListResp{}, onMailListResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_DETAIL_RESP, &pb.MailDetailResp{}, onMailDetailResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_READ_RESP, &pb.MailReadResp{}, onMailReadResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_CLAIM_RESP, &pb.MailClaimResp{}, onMailClaimResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_CLAIM_ALL_RESP, &pb.MailClaimAllResp{}, onMailClaimAllResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_DELETE_RESP, &pb.MailDeleteResp{}, onMailDeleteResp)
	RegisterRobotReceiveMessageHandler("mail", pb.MESSAGE_ID_MAIL_DELETE_CLAIMED_RESP, &pb.MailDeleteClaimedResp{}, onMailDeleteClaimedResp)
}

func (c *RobotMailController) GetManualProto(msgID pb.MESSAGE_ID) proto.Message {
	return nil
}

func onMailListResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailListResp)
	_ = robot
	return ok
}

func onMailDetailResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailDetailResp)
	_ = robot
	return ok
}

func onMailReadResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailReadResp)
	_ = robot
	return ok
}

func onMailClaimResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailClaimResp)
	_ = robot
	return ok
}

func onMailClaimAllResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailClaimAllResp)
	_ = robot
	return ok
}

func onMailDeleteResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailDeleteResp)
	_ = robot
	return ok
}

func onMailDeleteClaimedResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.MailDeleteClaimedResp)
	_ = robot
	return ok
}
