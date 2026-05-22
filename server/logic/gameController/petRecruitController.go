package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/petRecruit"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type PetRecruitController struct{}

func init() {
	RegisterController("petRecruit", &PetRecruitController{})
}

var _ LogicControllerInterface = (*PetRecruitController)(nil)

func (c *PetRecruitController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_RECRUIT_DETAIL_REQ, &pb.PetRecruitDetailReq{}, PetRecruitDetailHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_RECRUIT_REQ, &pb.PetRecruitReq{}, PetRecruitHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_RECRUIT_REFRESH_REQ, &pb.PetRecruitRefreshReq{}, PetRecruitRefreshHandle, enum.FUNCTION_ID_PET)
}

func PetRecruitDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet recruit detail req", player)
	_, ok := message.(*pb.PetRecruitDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	// 统一以服务器毫秒时间戳驱动自动刷新状态机，避免客户端时间不一致导致的面板错乱。
	nowMilli := tool.UnixNowMilli()
	info, code, _ := petRecruit.GetDetail(player, nowMilli)
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_DETAIL_RESP, code)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_RECRUIT_DETAIL_RESP, &pb.PetRecruitDetailResp{BasicInfo: info})
}

// PetRecruitHandle 宠物招募
func PetRecruitHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet recruit req", player)
	req, ok := message.(*pb.PetRecruitReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	// 招募与刷新同一套时间基准：招募前会补一次系统自动刷新同步。
	nowMilli := tool.UnixNowMilli()
	sanctuaryLevel := petRecruit.GetSanctuaryLevel(player)
	petRecruit.SetEventBusService(eventBusService)
	resp, code, _ := petRecruit.RecruitPetFromCandidates(player, req.GetRecruitType(), req.GetTripleRecruit(), sanctuaryLevel, nowMilli)
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_RESP, code)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_RECRUIT_RESP, resp)
}

func PetRecruitRefreshHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet recruit refresh req", player)
	_, ok := message.(*pb.PetRecruitRefreshReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_REFRESH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	// 手动刷新也走同一套“系统自动刷新同步”逻辑，保证无心跳时状态仍正确。
	nowMilli := tool.UnixNowMilli()
	sanctuaryLevel := petRecruit.GetSanctuaryLevel(player)
	info, code, _ := petRecruit.ManualRefresh(player, sanctuaryLevel, nowMilli)
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_RECRUIT_REFRESH_RESP, code)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_RECRUIT_REFRESH_RESP, &pb.PetRecruitRefreshResp{BasicInfo: info})
}
