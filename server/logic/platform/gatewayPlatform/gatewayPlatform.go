package gatewayPlatform

import (
	"encoding/json"
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/activityService"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/dbPool"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/logicCodec"
	"github.com/drop/GoServer/server/logic/platform/logicRouter"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/easyRpc"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/netService"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type gatewayRpcHooker struct {
}

var _ logicCommon.RPCServiceHooker = (*gatewayRpcHooker)(nil)

func (g *gatewayRpcHooker) OnNodeConnect(id int32, nodeType enum.NodeType) {
	if nodeType == enum.NODE_TYPE_GAME {
		ServerNodeService.InitGameRpcClient(id)
		rpcController.InitGameRpcClients(id)
	}
}

func (g *gatewayRpcHooker) OnNodeDisconnect(nodeId int32, nodeType enum.NodeType, nodeAddress string) {
	easyRpc.CloseConnect(nodeAddress)
	if nodeType == enum.NODE_TYPE_GAME {
		gatewaySessionManager.KickOutNodePlayer(nodeId, func(player logicCommon.UserBaseInterface) {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
			messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PLAYER_IS_KICK_OUT)
		})
		rpcController.RemoveGatewayRpcGameClient(nodeId)
	}
}

type gatewayMessageSender struct {
}

var _ logicCommon.MessageSenderInterface = (*gatewayMessageSender)(nil)

func (g *gatewayMessageSender) SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message) {
	player := gatewaySessionManager.GetPlayerByUserId(playerId)
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

func (g *gatewayMessageSender) SendMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message) {
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

func (g *gatewayMessageSender) Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int64) {
	var playerMap map[int64]*logicCommon.GatewayPlayerInfo
	switch broadcastType {
	case enum.BROADCAST_TYPE_SERVER_ID:
		playerMap = gatewaySessionManager.GetPlayerSessionsByServerId(int32(typeId))
	case enum.BROADCAST_TYPE_ALLIANCE:
		ids := logicCommon.GetAllianceMemberId(typeId)
		if len(ids) == 0 {
			return
		}
		playerMap = gatewaySessionManager.GetPlayerSessionsByUserIds(ids)
	default:
	}
	if len(playerMap) == 0 {
		return
	}
	for _, v := range playerMap {
		v.Session.Send(int32(msgId), msg)
	}
}

func (g *gatewayMessageSender) SendErrorMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	errorMsg := &pb.MessageError{
		MsgId:     msgId,
		ErrorCode: errorCode,
	}
	data, err := proto.Marshal(errorMsg)
	if err != nil {
		logger.ErrorBySprintf("[platform] Send error message error")
		return
	}
	backMessage := &rpcPb.BackwardClientMessage{
		SessionId: player.GetSession().GetID(),
		MsgId:     int32(msgId),
		Payload:   data,
	}
	g.SendMessage(player, pb.MESSAGE_ID_MESSAGE_ERROR, backMessage)
}

func (g *gatewayMessageSender) CloseSessionWithError(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	errorMsg := &pb.MessageError{
		MsgId:     msgId,
		ErrorCode: errorCode,
	}
	data, err := proto.Marshal(errorMsg)
	if err != nil {
		logger.ErrorBySprintf("[platform] Send error message error")
		return
	}
	backMessage := &rpcPb.BackwardClientMessage{
		SessionId: player.GetSession().GetID(),
		MsgId:     int32(pb.MESSAGE_ID_MESSAGE_ERROR),
		Payload:   data,
	}
	if player.GetSession() == nil {
		return
	}
	if !player.GetSession().IsActive() {
		return
	}
	player.GetSession().SendAndClose(int32(pb.MESSAGE_ID_MESSAGE_ERROR), backMessage)
}

var dbPoolManager *dbPool.DBPoolManager                              // 数据库连接池
var gatewaySessionManager *logicSessionManager.GatewaySessionManager // 网关会话管理
var router serviceInterface.RouterInterface                          // 消息路由
var dispatcher *dispatcherService.Dispatcher                         // 消息分发
var serverInfoService *gameServerInfoService.GameServerInfoService   // 游戏服务器信息
var activityInfoService *activityService.GatewayActivityService      // 活动信息
var messageSender logicCommon.MessageSenderInterface                 // 消息发送
var unlock logicCommon.UnlockServiceInterface                        // 解锁服务

