package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("CityFurniture", &FurnitureController{})
}

// FurnitureController 家具控制器
type FurnitureController struct{}

var _ LogicControllerInterface = (*FurnitureController)(nil)

func (c *FurnitureController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_FURNITURE_LEVEL_UP_REQ, &pb.CityFurnitureLevelUpReq{}, CityFurnitureLevelUpHandle, enum.FUNCTION_ID_NONE)
}

// CityFurnitureLevelUpHandle 家具升级
func CityFurnitureLevelUpHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("city furniture level up Req", player)

	req, ok := message.(*pb.CityFurnitureLevelUpReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_FURNITURE_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	err := furnitureService.LevelUp(player, req.ArchitectureType, req.FurnitureType)
	if err != nil {
		platformLogger.ErrorWithUser("furniture level up error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_FURNITURE_LEVEL_UP_RESP, getCityLumberErrorCode(err))
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_FURNITURE_LEVEL_UP_RESP, &pb.CityFurnitureLevelUpResp{
		IsSuccess: true,
	})
}
