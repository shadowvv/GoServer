package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("appearance", &AppearanceController{})
}

type AppearanceController struct {
}

var _ LogicControllerInterface = (*AppearanceController)(nil)

func (a *AppearanceController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_AVATAR_DETAILS_REQ, &pb.GetAvatarDetailsReq{}, GetAvatarDetailsHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_WEAR_AVATAR_REQ, &pb.WearAvatarReq{}, WearAvatarHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_UNFIX_AVATAR_REQ, &pb.UnfixAvatarReq{}, UnfixAvatarHandler, enum.FUNCTION_ID_NONE)
}

func GetAvatarDetailsHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetAvatarDetailsReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_AVATAR_DETAILS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	res := make([]*pb.AvatarDetail, 0)
	for _, v := range player.AppearanceModel.AppearanceEntities {
		res = append(res, &pb.AvatarDetail{
			Id:      v.AppearanceId,
			EndTime: v.EndTime,
			IsWear:  v.IsWear,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_AVATAR_DETAILS_RESP, &pb.GetAvatarDetailsResp{
		AvatarList: res,
	})
}

func WearAvatarHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.WearAvatarReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_WEAR_AVATAR_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	err := player.AppearanceModel.WearAppearance(req.Id)
	switch {
	case err == nil:
		messageSender.SendMessage(player, pb.MESSAGE_ID_WEAR_AVATAR_RESP, &pb.WearAvatarResp{})
	case err.Error() == "appearance entity is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_WEAR_AVATAR_RESP, pb.ERROR_CODE_APPEARANCE_ENTITY_NOT_EXIST)
	case err.Error() == "appearance cfg is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_WEAR_AVATAR_RESP, pb.ERROR_CODE_APPEARANCE_CFG_NOT_EXIST)
	default:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, &pb.CollectionEntryLevelUpResp{})
	}
}

func UnfixAvatarHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.UnfixAvatarReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_UNFIX_AVATAR_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	err := player.AppearanceModel.UnfixAppearance(req.Id)
	switch {
	case err == nil:
		messageSender.SendMessage(player, pb.MESSAGE_ID_UNFIX_AVATAR_RESP, &pb.UnfixAvatarResp{})
	case err.Error() == "appearance entity is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_UNFIX_AVATAR_RESP, pb.ERROR_CODE_APPEARANCE_ENTITY_NOT_EXIST)
	case err.Error() == "appearance cfg is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_UNFIX_AVATAR_RESP, pb.ERROR_CODE_APPEARANCE_CFG_NOT_EXIST)
	default:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, &pb.CollectionEntryLevelUpResp{})
	}
}