func BootGatewayPlatform() {
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
	// 初始化数据库连接池（gateway 只需要 player 类型的连接池用于读取玩家数据）
	for _, poolInfo := range cfg.RunConfig.DBPoolInfo {
		dbPoolManager.AddDBPool(enum.DBPoolType(poolInfo.PoolType), poolInfo.WorkerNum, poolInfo.WorkerTaskSize)
	}
	easyDB.SetGameDBPool(dbPoolManager)
	// 初始化redis
	data, _ = json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init redis error", zap.Error(err))
		panic("[platform] Init redis error")
	}

	// 网关会话,消息路由,分发
	gatewaySessionManager = logicSessionManager.NewGatewaySessionManager()
	logger.InfoWithSprintf("[platform] init session manager")
	dispatcher = dispatcherService.NewDispatcher(router, gatewaySessionManager, tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_INNER_MESSAGE)), enum.NodeType(nodeConfig.NodeType), cfg.RunConfig.MessageProcessConfig)
	logger.InfoWithSprintf("[platform] init dispatcher")
	router = logicRouter.NewGatewayRouter(dispatcher)
	logger.InfoWithSprintf("[platform] init router")

	// 网络服务
	err = initNetService(cfg.NetConfig, gatewaySessionManager, logicCodec.NewGatewayCodec(), router)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server error", zap.Error(err))
		panic("[platform] Init server error")
	}

	gameConfig.LoadAllConfig()
	// 游戏服务器信息服务
	serverInfoService = gameServerInfoService.NewGameServerInfoService()
	serverInfoService.ResetOnlinePlayerNum()

	unlock = unlockService.NewUnlockService(serverInfoService)
	activityInfoService = activityService.NewGatewayActivityService(nodeConfig.Env, serverInfoService, unlock)
	activityInfoService.StartService()
	logger.InfoWithSprintf("[platform] init activity info service")

	rpcController.InitGateRpc(serverInfoService, activityInfoService)
	logger.InfoWithSprintf("[platform] init server info service")

	messageSender = &gatewayMessageSender{}

	// 节点RPC服务
	hooker := &gatewayRpcHooker{}
	ServerNodeService.InitNodeService(hooker, nodeConfig.NodeId, enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, cfg.EtcdConfig, cfg.NetConfig.Address, rpcController.RegisterGatewayRpcServer)
	rpcController.InitRpcController(enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, dispatcher, gatewaySessionManager, messageSender)
	logger.InfoWithSprintf("[platform] init node proxy service")
	logger.InfoWithSprintf("[platform] Boot platform success !!!")

	gloryArenaService.NewGatewayGloryArenaStateService(serverInfoService).StartService()
	logger.InfoWithSprintf("[platform] init gateway glory arena state service")

	logger.InfoWithSprintf("[platform] init platform success !!")
}

// 初始化网络服务
func initNetService(config *netService.NetConfig, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) error {
	data, _ := json.MarshalIndent(config, "", "  ")
	logger.InfoWithSprintf("[platform] Init net service config:%s", string(data))
	server := netService.NewNetService(config, nodeConfig.NodeId, acceptorInterface, codec, router)
	go func() {
		err := server.Start()
		if err != nil {
			logger.ErrorWithZapFields("[platform] Start server error", zap.Error(err))
		}
	}()
	return nil
}

func GetSessionManager() *logicSessionManager.GatewaySessionManager {
	return gatewaySessionManager
}

func GetRouter() serviceInterface.RouterInterface {
	return router
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

func KickOutAllPlayer() {
	gatewaySessionManager.KickOutNodePlayer(1, func(player logicCommon.UserBaseInterface) {
		logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_SERVER_IS_MAINTAIN)
	})
	gatewaySessionManager.KickOutNodePlayer(2, func(player logicCommon.UserBaseInterface) {
		logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_SERVER_IS_MAINTAIN)
	})
}
