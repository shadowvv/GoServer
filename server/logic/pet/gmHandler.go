// File: gmHandler.go
// Description: 宠物系统GM命令处理器（测试用生成宠物）

package pet

import (
	"encoding/json"
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*PetGMHandler)(nil)

// PetGMHandler 宠物系统GM命令处理器
type PetGMHandler struct {
	sessionManager logicCommon.SessionManagerInterface
}

// NewPetGMHandler 创建宠物系统GM命令处理器
func NewPetGMHandler(manager logicCommon.SessionManagerInterface) *PetGMHandler {
	return &PetGMHandler{
		sessionManager: manager,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *PetGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_ADD_PET,
	}
}

// HandleCommand 处理GM命令
func (h *PetGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	switch req.GetCmdType() {
	case pb.GMCommandType_GM_CMD_ADD_PET:
		return h.handleAddPet(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("宠物系统不支持的命令类型: %d", req.GetCmdType()),
		}
	}
}

type addPetParams struct {
	ItemId int32 `json:"itemId"`
	PetId  int32 `json:"petId"`
	Direct int32 `json:"direct"` // 1=不走道具，直接生成宠物（仅测试使用）
	Level  int32 `json:"level"`  // direct=1 时可选，默认1
}

// handleAddPet 处理添加宠物命令
func (h *PetGMHandler) handleAddPet(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	var params addPetParams
	if extra := req.GetExtraParams(); extra != "" {
		_ = json.Unmarshal([]byte(extra), &params)
	}

	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_USER,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	// direct=1：不走 item，不需要宠物卡 itemId（用于 testbot 直接生成）
	if params.Direct == 1 {
		if params.PetId <= 0 {
			return &pb.MessageGmResp{
				Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
				Message: "direct=1 时请传 petId(宠物配置ID)",
			}
		}
		ent, err := obtainPetWithLevel(player, params.PetId, params.Level)
		if err != nil {
			return &pb.MessageGmResp{
				Result:  pb.GMResult_GM_RESULT_FAILED,
				Message: fmt.Sprintf("直接生成宠物失败: PetId=%d, Error=%s", params.PetId, err.Error()),
			}
		}
		detail := BuildPetDetailInfo(player, ent)
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_SUCCESS,
			Message: fmt.Sprintf("直接生成宠物成功: PetId=%d, PetOwnId=%d", ent.PetID, detail.GetPetOwnId()),
			ExtraData: map[string]string{
				"pet_id":     fmt.Sprintf("%d", ent.PetID),
				"pet_own_id": fmt.Sprintf("%d", detail.GetPetOwnId()),
				"user_id":    fmt.Sprintf("%d", userId),
			},
		}
	}

	// 仅走卡片逻辑：1) req.itemId 2) extra.itemId。
	cardItemID := int32(req.GetItemId())
	if cardItemID <= 0 {
		cardItemID = params.ItemId
	}
	// 支持传 petId（更贴近业务）：反查对应宠物卡 itemId 后走同一条路径
	if cardItemID <= 0 && params.PetId > 0 {
		cardItemID = gameConfig.GetPetCardItemID(params.PetId)
		if cardItemID <= 0 {
			return &pb.MessageGmResp{
				Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
				Message: fmt.Sprintf("petId 未找到对应宠物卡 itemId: petId=%d", params.PetId),
			}
		}
	}
	if cardItemID <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "请传 itemId(宠物卡道具ID) 或 petId(宠物配置ID)",
		}
	}
	itemCfg := gameConfig.GetItemCfg(cardItemID)
	if itemCfg == nil || itemCfg.ShowGroup != int32(enum.ITEM_TYPE_PET) {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("itemId 不是有效宠物卡: itemId=%d", cardItemID),
		}
	}
	ent, err := ObtainPetByCard(player, cardItemID)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("添加宠物失败: ItemId=%d, Error=%s", cardItemID, err.Error()),
		}
	}
	petId := ent.PetID
	detail := BuildPetDetailInfo(player, ent)

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("添加宠物成功: PetId=%d, ItemId=%d, PetOwnId=%d", petId, cardItemID, detail.GetPetOwnId()),
		ExtraData: map[string]string{
			"pet_id":     fmt.Sprintf("%d", petId),
			"item_id":    fmt.Sprintf("%d", cardItemID),
			"pet_own_id": fmt.Sprintf("%d", detail.GetPetOwnId()),
			"user_id":    fmt.Sprintf("%d", userId),
		},
	}
}
