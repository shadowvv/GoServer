// File: gmInterface.go
// Description: GM系统服务接口定义
// Author:木村凉太
// Create Time: 2026.02

package logicCommon

import (
	"github.com/drop/GoServer/server/logic/pb"
)

// GMCommandHandler GM命令处理器接口
// 各个系统实现此接口来处理自己负责的GM命令
type GMCommandHandler interface {
	// HandleCommand 处理GM命令
	// req: GM命令请求
	// userId: 用户ID
	// 返回GM命令响应
	HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp

	// GetSupportedCommands 返回该处理器支持的命令类型列表
	GetSupportedCommands() []pb.GMCommandType
}

// GMServiceInterface GM服务接口
type GMServiceInterface interface {
	// ExecuteCommand 执行GM命令
	// req: GM命令请求，包含命令类型和参数
	// invokerType: 调用方类型 1:客户端 2:内部 3:RPC
	// invokerId: 调用方标识（如IP、服务名等）
	// 返回GM命令响应和错误信息
	ExecuteCommand(req *pb.MessageGmReq, invokerType pb.GMInvokerType, invokerId string) (*pb.MessageGmResp, error)

	// CheckInvokerEnabled 检查调用方是否启用
	// invokerId: 调用方标识
	// 返回是否启用和错误信息
	CheckInvokerEnabled(invokerId string, invokerType pb.GMInvokerType) (bool, error)

	// RegisterHandler 注册命令处理器
	// handler: 命令处理器实现
	RegisterHandler(handler GMCommandHandler)
}
