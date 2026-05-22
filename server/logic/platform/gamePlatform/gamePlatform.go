package gamePlatform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/activityService"
	"github.com/drop/GoServer/server/logic/equipment"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/inventory"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/platform/payOrderService"
	"github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/service/easyRpc"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pass"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/dbPool"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/eventService"
	"github.com/drop/GoServer/server/logic/platform/logicCodec"
	"github.com/drop/GoServer/server/logic/platform/logicRouter"
	"github.com/drop/GoServer/server/logic/platform/logicScene"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/rankboardService"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/service/wordFilter"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type GameSessionCloserHooker struct {
}

var _ logicCommon.SessionCloseHooker = (*GameSessionCloserHooker)(nil)

func (g *GameSessionCloserHooker) OnSessionClose(player logicCommon.PlayerInterface) {
	if player == nil {
		return
	}
	dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MSG_PLAYER_LOGOUT, player.GetUserId(), nil, 0, 0, nil)
}

func RemovePlayerFromGame(playerModel *model.PlayerModel) {
	// Avoid mutating user model from a detached goroutine.
	playerModel.User.UpdateLastOfflineTime(tool.UnixNowMilli())
	playerModel.BuildPlayerCacheInfo()
	logicCommon.UpdatePlayerBasicInfo(playerModel.PlayerCacheInfo.BasicInfo)
	playerModel.User.SaveModelToDB()

	go func() {
		if loginMutexService.EnterMutex(playerModel.GetUserAccount(), playerModel.GetSession().GetID()) {
			logger.InfoWithSprintf("player logout,playerId:%d,sessionId:%d", playerModel.GetUserId(), playerModel.GetSession().GetID())
			err := easyDB.WaitPlayerBarrier(playerModel.GetUserId(), 60*time.Second)
			if err != nil {
				platformLogger.ErrorWithUser("wait player barrier timeout", playerModel, err)
			}

			_, err = sceneService.LeaveScene(playerModel.GetUserId())
			if err != nil {
				platformLogger.ErrorWithUser("logout leave scene error", playerModel, err)
				loginMutexService.ExitMutex(playerModel.GetUserAccount(), playerModel.GetSession().GetID())
				return
			}
			gameSessionManager.RemovePlayer(playerModel)
			loginMutexService.ExitMutex(playerModel.GetUserAccount(), playerModel.GetSession().GetID())
		}
	}()
}

type GameRpcHooker struct {
}

var _ logicCommon.RPCServiceHooker = (*GameRpcHooker)(nil)

func (g *GameRpcHooker) OnNodeConnect(id int32, nodeType enum.NodeType) {
	if nodeType == enum.NODE_TYPE_RANKBOARD {
		ServerNodeService.InitRankBoardRpcClient()
		rpcController.InitRankBoardRpcClients()
	} else if nodeType == enum.NODE_TYPE_GATEWAY {
		ServerNodeService.InitGateRpcClient()
		rpcController.InitGatewayRpcClients()
	} else if nodeType == enum.NODE_TYPE_SOCIAL {
		if !rpcController.IsSocialRpcEnabled() {
			return
		}
		ServerNodeService.InitSocialRpcClient()
		rpcController.InitSocialRpcClients()
	}
}

func (g *GameRpcHooker) OnNodeDisconnect(id int32, nodeType enum.NodeType, nodeAddress string) {
	easyRpc.CloseConnect(nodeAddress)
}

type gameMessageSender struct {
}

var _ logicCommon.MessageSenderInterface = (*gameMessageSender)(nil)

func (g *gameMessageSender) SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message) {
	player := gameSessionManager.GetPlayerBasicInfoByUserId(playerId)
	if player == nil {
		return
	}
	if player.GetSession() == nil {
		return
	}
	if !player.GetSession().IsActive() {
		return
	}
	player.GetSession().Send(int32(msgId), message)
}

func (g *gameMessageSender) SendMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message) {
	if msgId != pb.MESSAGE_ID_HEART_RESP {
		platformLogger.InfoWithUser(fmt.Sprintf("send msgId:%d msg:%+v", msgId, message), player)
	}
	if player.GetSession() == nil {
		return
	}
	if !player.GetSession().IsActive() {
		return
	}
	player.GetSession().Send(int32(msgId), message)
}

