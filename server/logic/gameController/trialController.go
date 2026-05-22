package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/trial"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("Trial", &TrialController{})
}

type TrialController struct{}

var _ LogicControllerInterface = (*TrialController)(nil)

func (c *TrialController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TRIAL_INFO_REQ, &pb.TrialInfoReq{}, TrialInfoHandle, enum.FUNCTION_ID_TRIAL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TRIAL_TASK_REWARD_REQ, &pb.TrialTaskRewardReq{}, TrialTaskRewardHandle, enum.FUNCTION_ID_TRIAL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TRIAL_REWARD_REQ, &pb.TrialRewardReq{}, TrialRewardHandle, enum.FUNCTION_ID_TRIAL)
}

func TrialInfoHandle(message proto.Message, player *model.PlayerModel) {
	if trial.Service == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	req, ok := message.(*pb.TrialInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	resp, err := trial.Service.GetTrialInfo(player, req.ActId)
	if err != nil {
		platformLogger.ErrorWithUser("get trial info error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_TRIAL_INFO_RESP, resp)
}

func TrialTaskRewardHandle(message proto.Message, player *model.PlayerModel) {
	if trial.Service == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_TASK_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	req, ok := message.(*pb.TrialTaskRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_TASK_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	pushUpdate, err := trial.Service.ClaimTaskReward(player, req.ActId, req.TrialTaskId)
	if err != nil {
		platformLogger.ErrorWithUser("claim trial task reward error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_TASK_REWARD_RESP, getTrialErrorCode(err))
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_TRIAL_TASK_REWARD_RESP, &pb.TrialTaskRewardResp{
		IsSuccess: true,
	})
	if pushUpdate != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, pushUpdate)
	}
}

func TrialRewardHandle(message proto.Message, player *model.PlayerModel) {
	if trial.Service == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	req, ok := message.(*pb.TrialRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	err := trial.Service.ClaimProgressReward(player, req.ActId)
	if err != nil {
		platformLogger.ErrorWithUser("claim trial progress reward error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TRIAL_REWARD_RESP, getTrialErrorCode(err))
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_TRIAL_REWARD_RESP, &pb.TrialRewardResp{
		IsSuccess: true,
	})
}

func getTrialErrorCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	switch err.Error() {
	case "activity not open":
		return pb.ERROR_CODE_SYSTEM_ERROR
	case "trial task config not found", "trial task not in activity", "reward config not found", "task core is nil":
		return pb.ERROR_CODE_CFG_NOT_FOUND
	case "task not found":
		return pb.ERROR_CODE_TASK_NOT_FOUND
	case "task not finished", "task already rewarded", "day not unlocked", "no reward available":
		return pb.ERROR_CODE_TASK_NOT_FINISH
	default:
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
}
