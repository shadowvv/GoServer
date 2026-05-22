// File: gmInternal.go
// Description: GM系统内部调用接口
// Author: 木村凉太
// Create Time: 2026.02

package gm

import (
	"encoding/json"
	"fmt"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/pb"
)

var internalGMService logicCommon.GMServiceInterface

// InitInternalGMService 初始化内部GM服务
// 注意：需要传入已注册好处理器的GM服务实例
func InitInternalGMService(gmService logicCommon.GMServiceInterface) {
	if internalGMService == nil {
		internalGMService = gmService
	}
}

// ExecuteGMCommand 执行GM命令（内部调用）
// 这是供程序内部其他模块调用的GM接口
// 注意：需要先调用 InitInternalGMService 初始化
func ExecuteGMCommand(req *pb.MessageGmReq) (*pb.MessageGmResp, error) {
	if internalGMService == nil {
		// 如果未初始化，返回错误提示需要先初始化
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: "GM内部服务未初始化，请先调用 InitInternalGMService",
		}, nil
	}

	invokerType := pb.GMInvokerType_GM_INVOKER_INTERNAL
	invokerId := "internal"

	// 设置调用方类型和标识
	req.InvokerType = invokerType
	req.InvokerId = invokerId

	return internalGMService.ExecuteCommand(req, invokerType, invokerId)
}

// AddItem 添加道具（内部调用）
func AddItem(userId int64, itemId int32, quantity int32, inventoryType int32) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:       pb.GMCommandType_GM_CMD_ADD_ITEM,
		UserId:        userId,
		ItemId:        int64(itemId),
		ItemQuantity:  quantity,
		InventoryType: inventoryType,
	}
	return ExecuteGMCommand(req)
}

// RemoveItem 移除道具（内部调用）
func RemoveItem(userId int64, itemId int32, quantity int32, inventoryType int32) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:       pb.GMCommandType_GM_CMD_REMOVE_ITEM,
		UserId:        userId,
		ItemId:        int64(itemId),
		ItemQuantity:  quantity,
		InventoryType: inventoryType,
	}
	return ExecuteGMCommand(req)
}

// AddExp 增加经验（内部调用）
func AddExp(userId int64, exp int64) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType: pb.GMCommandType_GM_CMD_ADD_EXP,
		UserId:  userId,
		Exp:     exp,
	}
	return ExecuteGMCommand(req)
}

// SetLevel 设置等级（内部调用）
func SetLevel(userId int64, level int32) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType: pb.GMCommandType_GM_CMD_SET_LEVEL,
		UserId:  userId,
		Level:   level,
	}
	return ExecuteGMCommand(req)
}

// HeroLevelUp 英雄升级（内部调用）
func HeroLevelUp(userId int64, heroOwnId int64, heroLevel int32) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:   pb.GMCommandType_GM_CMD_HERO_LEVEL_UP,
		UserId:    userId,
		HeroOwnId: heroOwnId,
		HeroLevel: heroLevel,
	}
	return ExecuteGMCommand(req)
}

// HeroStarUp 英雄升星（内部调用）
func HeroStarUp(userId int64, heroOwnId int64, heroStar int32) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:   pb.GMCommandType_GM_CMD_HERO_STAR_UP,
		UserId:    userId,
		HeroOwnId: heroOwnId,
		HeroStar:  heroStar,
	}
	return ExecuteGMCommand(req)
}

// AddCurrency 增加货币（内部调用）
func AddCurrency(userId int64, currencyType int32, currencyAmount int64) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:        pb.GMCommandType_GM_CMD_ADD_CURRENCY,
		UserId:         userId,
		CurrencyType:   currencyType,
		CurrencyAmount: currencyAmount,
	}
	return ExecuteGMCommand(req)
}

// RemoveCurrency 移除货币（内部调用）
func RemoveCurrency(userId int64, currencyType int32, currencyAmount int64) (*pb.MessageGmResp, error) {
	req := &pb.MessageGmReq{
		CmdType:        pb.GMCommandType_GM_CMD_REMOVE_CURRENCY,
		UserId:         userId,
		CurrencyType:   currencyType,
		CurrencyAmount: currencyAmount,
	}
	return ExecuteGMCommand(req)
}

func SendMail(cmdType pb.GMCommandType, mtp int32, userId int64, title string, cfgId int32, senderAvatar string, content string, items []*mail.MailAttachmentItem, isConvenient bool, expireDays int32) (*pb.MessageGmResp, error) {
	var mailParams struct {
		Title        string                     `json:"title"`
		Content      string                     `json:"content"`
		MailType     int32                      `json:"mail_type"`
		SenderAvatar string                     `json:"sender_avatar"` // 发送者头像（可选）
		Items        []*mail.MailAttachmentItem `json:"items"`         // 附件物品条目（业务约定：只有一个附件，直接传 items）
		ExpireDays   int32                      `json:"expire_days"`
		IsConvenient *bool                      `json:"is_convenient"` // 是否可一键领取（可选，默认true）
	}
	mailParams.Title = title
	mailParams.Content = content
	mailParams.MailType = mtp
	mailParams.SenderAvatar = senderAvatar
	mailParams.Items = items
	mailParams.ExpireDays = expireDays
	mailParams.IsConvenient = &isConvenient

	extraParamsBytes, err := json.Marshal(mailParams)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("解析邮件参数失败: %v", err),
		}, err
	}
	req := &pb.MessageGmReq{
		CmdType:     cmdType,
		ExtraParams: string(extraParamsBytes),
	}
	if cmdType == pb.GMCommandType_GM_CMD_SEND_MAIL {
		req.UserId = userId
	}

	return ExecuteGMCommand(req)
}
