package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/cityAge"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("cityAge", &CityAgeController{})
}

type CityAgeController struct {
}

var _ LogicControllerInterface = (*CityAgeController)(nil)

func (c *CityAgeController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_AGE_DETAIL_REQ, &pb.CityAgeDetailReq{}, CityAgeDetailHandle, enum.FUNCTION_ID_CITYAGE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_REQ, &pb.CityAgeGroupRewardReq{}, CityAgeGroupRewardHandle, enum.FUNCTION_ID_CITYAGE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_AGE_UP_REQ, &pb.CityAgeUpReq{}, CityAgeUpHandle, enum.FUNCTION_ID_CITYAGE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_REQ, &pb.CityAgeDailyRewardReq{}, CityAgeDailyRewardHandle, enum.FUNCTION_ID_CITYAGE)
}

func CityAgeDetailHandle(message proto.Message, player *model.PlayerModel) {
	if _, ok := message.(*pb.CityAgeDetailReq); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	info, ok := cityAge.Service.GetDetail(player)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DETAIL_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_AGE_DETAIL_RESP, &pb.CityAgeDetailResp{
		Info: info,
	})
}

func CityAgeGroupRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CityAgeGroupRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	pushes, err := cityAge.Service.ClaimGroupReward(player, req.GroupIndex)
	if err != nil {
		switch err.Error() {
		case "group index error", "group reward not found", "group reward already claimed":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		case "city age cfg not found":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		case "task not finish":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_TASK_NOT_FINISH)
		case "system error":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		default:
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		}
		return
	}

	for _, push := range pushes {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_AGE_GROUP_REWARD_RESP, &pb.CityAgeGroupRewardResp{
		IsSuccess: true,
	})
}

func CityAgeUpHandle(message proto.Message, player *model.PlayerModel) {
	if _, ok := message.(*pb.CityAgeUpReq); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	info, pushes, err := cityAge.Service.Upgrade(player)
	if err != nil {
		switch err.Error() {
		case "city age cfg not found":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		case "city age max", "group reward not all claimed", "upgrade reward not found":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		case "system error":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		default:
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		}
		return
	}

	for _, push := range pushes {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_AGE_UP_RESP, &pb.CityAgeUpResp{
		IsSuccess: true,
		Info:      info,
	})
	eventBusService.SubmitCityAgeChangeEvent(player.GetUserId(), info.CityAgeId)
}

func CityAgeDailyRewardHandle(message proto.Message, player *model.PlayerModel) {
	if _, ok := message.(*pb.CityAgeDailyRewardReq); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	err := cityAge.Service.ClaimDailyReward(player)
	if err != nil {
		switch err.Error() {
		case "city age cfg not found":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		case "daily reward not found", "daily reward already claimed":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		case "system error":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		default:
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		}
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_AGE_DAILY_REWARD_RESP, &pb.CityAgeDailyRewardResp{
		IsSuccess: true,
	})
}
