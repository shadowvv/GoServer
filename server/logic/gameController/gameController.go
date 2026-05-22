package gameController

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/adventure"
	"github.com/drop/GoServer/server/logic/cityAge"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pass"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/eventService"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/rankBoardPlatform"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/logic/socialService"
	"github.com/drop/GoServer/server/logic/task"
	"github.com/drop/GoServer/server/logic/trial"
	"github.com/drop/GoServer/server/logic/turnTable"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type PlayerMessageHandler func(message proto.Message, player *model.PlayerModel)
type AllianceMessageHandler func(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel)

type LogicControllerInterface interface {
	RegisterLogicMessage()
}

var allController = make(map[string]LogicControllerInterface)
var eventBusService *eventService.EventBus
var routerService serviceInterface.RouterInterface
var dispatcher *dispatcherService.Dispatcher
var messageSender logicCommon.MessageSenderInterface
var sessionManager logicCommon.SessionManagerInterface
var unlockService logicCommon.UnlockServiceInterface
var activityService logicCommon.GameActivityServiceInterface
var passService logicCommon.PassServiceInterface

func RegisterController(name string, controller LogicControllerInterface) {
	allController[name] = controller
}

func InitGatewayController(router serviceInterface.RouterInterface, manager logicCommon.SessionManagerInterface, d *dispatcherService.Dispatcher, sender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface, activity logicCommon.GameActivityServiceInterface) {
	routerService = router
	dispatcher = d
	messageSender = sender
	sessionManager = manager
	unlockService = unlock
	activityService = activity
	InitGatewayIdGenerator()
}

func InitRankBoardController(router serviceInterface.RouterInterface, manager logicCommon.SessionManagerInterface, d *dispatcherService.Dispatcher, sender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface, activity logicCommon.GameActivityServiceInterface) {
	routerService = router
	dispatcher = d
	messageSender = sender
	sessionManager = manager
	unlockService = unlock
	activityService = activity
}

func InitSocialController(router serviceInterface.RouterInterface, manager logicCommon.SessionManagerInterface, d *dispatcherService.Dispatcher, sender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface, activity logicCommon.GameActivityServiceInterface) {
	routerService = router
	dispatcher = d
	messageSender = sender
	sessionManager = manager
	unlockService = unlock
	activityService = activity
}

func InitGameController(bus *eventService.EventBus, router serviceInterface.RouterInterface, manager logicCommon.SessionManagerInterface, d *dispatcherService.Dispatcher, sender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface, activity logicCommon.GameActivityServiceInterface) {
	eventBusService = bus
	routerService = router
	dispatcher = d
	messageSender = sender
	sessionManager = manager
	unlockService = unlock
	activityService = activity
	passService = gamePlatform.GetPassService()

	// 在 unlockService 就绪后初始化各子系统（包括挂机系统）
	InitEquipmentService()
	// 初始化背包系统
	InitinvService()
	// 初始化GM系统
	InitGMService(sessionManager)
	// 初始化邮件服务
	InitMailService(sessionManager, messageSender, unlockService, gamePlatform.GetServerInfoService())
	// 通行证过期未领奖励发邮件：注入 Mail（登录时检测补发，不跑定时器）
	if ps, ok := passService.(*pass.PassService); ok && GetMailService() != nil {
		ps.MailService = GetMailService()
	}
	// 初始化任务系统
	task.TaskInitServer(bus, messageSender, dispatcher)
	// 初始化主城时代
	cityAge.InitCityAgeService(messageSender)
	turnTable.InitTurnTableService(activityService, messageSender)
	// 初始化道具系统
	itemService.RegisterItemService(messageSender, invService, equipmentService, eventBusService, passService)
	// 初始化英雄系统
	hero.InitHeroService(unlockService, messageSender)
	// 初始化宠物系统
	pet.InitPetService(unlockService)
	// 初始化挂机系统
	InitIdleService()
	// 初始化家具系统
	InitFurnitureService()
	// 初始化七日试炼
	trial.InitTrialService(activityService, messageSender, itemService.GetItemCount, GetMailService())
	// 初始化秘境副本
	adventure.InitAdventureService(messageSender, unlockService)
}

func RegisterAllMessage() {
	for name, controller := range allController {
		logger.InfoWithSprintf("[gameController] register controller: %s", name)
		controller.RegisterLogicMessage()
	}
}

func RegisterLoginMessageHandler(msgType enum.MessageType, msgID pb.MESSAGE_ID, msg proto.Message, h logicCommon.LoginMessageHandler) {
	if msgType != enum.MSG_TYPE_LOGIN {
		panic(fmt.Sprintf("[gameController] register login process msgType error type:%d msgId:%d", msgType, msgID))
		return
	}

	messageId := int32(msgID)
	routerService.RegisterMessage(int32(msgType), messageId, msg)
	dispatcher.RegisterLoginMessageHandler(messageId, h)
}

