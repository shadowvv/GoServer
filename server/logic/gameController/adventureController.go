package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/adventure"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/raid"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("adventure", &AdventureController{})
}

type AdventureController struct{}

var _ LogicControllerInterface = (*AdventureController)(nil)

func (c *AdventureController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ADVENTURE_INFO_REQ, &pb.AdventureInfoReq{}, AdventureInfoHandle, enum.FUNCTION_ID_MYSTICREALM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ADVENTURE_START_REQ, &pb.AdventureStartReq{}, AdventureStartHandle, enum.FUNCTION_ID_MYSTICREALM)
}

func AdventureInfoHandle(message proto.Message, player *model.PlayerModel) {
	messageSender.SendMessage(player, pb.MESSAGE_ID_ADVENTURE_INFO_RESP, adventure.BuildAdventureInfoResp(player))
}

func AdventureStartHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AdventureStartReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerInstanceModel == nil || player.PlayerInstanceModel.CurrentRaidInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	raidData, errorCode := adventure.StartAdventure(player, req.UniqueId)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, errorCode)
		return
	}

	raid.OnLeaveRaid(player.PlayerInstanceModel.CurrentRaidInfo)

	err := raid.BuildInstanceRaid(raidData)
	if err != nil {
		adventure.CancelStartAdventure(player, req.UniqueId)
		platformLogger.ErrorWithUser("enter adventure instance error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if !loginMutexService.EnterMutex(player.GetUserAccount(), player.GetUserId()) {
		adventure.CancelStartAdventure(player, req.UniqueId)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, pb.ERROR_CODE_ENTER_SCENE_REPEAT)
		return
	}
	err = raid.EnterScene(player, raidData)
	loginMutexService.ExitMutex(player.GetUserAccount(), player.GetUserId())

	if err != nil {
		adventure.CancelStartAdventure(player, req.UniqueId)
		platformLogger.ErrorWithUser("enter scene error", player, err)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_ADVENTURE_START_RESP, &pb.AdventureStartResp{
		Info: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	operationLogService.OnUserAdventureInstanceEnter(player.GetUserId(), raidData.CurrentStageId)
	eventBusService.SubmitJoinInstanceEvent(player.GetUserId(), player.GetUserServerId(), raidData.InstanceID, raidData.CurrentStageId)
}
