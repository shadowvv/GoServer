// File: vipCardController.go
// Description: 特权卡系统控制器
// Author: 木村凉太
// Create Time: 2026.02

package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/vipCard"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("vipCard", &VipCardController{})
}

type VipCardController struct {
}

var _ LogicControllerInterface = (*VipCardController)(nil)

func (v *VipCardController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CLAIM_PRIVILEGE_REWARD_REQ, &pb.ClaimPrivilegeRewardReq{}, ClaimPrivilegeRewardHandle, enum.FUNCTION_ID_NONE)
}

// ClaimPrivilegeRewardHandle 处理领取特权奖励请求
func ClaimPrivilegeRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ClaimPrivilegeRewardReq)
	if !ok {

		return
	}

	// 领取奖励
	items, err := vipCard.Service.ClaimPrivilegeReward(player, req.RewardType)
	if err != nil {
		platformLogger.ErrorWithUser("ClaimPrivilegeReward failed", player, err)
		return
	}

	// 发放奖励
	if len(items) > 0 {
		err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_VIP_PRIVILEGE_REWARD)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return
		}
	}

	// 转换为PB格式
	rewards := make([]*pb.ItemBasicInfo, 0, len(items))
	for _, item := range items {
		if item != nil && item.ID > 0 {
			rewards = append(rewards, &pb.ItemBasicInfo{
				ItemId: item.ID,
				Count:  item.Num,
			})
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CLAIM_PRIVILEGE_REWARD_RESP, &pb.ClaimPrivilegeRewardResp{
		Rewards: rewards,
	})
}
