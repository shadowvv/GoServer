package socialPlatform

import (
	"encoding/json"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
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
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/easyRpc"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type socialRpcHooker struct {
}

var _ logicCommon.RPCServiceHooker = (*socialRpcHooker)(nil)

func (s *socialRpcHooker) OnNodeConnect(id int32, nodeType enum.NodeType) {
	if nodeType == enum.NODE_TYPE_RANKBOARD {
		ServerNodeService.InitRankBoardRpcClient()
		rpcController.InitRankBoardRpcClients()
	} else if nodeType == enum.NODE_TYPE_GAME {
		ServerNodeService.InitGameRpcClient(id)
	}
}

func (s *socialRpcHooker) OnNodeDisconnect(id int32, nodeType enum.NodeType, nodeAddress string) {
	easyRpc.CloseConnect(nodeAddress)
}

type socialSessionManager struct {
}

var _ logicCommon.SessionManagerInterface = (*socialSessionManager)(nil)

func (s socialSessionManager) GetPlayerBasicInfoBySessionId(sessionId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[socialPlatform] GetPlayerBasicInfoBySessionId not supported sessionId:%d", sessionId)
	return nil
}

func (s socialSessionManager) GetPlayerBasicInfoByUserId(userId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[socialPlatform] GetPlayerBasicInfoByUserId not supported userId:%d", userId)
	return nil
}

type socialMessageSender struct {
}

var _ logicCommon.MessageSenderInterface = (*socialMessageSender)(nil)

func (s socialMessageSender) SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message) {
	logger.ErrorBySprintf("[socialPlatform] SendMessage not supported msgId:%d playerId:%v", msgId, playerId)
}

func (s socialMessageSender) SendMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message) {
	logger.ErrorBySprintf("[socialPlatform] SendMessage not supported msgId:%d player:%v", msgId, player)
}

func (s socialMessageSender) Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int32) {
	logger.ErrorBySprintf("[socialPlatform] Broadcast not supported msgId:%d", msgId)
}

func (s socialMessageSender) SendErrorMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[socialPlatform] SendErrorMessage not supported msgId:%d errorCode:%d", msgId, errorCode)
}

func (s socialMessageSender) CloseSessionWithError(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[socialPlatform] CloseSessionWithError not supported msgId:%d errorCode:%d", msgId, errorCode)
}

var dbPoolManager *dbPool.DBPoolManager
var router serviceInterface.RouterInterface
var dispatcher *dispatcherService.Dispatcher
var sessionManager logicCommon.SessionManagerInterface
var messageSender logicCommon.MessageSenderInterface

func BootSocialPlatform() {
	cfg := platform.BootBasicService()

	gameDB, err := dbService.InitMySQL(cfg.GameDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[socialPlatform] init game db error", zap.Error(err))
		panic("[socialPlatform] init game db error")
	}
	dbPoolManager = dbPool.NewDBPoolManager(gameDB)
	easyDB.SetGameDBPool(dbPoolManager)

	data, _ := json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[socialPlatform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[socialPlatform] init redis error", zap.Error(err))
		panic("[socialPlatform] init redis error")
	}

	gameConfig.LoadAllConfig()

	sessionManager = &socialSessionManager{}
	dispatcher = dispatcherService.NewDispatcher(router, sessionManager, tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_INNER_MESSAGE)), enum.NodeType(nodeConfig.NodeType), cfg.RunConfig.MessageProcessConfig)
	router = logicRouter.NewGameRouter(dispatcher)
	messageSender = &socialMessageSender{}

	// Ensure rpc handler dependencies are ready before exposing grpc service / etcd registration.
	rpcController.InitSocialProxy(logicCodec.NewGameCodec(), router)
	ServerNodeService.InitNodeService(&socialRpcHooker{}, nodeConfig.NodeId, enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, cfg.EtcdConfig, "", rpcController.RegisterSocialRpcServer)
	rpcController.InitRpcController(enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, dispatcher, sessionManager, messageSender)

	logger.InfoWithSprintf("[socialPlatform] boot platform success")
}

func GetRouter() serviceInterface.RouterInterface {
	return router
}

func GetSessionManager() logicCommon.SessionManagerInterface {
	return sessionManager
}

func GetDispatcher() *dispatcherService.Dispatcher {
	return dispatcher
}

func GetMessageSender() logicCommon.MessageSenderInterface {
	return messageSender
}

func GetUnlockService() logicCommon.UnlockServiceInterface {
	return nil
}

func GetActivityService() logicCommon.GameActivityServiceInterface {
	return nil
}

func SendErrorMessageBySession(session serviceInterface.SessionInterface, messageId rpcPb.RPC_MESSAGE_ID, errorCode pb.ERROR_CODE) {
	socialSession := session.(*logicSessionManager.AllianceSession)
	if socialSession == nil {
		return
	}
	socialSession.ErrorCode = int32(errorCode)
	session.Send(int32(messageId), &rpcPb.EmptyResp{})
}

func SendMessageBySession(session serviceInterface.SessionInterface, messageId rpcPb.RPC_MESSAGE_ID, msg proto.Message) {
	session.Send(int32(messageId), msg)
}
