package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("sign", &SignController{})
}

type SignController struct{}

var _ LogicControllerInterface = (*SignController)(nil)

func (s *SignController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_SIGN_INFO_REQ, &pb.SignInfoReq{}, SignInfoHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_SIGN_CLAIM_REQ, &pb.SignClaimReq{}, SignClaimHandle, enum.FUNCTION_ID_NONE)
}

func SignInfoHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.SignInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player.PlayerSignModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	nowMilli := tool.UnixNowMilli()
	signIDs := player.PlayerSignModel.SyncAndGetVisibleSigns(req.GetActIds(), nowMilli)

	resp := &pb.SignInfoResp{SignMap: make(map[int32]*pb.SignInfo)}
	for _, signID := range signIDs {
		cfg := gameConfig.GetDaySignCfg(signID)
		if cfg == nil {
			continue
		}
		e := player.PlayerSignModel.GetOrCreateSign(signID)
		if e == nil {
			continue
		}
		resp.SignMap[signID] = &pb.SignInfo{
			SignId:    signID,
			ActId:     cfg.ActID,
			SignedDay: e.SignedDay,
			ClaimedId: e.ClaimedIndex,
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_SIGN_INFO_RESP, resp)
}

func SignClaimHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.SignClaimReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player.PlayerSignModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	signID := req.GetSignId()
	cfg := gameConfig.GetDaySignCfg(signID)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SIGN_NOT_FOUND)
		return
	}
	open, _ := player.PlayerActivityModel.CheckActivityOpen(cfg.ActID)
	if !open {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SIGN_CLAIM_NOT_UNLOCKED)
		return
	}

	e := player.PlayerSignModel.GetOrCreateSign(signID)
	claimID := e.SignedDay
	rewardDayCount := int32(len(cfg.DropID))
	if claimID > rewardDayCount {
		claimID = rewardDayCount
	}
	if claimID <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SIGN_CLAIM_NOT_UNLOCKED)
		return
	}
	if claimID <= e.ClaimedIndex {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SIGN_ALREADY_CLAIMED)
		return
	}

	claimFrom := e.ClaimedIndex + 1
	items, err := model.BuildSignClaimItems(signID, claimFrom, claimID)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_SIGN); err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}

	player.PlayerSignModel.MergeClaimed(signID, claimID)

	rewards := make([]*pb.ItemBasicInfo, 0, len(items))
	for _, it := range items {
		if it == nil || it.ID <= 0 || it.Num <= 0 {
			continue
		}
		rewards = append(rewards, &pb.ItemBasicInfo{
			ItemId: it.ID,
			Count:  it.Num,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_SIGN_CLAIM_RESP, &pb.SignClaimResp{Rewards: rewards})
}
