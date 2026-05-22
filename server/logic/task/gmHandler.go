package task

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*TaskGMHandler)(nil)

// TaskGMHandler 任务系统GM命令处理器
type TaskGMHandler struct {
	sessionManager logicCommon.SessionManagerInterface
}

// NewTaskGMHandler 创建任务系统GM命令处理器
func NewTaskGMHandler(manager logicCommon.SessionManagerInterface) *TaskGMHandler {
	return &TaskGMHandler{
		sessionManager: manager,
	}
}

func (t *TaskGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_JUMP_THIS_MAIN_TASK:
		return t.handleJumpThisMainTask(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("任务系统不支持的命令类型: %d", cmdType),
		}
	}
}

func (t *TaskGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_JUMP_THIS_MAIN_TASK,
	}
}

func (t *TaskGMHandler) handleJumpThisMainTask(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	p := t.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)
	for _, v := range player.TaskModel.TaskEntity[enum.TaskAffiliationMain] {
		for _, value := range v {
			if value.Status == 0 {
				taskCoreId, _ := gameConfig.GetCoreTaskId(enum.TaskAffiliationMain, value.TaskID)
				taskCfg := gameConfig.GetCoreCfg(taskCoreId)
				player.TaskModel.UpdateTaskStatus(value.TaskID, taskCfg.TaskType, enum.TaskAffiliationMain, enum.TaskStatusFinishUnReward)
				player.TaskModel.UpdateTaskProgressData(value.TaskID, taskCfg.TaskType, enum.TaskAffiliationMain, taskCfg.TaskNum)
				messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, &pb.PushTaskUpdate{
					Attribution: enum.TaskAffiliationMain,
					TaskId:      value.TaskID,
					TaskState:   enum.TaskStatusFinishUnReward,
					Progress:    taskCfg.TaskNum,
				})
				break
			}
		}
	}
	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: "跳转任务成功",
	}
}
