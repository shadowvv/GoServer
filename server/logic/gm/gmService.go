// File: gmService.go
// Description: GM系统服务实现
// Author: 木村凉太
// Create Time: 2026.02

package gm

import (
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"

	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMServiceInterface = (*GMService)(nil)

// GMService GM服务实现（分发器模式）
type GMService struct {
	// handlers 命令类型到处理器的映射
	handlers map[pb.GMCommandType]logicCommon.GMCommandHandler
}

// NewGMService 创建GM服务
func NewGMService() *GMService {
	return &GMService{
		handlers: make(map[pb.GMCommandType]logicCommon.GMCommandHandler),
	}
}

// RegisterHandler 注册命令处理器
func (s *GMService) RegisterHandler(handler logicCommon.GMCommandHandler) {
	for _, cmdType := range handler.GetSupportedCommands() {
		if existing, exists := s.handlers[cmdType]; exists {
			// 如果已存在处理器，记录警告但不覆盖（可以根据需要调整策略）
			fmt.Printf("[GMService] Warning: Command type %d already has handler %T, new handler %T will not be registered\n",
				cmdType, existing, handler)
			continue
		}
		s.handlers[cmdType] = handler
	}
}

// CheckInvokerEnabled 检查调用方是否启用
func (s *GMService) CheckInvokerEnabled(invokerId string, invokerType pb.GMInvokerType) (bool, error) {
	// 非正式环境，默认允许调用
	if nodeConfig.Env != enum.ENV_PRODUCT && nodeConfig.Env != enum.ENV_STAGE {
		return true, nil
	}

	// 正式环境, 仅允许程序内部调用或RPC调用
	if nodeConfig.Env == enum.ENV_PRODUCT || nodeConfig.Env == enum.ENV_STAGE {
		return invokerType == pb.GMInvokerType_GM_INVOKER_INTERNAL || invokerType == pb.GMInvokerType_GM_INVOKER_RPC, nil
	}

	return false, nil
}

// ExecuteCommand 执行GM命令
func (s *GMService) ExecuteCommand(req *pb.MessageGmReq, invokerType pb.GMInvokerType, invokerId string) (*pb.MessageGmResp, error) {
	logger.InfoWithSprintf("[GMService] type: %v", req.CmdType)

	// 检查调用方是否启用
	enabled, err := s.CheckInvokerEnabled(invokerId, invokerType)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("检查调用方状态失败: %v", err),
		}, nil
	}

	if !enabled {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVOKER_DISABLED,
			Message: "调用方已禁用",
		}, nil
	}

	// 获取用户ID
	userId := req.GetUserId()
	if userId == 0 && invokerType == pb.GMInvokerType_GM_INVOKER_CLIENT {
		// 客户端调用时，用户ID应该从session中获取，这里暂时返回错误
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_USER,
			Message: "用户ID不能为空",
		}, nil
	}

	//// 记录命令日志
	//log := &model.GMCommandLogEntity{
	//	ID:          tool.NewIdGenerator(1, int64(enum.ID_GENERATOR_GM)).NextId(),
	//	UserId:      userId,
	//	InvokerId:   invokerId,
	//	InvokerType: int32(invokerType),
	//	CmdType:     int32(req.GetCmdType()),
	//	Result:      0,
	//	Message:     "",
	//	CreatedAt:   time.Now(),
	//}

	//// 序列化命令参数
	//cmdParams, _ := json.Marshal(req)
	//log.CmdParams = string(cmdParams)

	// 执行命令
	resp := s.executeCommandInternal(req, userId)

	//// 记录结果
	//log.Result = int32(resp.GetResult())
	//log.Message = resp.GetMessage()

	return resp, nil
}

// executeCommandInternal 内部执行GM命令（分发到对应的处理器）
func (s *GMService) executeCommandInternal(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	// 查找对应的处理器
	handler, exists := s.handlers[cmdType]
	if !exists {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("未找到命令类型 %d 的处理器，请确保对应的系统已注册", cmdType),
		}
	}

	// 分发到对应的处理器
	return handler.HandleCommand(req, userId)
}
