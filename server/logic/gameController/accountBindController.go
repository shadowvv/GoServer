package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("accountBind", &AccountBindController{})
}

type AccountBindController struct{}

var _ LogicControllerInterface = (*AccountBindController)(nil)

// RegisterLogicMessage 注册账号绑定与领奖路由
func (a *AccountBindController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ACCOUNT_BIND_REQ, &pb.AccountBindReq{}, AccountBindHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ACCOUNT_BIND_CLAIM_REQ, &pb.AccountBindClaimReq{}, AccountBindClaimHandle, enum.FUNCTION_ID_NONE)
}

// AccountBindHandle 账号绑定渠道：无记录则插入 0 再 0→1（已绑定），已有 1/2 则返回 success
func AccountBindHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AccountBindReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	channel := req.GetChannel()
	if channel == "" {
		messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_RESP, &pb.AccountBindResp{Success: false})
		return
	}
	if err := model.BindChannelIfNotExists(player.GetUserId(), channel); err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_RESP, &pb.AccountBindResp{Success: false})
		return
	}
	bound, _ := model.TryMarkAsBound(player.GetUserId(), channel)
	if bound {
		messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_RESP, &pb.AccountBindResp{Success: true})
		return
	}
	// 已存在且状态为 1 或 2，也算绑定成功
	ent, _ := model.GetChannelBind(player.GetUserId(), channel)
	success := ent != nil && ent.ClaimStatus >= 1
	messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_RESP, &pb.AccountBindResp{Success: success})
}

// AccountBindClaimHandle 账号绑定领奖：校验绑定与领取状态，具体奖励字段后续扩展
func AccountBindClaimHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AccountBindClaimReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_CLAIM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	channel := req.GetChannel()
	if channel == "" {
		messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_CLAIM_RESP, &pb.AccountBindClaimResp{Success: false})
		return
	}
	claimed, err := model.TryMarkChannelClaimed(player.GetUserId(), channel)
	if err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_CLAIM_RESP, &pb.AccountBindClaimResp{Success: false})
		return
	}
	// 发奖
	if claimed {
		item := &gameConfig.ItemConfig{
			ID:  gameConfig.GetBindingBonus().ID,
			Num: gameConfig.GetBindingBonus().Num,
		}
		_ = itemService.AddItem(player, item, enum.ITEM_CHANGE_REASON_BINDING_BONUS)
	}

	// claimed == true：本次请求从“未领取”成功改为“已领取”，允许发放奖励
	// claimed == false：未绑定或已领取过，都视为失败（不重复领奖）
	messageSender.SendMessage(player, pb.MESSAGE_ID_ACCOUNT_BIND_CLAIM_RESP, &pb.AccountBindClaimResp{Success: claimed})
}