func (g *gameMessageSender) Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int32) {
	frame, err := codec.Marshal(int32(msgId), msg)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] broadcast messageId:%d marshal error: %v", msgId, err))
		return
	}
	if int32(len(frame)) > 256*1024 {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] broadcast send messageId:%d msg too large: %d", msgId, len(frame)))
		return
	}

	resp := &rpcPb.BackwardClientMessage{
		MsgId:         int32(msgId),
		Payload:       frame,
		BroadcastType: int32(broadcastType),
		BroadcastId:   typeId,
	}
	err = rpcController.BroadcastMessageToGateway(resp)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] broadcast send messageId:%d error: %v", msgId, err))
		return
	}
}

func (g *gameMessageSender) SendErrorMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	errorMsg := &pb.MessageError{
		MsgId:     msgId,
		ErrorCode: errorCode,
	}
	g.SendMessage(player, pb.MESSAGE_ID_MESSAGE_ERROR, errorMsg)
}

func (g *gameMessageSender) CloseSessionWithError(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	errorMsg := &pb.MessageError{
		MsgId:     msgId,
		ErrorCode: errorCode,
	}
	platformLogger.InfoWithUser(fmt.Sprintf("send msg:%+v", errorMsg), player)
	player.GetSession().SendAndClose(int32(pb.MESSAGE_ID_MESSAGE_ERROR), errorMsg)
}

type gameRpcMessageSender struct {
}

var _ logicCommon.RpcMessageSenderInterface = (*gameRpcMessageSender)(nil)

