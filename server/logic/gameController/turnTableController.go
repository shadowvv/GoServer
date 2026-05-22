package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/turnTable"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("turnTable", &TurnTableController{})
}

type TurnTableController struct {
}

var _ LogicControllerInterface = (*TurnTableController)(nil)

func (c *TurnTableController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TURN_TABLE_DETAIL_REQ, &pb.TurnTableDetailReq{}, TurnTableDetailHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TURN_TABLE_DRAW_REQ, &pb.TurnTableDrawReq{}, TurnTableDrawHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TURN_TABLE_USUALLY_REWARD_REQ, &pb.TurnTableUsuallyRewardReq{}, TurnTableUsuallyRewardHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_TURN_TABLE_TASK_REWARD_REQ, &pb.TurnTableTaskRewardReq{}, TurnTableTaskRewardHandle, enum.FUNCTION_ID_NONE)
}

func TurnTableDetailHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.TurnTableDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TURN_TABLE_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	info, err := turnTable.Service.GetDetail(player, req.ModId)
	if err != nil {
		sendTurnTableError(player, pb.MESSAGE_ID_TURN_TABLE_DETAIL_RESP, err)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TURN_TABLE_DETAIL_RESP, &pb.TurnTableDetailResp{Info: info})
}

func TurnTableDrawHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.TurnTableDrawReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TURN_TABLE_DRAW_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	drawCountMap, err := turnTable.Service.Draw(player, req.ModId, req.Count)
	if err != nil {
		sendTurnTableError(player, pb.MESSAGE_ID_TURN_TABLE_DRAW_RESP, err)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TURN_TABLE_DRAW_RESP, &pb.TurnTableDrawResp{IsSuccess: true, DrawCountMap: drawCountMap})
}

func TurnTableUsuallyRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.TurnTableUsuallyRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TURN_TABLE_USUALLY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := turnTable.Service.ClaimUsuallyReward(player, req.ModId, req.Type, req.RewardId); err != nil {
		sendTurnTableError(player, pb.MESSAGE_ID_TURN_TABLE_USUALLY_REWARD_RESP, err)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TURN_TABLE_USUALLY_REWARD_RESP, &pb.TurnTableUsuallyRewardResp{IsSuccess: true})
}

func TurnTableTaskRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.TurnTableTaskRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_TURN_TABLE_TASK_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	push, err := turnTable.Service.ClaimTaskReward(player, req.ActTaskId)
	if err != nil {
		sendTurnTableError(player, pb.MESSAGE_ID_TURN_TABLE_TASK_REWARD_RESP, err)
		return
	}
	if push != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_TURN_TABLE_TASK_REWARD_RESP, &pb.TurnTableTaskRewardResp{IsSuccess: true})
}

func sendTurnTableError(player *model.PlayerModel, msgId pb.MESSAGE_ID, err error) {
	switch err.Error() {
	case "turn table count error":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ERROR_CODE_TURN_TABLE_COUNT_ERROR)
	case "reward not reach":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ERROR_CODE_TURN_TABLE_REWARD_NOT_REACH)
	case "reward already claimed", "task already rewarded":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ERROR_CODE_TURN_TABLE_REWARD_ALREADY_CLAIMED)
	case "reward order error":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ERROR_CODE_TURN_TABLE_REWARD_ORDER_ERROR)
	case "pool empty":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ERROR_CODE_TURN_TABLE_POOL_EMPTY)
	case "activity not open", "activity settled":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
	case "item not enough":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
	case "cfg not found":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_CFG_NOT_FOUND)
	case "task not finish":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_TASK_NOT_FINISH)
	case "invalid request", "task not found":
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
	default:
		messageSender.SendErrorMessage(player, msgId, pb.ERROR_CODE_SYSTEM_ERROR)
	}
}
