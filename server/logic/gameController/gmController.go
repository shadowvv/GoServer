// File: gmController.go
// Description: GM系统控制器
// Author: 木村凉太
// Create Time: 2025.11

package gameController

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/raid"
	"github.com/drop/GoServer/server/logic/task"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/equipment"
	"github.com/drop/GoServer/server/logic/gm"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/inventory"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

var gmService logicCommon.GMServiceInterface

// InitGMService 初始化GM服务并注册各个系统的处理器
func InitGMService(manager logicCommon.SessionManagerInterface) {
	gmService = gm.NewGMService()

	// 注册背包系统处理器
	invHandler := inventory.NewInventoryGMHandler(manager, itemService.GetItemService())
	gmService.RegisterHandler(invHandler)

	// 注册英雄系统处理器
	heroHandler := hero.NewHeroGMHandler(manager)
	gmService.RegisterHandler(heroHandler)

	// 注册宠物系统处理器
	petHandler := pet.NewPetGMHandler(manager)
	gmService.RegisterHandler(petHandler)

	// 注册装备系统处理器
	equipmentHandler := equipment.NewEquipmentGMHandler(equipmentService)
	gmService.RegisterHandler(equipmentHandler)

	// 注册邮件系统处理器
	mailHandler := mail.NewMailGMHandler(mailService)
	gmService.RegisterHandler(mailHandler)

	raidHandler := raid.NewRaidGMHandler(manager)
	gmService.RegisterHandler(raidHandler)

	taskHandler := task.NewTaskGMHandler(manager)
	gmService.RegisterHandler(taskHandler)

	// 初始化内部GM服务（供程序内部调用）
	gm.InitInternalGMService(gmService)
}

func init() {
	RegisterController("gm", &GmController{})
}

type GmController struct {
}

var _ LogicControllerInterface = (*GmController)(nil)

// RegisterGMMessage 注册GM消息处理
func (g *GmController) RegisterLogicMessage() {
	// 使用已有的 MESSAGE_ID_MESSAGE_GM = 2
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MESSAGE_GM_REQ, &pb.MessageGmReq{}, GMCommandHandle, enum.FUNCTION_ID_NONE)
}

// GMCommandHandle 处理GM命令请求
func GMCommandHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MessageGmReq)
	if !ok {
		return
	}

	// 从session中获取调用方信息
	session := player.GetSession()
	invokerId := fmt.Sprintf("client_%d", session.GetID())
	invokerType := pb.GMInvokerType_GM_INVOKER_CLIENT

	// 如果用户ID为空，使用登录用户的ID
	if req.GetUserId() == 0 {
		req.UserId = player.GetUserId()
	}

	// 设置调用方类型和标识
	req.InvokerType = invokerType
	req.InvokerId = invokerId

	// 执行GM命令
	resp, err := gmService.ExecuteCommand(req, invokerType, invokerId)
	if err != nil {
		resp = &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("执行GM命令失败: %v", err),
		}
	}

	// 发送响应
	session.Send(int32(pb.MESSAGE_ID_MESSAGE_GM_RESP), resp)
}