func (g gameRpcMessageSender) SendMessageToRankBoard(userId int64, rankId string, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error {
	return rpcController.SendMessageToRankBoard(userId, rankId, backMessageId, msgId, req)
}

var eventBus *eventService.EventBus
var router serviceInterface.RouterInterface                        // 消息路由
var gameSessionManager *logicSessionManager.GameSessionManager     // 游戏会话管理
var dispatcher *dispatcherService.Dispatcher                       // 消息分发
var dbPoolManager *dbPool.DBPoolManager                            // 数据库连接池
var sceneService *logicScene.SceneService                          // 场景服务
var serverInfoService *gameServerInfoService.GameServerInfoService // 游戏服务器信息
var activityInfoService *activityService.GameActivityService       // 活动信息
var messageSender logicCommon.MessageSenderInterface               // 消息发送
var unlock logicCommon.UnlockServiceInterface                      // 解锁服务
var passService logicCommon.PassServiceInterface                   // 通行证服务
var gloryArenaMatchService *gloryArenaService.GameGloryArenaService
var codec serviceInterface.CodecInterface
var rpcMessageSender logicCommon.RpcMessageSenderInterface

func BootGamePlatform() {
	cfg := platform.BootBasicService()

	data, _ := json.MarshalIndent(cfg.ServerDBConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init server db config:%s", string(data))
	serverDB, err := dbService.InitMySQL(cfg.ServerDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetServerDB(serverDB)
	data, _ = json.MarshalIndent(cfg.GameDBConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init game db config:%s", string(data))
	gameDB, err := dbService.InitMySQL(cfg.GameDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init game db error", zap.Error(err))
		panic("[platform] Init game db error")
	}
	dbPoolManager = dbPool.NewDBPoolManager(gameDB)
	for _, poolInfo := range cfg.RunConfig.DBPoolInfo {
		dbPoolManager.AddDBPool(enum.DBPoolType(poolInfo.PoolType), poolInfo.WorkerNum, poolInfo.WorkerTaskSize)
	}
	easyDB.SetGameDBPool(dbPoolManager)
	// 初始化日志库
	if cfg.LogDBConfig != nil {
		data, _ = json.MarshalIndent(cfg.LogDBConfig, "", "  ")
		logger.InfoWithSprintf("[platform] Init log db config:%s", string(data))
		logDB, err := dbService.InitMySQL(cfg.LogDBConfig, logger.Logger)
		if err != nil {
			logger.ErrorWithZapFields("[platform] Init log db error", zap.Error(err))
			panic("[platform] Init log db error")
		}
		easyDB.SetLogDB(logDB)
	} else {
		logger.InfoWithSprintf("[platform] logDBConfig not found, using gameDB as log db")
		easyDB.SetLogDB(gameDB)
	}
	// 初始化redis
	data, _ = json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init redis error", zap.Error(err))
		panic("[platform] Init redis error")
	}
	hero.InitHero()
	inventory.InitInventory()
	pet.InitPet()
	equipment.InitEquipment()

	// 事件总线
	eventBus = eventService.NewEventBus(eventService.BufferSize)
	eventBus.Start()
	// 游戏会话,消息路由,分发
	gameSessionManager = logicSessionManager.NewGameSessionManager(&GameSessionCloserHooker{})
	logger.InfoWithSprintf("[platform] init session manager")
	dispatcher = dispatcherService.NewDispatcher(router, gameSessionManager, tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_INNER_MESSAGE)), enum.NodeType(nodeConfig.NodeType), cfg.RunConfig.MessageProcessConfig)
	logger.InfoWithSprintf("[platform] init message dispatcher")
	router = logicRouter.NewGameRouter(dispatcher)
	logger.InfoWithSprintf("[platform] init router")
	// 游戏服务器信息服务
	serverInfoService = gameServerInfoService.NewGameServerInfoService()
	unlock = unlockService.NewUnlockService(serverInfoService)
	logger.InfoWithSprintf("[platform] init server info service")
	activityInfoService = activityService.NewGameActivityService(serverInfoService, unlock)
	if unlockImpl, ok := unlock.(*unlockService.UnlockService); ok {
		unlockImpl.SetActivityService(activityInfoService)
	}
	logger.InfoWithSprintf("[platform] init activity info service")
	messageSender = &gameMessageSender{}

	// 节点RPC服务
	hooker := &GameRpcHooker{}
	ServerNodeService.InitNodeService(hooker, nodeConfig.NodeId, enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, cfg.EtcdConfig, "", rpcController.RegisterGameRpcServer)
	rpcController.InitRpcController(enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, dispatcher, gameSessionManager, messageSender)
	codec = logicCodec.NewGameCodec()
	rpcController.InitGameRpc(gameSessionManager, codec, router, activityInfoService, serverInfoService, cfg.RunConfig.PlayerOfflineTimeout)
	logger.InfoWithSprintf("[platform] init node proxy service")
	gameConfig.LoadAllConfig()
	logger.InfoWithSprintf("[platform] load all config success !!!")

	// 游戏服服务初始化
	passService = &pass.PassService{
		ActivityService: activityInfoService,
	}
	logger.InfoWithSprintf("[platform] init pass service")
	gloryArenaMatchService = gloryArenaService.NewGloryArenaService()
	logger.InfoWithSprintf("[platform] init glory arena service")
	wordFilter.InitWordFilterService("config/dirtyWord.txt")
	logger.InfoWithSprintf("[platform] Init word filter service")
	sceneProcessConfig := cfg.RunConfig.MessageProcessConfig["scene"]
	if sceneProcessConfig == nil {
		panic("[platform] Scene process config is nil")
	}
	playerProcessConfig := cfg.RunConfig.MessageProcessConfig["player"]
	if playerProcessConfig == nil {
		panic("[platform] Player process config is nil")
	}
	sceneService = logicScene.NewSceneService(gameSessionManager, dispatcher, sceneProcessConfig, cfg.RunConfig.SceneGoroutineMaxPlayerNum, playerProcessConfig)
	logger.InfoWithSprintf("[platform] init scene service")
	// 初始化排行榜系统
	rankboardService.InitEventHandler(eventBus, activityInfoService, dispatcher)
	rpcMessageSender = &gameRpcMessageSender{}
	logger.InfoWithSprintf("[platform] init rank board service")

	payOrderService.InitService(nodeConfig.NodeId)
	logger.InfoWithSprintf("[platform] init pay order service")

	logger.InfoWithSprintf("[platform] Boot platform success !!!")
}

func GetEventBus() *eventService.EventBus {
	return eventBus
}

func GetServerInfoService() *gameServerInfoService.GameServerInfoService {
	return serverInfoService
}

func GetRouter() serviceInterface.RouterInterface {
	return router
}

func GetSessionManager() logicCommon.SessionManagerInterface {
	return gameSessionManager
}

func GetDispatcher() *dispatcherService.Dispatcher {
	return dispatcher
}

func GetMessageSender() logicCommon.MessageSenderInterface {
	return messageSender
}

func GetUnlockService() logicCommon.UnlockServiceInterface {
	return unlock
}

func GetActivityService() logicCommon.GameActivityServiceInterface {
	return activityInfoService
}

func GetPassService() logicCommon.PassServiceInterface {
	return passService
}

func GetGloryArenaService() *gloryArenaService.GameGloryArenaService {
	return gloryArenaMatchService
}

func GetRpcMessageSender() logicCommon.RpcMessageSenderInterface {
	return rpcMessageSender
}

func GetSceneService() *logicScene.SceneService {
	return sceneService
}