func RegisterPlayerMessageHandler(msgType enum.MessageType, msgID pb.MESSAGE_ID, msg proto.Message, h PlayerMessageHandler, function enum.FunctionIdEnum) {
	if msgType != enum.MSG_TYPE_PLAYER {
		panic(fmt.Sprintf("[gameController] register player process msgType error type:%d msgId:%d", msgType, msgID))
		return
	}

	messageHandler := func(message proto.Message, player logicCommon.PlayerInterface, function enum.FunctionIdEnum) {
		if msgID != pb.MESSAGE_ID_HEART_REQ {
			logger.InfoWithSprintf("[gameController] player:%d, receive message messageId:%d,message:%v", player.GetUserId(), msgID, message)
		}
		p, ok := player.(*model.PlayerModel)
		if !ok {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameController] player message covert error player:%v", player))
			messageSender.SendErrorMessage(player, msgID+1, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		unlockConfig := gameConfig.GetSystemUnlockCfg(int32(function))
		if unlockConfig != nil {
			find := true
			if unlockConfig.AreaLimit == nil || len(unlockConfig.AreaLimit) == 0 {
				find = true
			} else {
				for _, areaId := range unlockConfig.AreaLimit {
					if p.User.GetChannelId() == areaId {
						find = true
						break
					}
				}
			}
			if !find {
				logger.InfoWithSprintf("[gameController] player function not unlock functionId:%d,playerId:%d", function, player.GetUserId())
				messageSender.SendErrorMessage(player, msgID+1, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
				return
			}
			for _, unlock := range unlockConfig.UnlockId {
				if !unlockService.CheckUnlock(unlock, player.(*model.PlayerModel)) {
					logger.InfoWithSprintf("[gameController] player function not unlock functionId:%d,playerId:%d", function, player.GetUserId())
					messageSender.SendErrorMessage(player, msgID+1, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
					return
				}
			}
		}
		h(message, p)
	}
	messageId := int32(msgID)
	routerService.RegisterMessage(int32(msgType), messageId, msg)
	dispatcher.RegisterPlayerMessageHandler(messageId, messageHandler, function)
}

func RegisterRpcPlayerMessageHandler(msgType enum.MessageType, msgID rpcPb.RPC_MESSAGE_ID, msg proto.Message, h PlayerMessageHandler) {
	if msgType != enum.MSG_TYPE_PLAYER {
		panic(fmt.Sprintf("[gameController] register player process msgType error type:%d msgId:%d", msgType, msgID))
		return
	}

	messageHandler := func(message proto.Message, player logicCommon.PlayerInterface, function enum.FunctionIdEnum) {
		p, ok := player.(*model.PlayerModel)
		if !ok {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameController] player message covert error player:%v", player))
			return
		}
		h(message, p)
	}
	messageId := int32(msgID)
	routerService.RegisterMessage(int32(msgType), messageId, msg)
	dispatcher.RegisterPlayerMessageHandler(messageId, messageHandler, enum.FUNCTION_ID_NONE)
}

func RegisterRankBoardMessageHandler(msgType enum.MessageType, msgID rpcPb.RPC_MESSAGE_ID, msg proto.Message, h logicCommon.RankBoardMessageHandler) {
	if msgType != enum.MSG_TYPE_RANKBOARD {
		panic(fmt.Sprintf("[gameController] register login process msgType error type:%d msgId:%d", msgType, msgID))
		return
	}

	messageId := int32(msgID)
	routerService.RegisterMessage(int32(msgType), messageId, msg)
	rankBoardPlatform.RegisterRankBoardMessageHandler(messageId, h)
}

func RegisterAllianceMessageHandler(msgType enum.MessageType, msgID rpcPb.RPC_MESSAGE_ID, msg proto.Message, h AllianceMessageHandler) {
	if msgType != enum.MSG_TYPE_Alliance {
		panic(fmt.Sprintf("[gameController] register alliance process msgType error type:%d msgId:%d", msgType, msgID))
		return
	}
	messageHandler := func(message proto.Message, session serviceInterface.SessionInterface, alliance logicCommon.AllianceInterface) {
		if session == nil {
			logger.ErrorBySprintf("[gameController] msgId:%d session is nil", msgID)
			return
		}
		socialSession, ok := session.(*logicSessionManager.AllianceSession)
		if !ok {
			logger.ErrorBySprintf("[gameController] msgId:%d session is not social session", msgID)
			return
		}

		var allianceModel *socialService.AllianceModel = nil
		if msgID != rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_REQ {
			allianceModel, ok = alliance.(*socialService.AllianceModel)
			if !ok {
				logger.ErrorBySprintf("[gameController] msgId:%d alliance transform err", msgID)
				return
			}
		}
		h(message, socialSession, allianceModel)
	}
	messageId := int32(msgID)
	routerService.RegisterMessage(int32(msgType), messageId, msg)
	dispatcher.RegisterAllianceMessageHandler(messageId, messageHandler)
}

func RegisterPlayerInnerTask(innerMsgId enum.InnerMessageId, h logicCommon.InnerMessageHandler) {
	dispatcher.RegisterInnerMessageHandler(enum.INNER_MSG_TYPE_PLAYER, int32(innerMsgId), h)
}
