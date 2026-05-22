// File: gmHandler.go
// Description: 背包系统GM命令处理器
// Author: 木村凉太
// Create Time: 2025.11

package inventory

import (
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*InventoryGMHandler)(nil)

// InventoryGMHandler 背包系统GM命令处理器
type InventoryGMHandler struct {
	itemService    logicCommon.ItemService
	sessionManager logicCommon.SessionManagerInterface
}

// NewInventoryGMHandler 创建背包系统GM命令处理器
func NewInventoryGMHandler(manager logicCommon.SessionManagerInterface, itemService logicCommon.ItemService) *InventoryGMHandler {
	return &InventoryGMHandler{
		sessionManager: manager,
		itemService:    itemService,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *InventoryGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_ADD_ITEM,
		pb.GMCommandType_GM_CMD_REMOVE_ITEM,
		pb.GMCommandType_GM_CMD_ADD_CURRENCY,
		pb.GMCommandType_GM_CMD_REMOVE_CURRENCY,
	}
}

// HandleCommand 处理GM命令
func (h *InventoryGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_ADD_ITEM:
		return h.handleAddItem(req, userId)
	case pb.GMCommandType_GM_CMD_REMOVE_ITEM:
		return h.handleRemoveItem(req, userId)
	case pb.GMCommandType_GM_CMD_ADD_CURRENCY:
		return h.handleAddCurrency(req, userId)
	case pb.GMCommandType_GM_CMD_REMOVE_CURRENCY:
		return h.handleRemoveCurrency(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("背包系统不支持的命令类型: %d", cmdType),
		}
	}
}

// handleAddItem 处理添加道具命令
func (h *InventoryGMHandler) handleAddItem(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	itemId := req.GetItemId()
	quantity := req.GetItemQuantity()
	invType := req.GetInventoryType()

	if itemId <= 0 || quantity <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "道具ID和数量必须大于0",
		}
	}

	if invType <= 0 {
		invType = 1 // 默认主背包
	}

	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	// 货币通常也是作为道具存储在背包中
	err := h.itemService.AddItem(player, &gameConfig.ItemConfig{ID: int32(itemId), Num: int64(quantity)}, enum.ITEM_CHANGE_REASON_GM)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("添加道具失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: "添加道具成功",
		ExtraData: map[string]string{
			"item_id":  fmt.Sprintf("%d", itemId),
			"quantity": fmt.Sprintf("%d", quantity),
			"user_id":  fmt.Sprintf("%d", userId),
		},
	}
}

// handleRemoveItem 处理移除道具命令
func (h *InventoryGMHandler) handleRemoveItem(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	itemId := req.GetItemId()
	quantity := req.GetItemQuantity()
	invType := req.GetInventoryType()

	if itemId <= 0 || quantity <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "道具ID和数量必须大于0",
		}
	}

	if invType <= 0 {
		invType = 1 // 默认主背包
	}
	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	err := h.itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: int32(itemId), Num: int64(quantity)}, enum.ITEM_CHANGE_REASON_GM)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("移除道具失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: "移除道具成功",
		ExtraData: map[string]string{
			"item_id":  fmt.Sprintf("%d", itemId),
			"quantity": fmt.Sprintf("%d", quantity),
			"user_id":  fmt.Sprintf("%d", userId),
		},
	}
}

// handleAddCurrency 处理增加货币命令
func (h *InventoryGMHandler) handleAddCurrency(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	currencyType := req.GetCurrencyType()
	currencyAmount := req.GetCurrencyAmount()

	if currencyType <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "货币类型必须大于0",
		}
	}

	if currencyAmount <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "货币数量必须大于0",
		}
	}

	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	// 货币通常也是作为道具存储在背包中
	err := h.itemService.AddItem(player, &gameConfig.ItemConfig{ID: currencyType, Num: currencyAmount}, enum.ITEM_CHANGE_REASON_GM)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("增加货币失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("增加货币成功: Type=%d, Amount=%d", currencyType, currencyAmount),
		ExtraData: map[string]string{
			"currency_type":   fmt.Sprintf("%d", currencyType),
			"currency_amount": fmt.Sprintf("%d", currencyAmount),
			"user_id":         fmt.Sprintf("%d", userId),
		},
	}
}

// handleRemoveCurrency 处理移除货币命令
func (h *InventoryGMHandler) handleRemoveCurrency(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	currencyType := req.GetCurrencyType()
	currencyAmount := req.GetCurrencyAmount()

	if currencyType <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "货币类型必须大于0",
		}
	}

	if currencyAmount <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "货币数量必须大于0",
		}
	}

	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	err := h.itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: currencyType, Num: currencyAmount}, enum.ITEM_CHANGE_REASON_GM)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("移除货币失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("移除货币成功: Type=%d, Amount=%d", currencyType, currencyAmount),
		ExtraData: map[string]string{
			"currency_type":   fmt.Sprintf("%d", currencyType),
			"currency_amount": fmt.Sprintf("%d", currencyAmount),
			"user_id":         fmt.Sprintf("%d", userId),
		},
	}
}
