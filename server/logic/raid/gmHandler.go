package raid

import (
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"

	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*RaidGMHandler)(nil)

// RaidGMHandler 英雄系统GM命令处理器
type RaidGMHandler struct {
	sessionManager logicCommon.SessionManagerInterface
}

// NewRaidGMHandler 创建英雄系统GM命令处理器
func NewRaidGMHandler(manager logicCommon.SessionManagerInterface) *RaidGMHandler {
	return &RaidGMHandler{
		sessionManager: manager,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *RaidGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_CHANGE_MAIN_STAGE,
	}
}

// HandleCommand 处理GM命令
func (h *RaidGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_CHANGE_MAIN_STAGE:
		return h.handleRaidChangeMainStage(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("副本系统不支持的命令类型: %d", cmdType),
		}
	}
}

// handleRaidLevelUp 处理英雄升级命令
func (h *RaidGMHandler) handleRaidChangeMainStage(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	stageId := req.MainStageId
	subStageId := req.MainSubStageId

	stageConfig := gameConfig.GetMainStageCfg(stageId)
	if stageConfig == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "无效的副本ID",
		}
	}
	find := false
	for _, id := range stageConfig.SubStageId {
		if subStageId == id {
			find = true
			break
		}
	}
	if !find {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "无效的子关卡ID",
		}
	}

	// 获取玩家模型
	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	playerModel := p.(*model.PlayerModel)

	entity := playerModel.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if entity == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家无副本数据",
		}
	}
	current, next, err := BuildAllMainInstanceData(userId, stageId, subStageId, entity.MaxStageId, entity.MaxSubStageId, logicCommon.NewInstanceStageInfo(), playerModel.StaticData.GetDailyPrivilegeDrop())
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "副本数据错误",
		}
	}
	playerModel.PlayerInstanceModel.CurrentRaidInfo = current
	playerModel.PlayerInstanceModel.CurrentMainInstanceInfo = current
	playerModel.PlayerInstanceModel.NextMainInstanceInfo = next
	playerModel.PlayerInstanceModel.UpdateMainInstanceInfo(stageId, subStageId, 0)

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("副本数据更新成功: StageId=%d, SubStageId=%d", stageId, subStageId),
	}
}
