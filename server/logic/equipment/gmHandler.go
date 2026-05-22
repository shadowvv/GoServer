// File: gmHandler.go
// Description: 装备系统GM命令处理器
// Author: 木村凉太
// Create Time: 2025.11

package equipment

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*EquipmentGMHandler)(nil)

// EquipmentGMHandler 装备系统GM命令处理器
type EquipmentGMHandler struct {
	equipmentService logicCommon.EquipmentInterface
}

// NewEquipmentGMHandler 创建装备系统GM命令处理器
func NewEquipmentGMHandler(equipmentService logicCommon.EquipmentInterface) *EquipmentGMHandler {
	return &EquipmentGMHandler{
		equipmentService: equipmentService,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *EquipmentGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_ADD_EQUIPMENT,
		pb.GMCommandType_GM_CMD_REMOVE_EQUIPMENT,
	}
}

// HandleCommand 处理GM命令
func (h *EquipmentGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_ADD_EQUIPMENT:
		return h.handleAddEquipment(req, userId)
	case pb.GMCommandType_GM_CMD_REMOVE_EQUIPMENT:
		return h.handleRemoveEquipment(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("装备系统不支持的命令类型: %d", cmdType),
		}
	}
}

// handleAddEquipment 处理添加装备命令
// 参数说明：
//   - item_id: 装备模板ID (equipment_id)
//   - level: 装备等级 (equipment_level)
//   - item_quantity: 装备数量（可选，默认为1）
func (h *EquipmentGMHandler) handleAddEquipment(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	equipmentID := req.GetItemId() // 使用item_id作为equipment_id
	level := req.GetLevel()        // 使用level作为equipment_level
	quantity := req.GetItemQuantity()

	if equipmentID <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "装备ID必须大于0",
		}
	}

	if level <= 0 {
		level = 1 // 默认等级1
	}

	if quantity <= 0 {
		quantity = 1 // 默认数量1
	}

	// 批量添加装备
	var equipmentOwnIDs []int64
	var failedCount int
	for i := int32(0); i < quantity; i++ {
		equipmentOwnID, err := h.equipmentService.AddEquipment(userId, int32(equipmentID), level)
		if err != nil {
			failedCount++
			continue
		}
		equipmentOwnIDs = append(equipmentOwnIDs, equipmentOwnID)
	}

	if failedCount > 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("添加装备部分失败: 成功%d件, 失败%d件", len(equipmentOwnIDs), failedCount),
			ExtraData: map[string]string{
				"equipment_id":  fmt.Sprintf("%d", equipmentID),
				"level":         fmt.Sprintf("%d", level),
				"quantity":      fmt.Sprintf("%d", quantity),
				"success_count": fmt.Sprintf("%d", len(equipmentOwnIDs)),
				"failed_count":  fmt.Sprintf("%d", failedCount),
				"user_id":       fmt.Sprintf("%d", userId),
			},
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("添加装备成功: 装备ID=%d, 等级=%d, 数量=%d", equipmentID, level, quantity),
		ExtraData: map[string]string{
			"equipment_id": fmt.Sprintf("%d", equipmentID),
			"level":        fmt.Sprintf("%d", level),
			"quantity":     fmt.Sprintf("%d", quantity),
			"user_id":      fmt.Sprintf("%d", userId),
		},
	}
}

// handleRemoveEquipment 处理移除装备命令
// 参数说明：
// - item_id: 装备唯一ID (equipment_own_id)
func (h *EquipmentGMHandler) handleRemoveEquipment(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	equipmentOwnID := int64(req.GetItemId()) // 使用item_id作为equipment_own_id

	if equipmentOwnID <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "装备唯一ID必须大于0（暂不支持批量删除）",
		}
	}

	// 获取装备列表，找到要删除的装备
	equipments, err := h.equipmentService.GetEquipmentList(userId)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("获取装备列表失败: %v", err),
		}
	}

	// 查找装备
	var found bool
	for _, eq := range equipments {
		if eq.GetEquipmentOwnId() == equipmentOwnID {
			found = true
			break
		}
	}

	if !found {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("装备不存在: equipment_own_id=%d", equipmentOwnID),
		}
	}

	// 分解装备（软删除）
	decomposeResp, err := h.equipmentService.DecomposeEquipments(userId, []int64{equipmentOwnID})
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("删除装备失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("删除装备成功: equipment_own_id=%d", equipmentOwnID),
		ExtraData: map[string]string{
			"equipment_own_id": fmt.Sprintf("%d", equipmentOwnID),
			"user_id":          fmt.Sprintf("%d", userId),
			"drop_count":       fmt.Sprintf("%d", len(decomposeResp.GetDropId())),
		},
	}
}
