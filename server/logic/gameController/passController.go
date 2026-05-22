// File: passController.go
// Description: 通行证系统控制器
// Author: 木村凉太
// Create Time: 2026.02

package gameController

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("pass", &PassController{})
}

type PassController struct {
}

var _ LogicControllerInterface = (*PassController)(nil)

func (p *PassController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_PASS_INFO_REQ, &pb.GetPassInfoReq{}, GetPassInfoHandle, enum.FUNCTION_ID_NONE)
	//RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_PASS_REWARD_OPTIONS_REQ, &pb.GetPassRewardOptionsReq{}, GetPassRewardOptionsHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BUY_PASS_PROGRESS_REQ, &pb.BuyPassProgressReq{}, BuyPassProgressHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_REQ, &pb.ClaimLoopRewardReq{}, ClaimLoopRewardHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CLAIM_ALL_PASS_REWARD_REQ, &pb.ClaimAllPassRewardReq{}, ClaimAllPassRewardHandle, enum.FUNCTION_ID_NONE)
}

// GetPassInfoHandle 处理获取通行证信息请求
func GetPassInfoHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetPassInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	var passInfoList []*pb.PassInfo

	if req.PassId > 0 {
		// 获取指定通行证信息
		passInfo, err := passService.GetPassInfo(player, req.PassId)
		if err != nil {
			platformLogger.ErrorWithUser("GetPassInfo failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		if passInfo != nil {
			passInfoList = append(passInfoList, passInfo)
		}
	} else {
		// 获取所有通行证信息
		allPassInfo, err := passService.GetAllPassInfo(player)
		if err != nil {
			platformLogger.ErrorWithUser("GetAllPassInfo failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		passInfoList = allPassInfo
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_PASS_INFO_RESP, &pb.GetPassInfoResp{
		PassList: passInfoList,
	})
}

// GetPassRewardOptionsHandle 处理获取通行证奖励选项请求
//func GetPassRewardOptionsHandle(message proto.Message, player *model.PlayerModel) {
//    req, ok := message.(*pb.GetPassRewardOptionsReq)
//    if !ok {
//
//        platform.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_REWARD_OPTIONS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
//        return
//    }
//
//    // 获取奖励选项
//    options, err := pass.Service.GetPassRewardOptions(player, req.PassId, req.Level, req.RewardLevel)
//    if err != nil {
//        platformLogger.ErrorWithUser("GetPassRewardOptions failed", player, err)
//        platform.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_REWARD_OPTIONS_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
//        return
//    }
//
//    platform.SendMessage(player, pb.MESSAGE_ID_GET_PASS_REWARD_OPTIONS_RESP, &pb.GetPassRewardOptionsResp{
//        Options: options,
//    })
//}

// BuyPassProgressHandle 处理购买通行证积分请求
func BuyPassProgressHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.BuyPassProgressReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// 获取需要扣除的道具信息
	costItems, err := passService.BuyPassProgress(player, req.PassId, req.Points)
	if err != nil {
		platformLogger.ErrorWithUser("BuyPassProgress failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	// 检查并扣除道具
	if len(costItems) > 0 {
		items := make([]*gameConfig.ItemConfig, 0, len(costItems))
		for _, costItem := range costItems {
			items = append(items, &gameConfig.ItemConfig{
				ID:  costItem.ItemId,
				Num: costItem.Count,
			})
		}
		// 检查道具数量
		hasItems, err := itemService.CheckItemsCount(player, items)
		if err != nil || !hasItems {
			platformLogger.ErrorWithUser("CheckItemsCount failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		// 扣除道具
		player.PassModel.SetReqPassId(req.PassId)
		err = itemService.RemoveItems(player, items, enum.ITEM_CHANGE_REASON_PASS_CARD)
		if err != nil {
			platformLogger.ErrorWithUser("RemoveItems failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}

	// 扣除道具成功后，添加通行证进度
	err = passService.AddPassProgress(player, req.PassId, req.Points)
	if err != nil {
		platformLogger.ErrorWithUser("AddPassProgress failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	// 获取更新后的通行证信息
	passInfo, err := passService.GetPassInfo(player, req.PassId)
	if err != nil {
		platformLogger.ErrorWithUser("GetPassInfo failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_PASS_PROGRESS_RESP, &pb.BuyPassProgressResp{
		Progress: passInfo.Progress,
		Level:    passInfo.Level,
	})
}

// ClaimLoopRewardHandle 处理领取循环奖励请求
func ClaimLoopRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ClaimLoopRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// 领取循环奖励
	rewards, err := passService.ClaimLoopReward(player, req.PassId)
	if err != nil {
		platformLogger.ErrorWithUser("ClaimLoopReward failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	// 发放奖励
	if len(rewards) > 0 {
		items := make([]*gameConfig.ItemConfig, 0, len(rewards))
		for _, reward := range rewards {
			items = append(items, &gameConfig.ItemConfig{
				ID:  reward.ItemId,
				Num: reward.Count,
			})
		}
		player.PassModel.SetReqPassId(req.PassId)
		err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_PASS_CARD)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}

	// 获取剩余循环积分（通过重新获取通行证信息）
	passInfo, err := passService.GetPassInfo(player, req.PassId)
	if err != nil {
		platformLogger.ErrorWithUser("GetPassInfo failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	// 获取剩余循环积分（通过重新获取通行证信息）
	passInfo, err2 := passService.GetPassInfo(player, req.PassId)
	loopProgress := int32(0)
	if err2 == nil && passInfo != nil {
		loopProgress = passInfo.LoopProgress
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CLAIM_LOOP_REWARD_RESP, &pb.ClaimLoopRewardResp{
		Rewards:      rewards,
		LoopProgress: loopProgress,
	})
}

// ClaimAllPassRewardHandle 处理一次性领取所有可领取奖励请求
func ClaimAllPassRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ClaimAllPassRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_ALL_PASS_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// 将 choices 转换为 map
	choicesMap := make(map[string]*pb.PassRewardChoice)
	for _, choice := range req.Choices {
		key := fmt.Sprintf("%d_%d", choice.Level, choice.RewardLevel)
		choicesMap[key] = choice
	}

	// 领取所有奖励
	rewards, maxLevel, err := passService.ClaimAllPassReward(player, req.PassId, choicesMap)
	if err != nil {
		platformLogger.ErrorWithUser("ClaimAllPassReward failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_ALL_PASS_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	// 发放奖励
	if len(rewards) > 0 {
		items := make([]*gameConfig.ItemConfig, 0, len(rewards))
		for _, reward := range rewards {
			items = append(items, &gameConfig.ItemConfig{
				ID:  reward.ItemId,
				Num: reward.Count,
			})
		}
		player.PassModel.SetReqPassId(req.PassId)
		err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_PASS_CARD)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CLAIM_ALL_PASS_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}
	passInfo, err := passService.GetPassInfo(player, req.PassId)

	messageSender.SendMessage(player, pb.MESSAGE_ID_CLAIM_ALL_PASS_REWARD_RESP, &pb.ClaimAllPassRewardResp{
		Rewards:         rewards,
		MaxClaimedLevel: maxLevel,
		Pass:            passInfo,
	})
}
