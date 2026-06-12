package httpPlatform

import (
	"encoding/json"
	"net/http"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/dbPool"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/easyRpc"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/payService"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/service/webService"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var auditServerConfig *nodeConfig.AuditServerConfig                // 审核服务器配置
var dbPoolManager *dbPool.DBPoolManager                            // 数据库连接池
var httpServiceInstance *webService.HttpWebService                 // http服务
var serverInfoService *gameServerInfoService.GameServerInfoService // 游戏服务器信息
var payServiceImpl *payService.PaymentService

func BootHttpPlatform() {
	cfg := platform.BootBasicService()
	auditServerConfig = cfg.AuditServerConfig

	data, _ := json.MarshalIndent(cfg.HttpConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Boot http platform config:%s", string(data))
	httpServiceInstance = webService.NewHttpWebService(cfg.HttpConfig)
	if httpServiceInstance == nil {
		logger.ErrorWithZapFields("[platform] Init web service error")
		panic("[platform] Init web service error")
	}

	data, _ = json.MarshalIndent(cfg.ServerDBConfig, "", "  ")
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
	easyDB.SetGameDBPool(dbPoolManager)

	data, _ = json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init redis error", zap.Error(err))
		panic("[platform] Init redis error")
	}

	gameConfig.LoadAllConfig()
	logger.InfoWithSprintf("[platform] load all config success !!!")

	// 节点RPC服务
	hooker := &httpRpcHooker{}
	ServerNodeService.InitNodeService(hooker, nodeConfig.NodeId, enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, cfg.EtcdConfig, "")
	rpcController.InitRpcController(enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, &httpDispatcherService{}, &httpSessionManager{}, &httpMessageSender{})
	logger.InfoWithSprintf("[platform] init node proxy service")

	// 游戏服务器信息服务
	serverInfoService = gameServerInfoService.NewGameServerInfoService()
	logger.InfoWithSprintf("[platform] init server info service")
	// 充值服务
	payServiceImpl = payService.NewPaymentService(cfg.PayConfig)
	logger.InfoWithSprintf("[platform] init recharge service")

	logger.InfoWithSprintf("[platform] Boot platform success !!!")
}

func RegisterHttpMessage(path string, handler http.HandlerFunc) {
	httpServiceInstance.RegisterRoutes(path, handler)
	logger.InfoWithSprintf("[platform] register http message: %s", path)
}

func StartHttpService() {
	logger.InfoWithSprintf("[platform] start http service")
	go httpServiceInstance.Start()
}

func GetServerInfoService() *gameServerInfoService.GameServerInfoService {
	return serverInfoService
}
func GetPayService() *payService.PaymentService {
	return payServiceImpl
}

type httpRpcHooker struct {
}

var _ logicCommon.RPCServiceHooker = (*httpRpcHooker)(nil)

func (h *httpRpcHooker) OnNodeConnect(id int32, nodeType enum.NodeType) {
	if nodeType == enum.NODE_TYPE_GATEWAY {
		ServerNodeService.InitGateRpcClient()
		rpcController.InitGatewayRpcClients()
	}
}

func (h *httpRpcHooker) OnNodeDisconnect(id int32, nodeType enum.NodeType, nodeAddress string) {
	easyRpc.CloseConnect(nodeAddress)
}

type httpDispatcherService struct {
}

var _ serviceInterface.DispatchInterface = (*httpDispatcherService)(nil)

func (d *httpDispatcherService) DispatchGameMessage(session serviceInterface.SessionInterface, msgID, msgType int32, msg proto.Message) {
	logger.ErrorBySprintf("[platform] DispatchGameMessage message error, msgId:%d, msgType:%d", msgID, msgType)
}

func (d *httpDispatcherService) DispatchInnerTask(task serviceInterface.InnerTaskInterface) {
	logger.ErrorBySprintf("[platform] DispatchInnerTask inner task error, taskId:%d", task.GetReqId())
}

func (d *httpDispatcherService) DispatchInnerTaskResp(task serviceInterface.InnerTaskInterface, respHandler func()) {
	logger.ErrorBySprintf("[platform] DispatchInnerTaskResp inner task response error, taskId:%d", task.GetReqId())
}

func (d *httpDispatcherService) DispatchInnerMessageTask(reqType enum.InnerMessageType, msgId enum.InnerMessageId, reqId int64, parameter any, respType enum.InnerMessageType, respId int64, respCallback serviceInterface.InnerTaskResult) {
	logger.ErrorBySprintf("[platform] DispatchInnerMessageTask inner task error, reqType:%d, reqId:%d, respType:%d, respId:%d", reqType, reqId, respType, respId)
}

type httpSessionManager struct {
}

var _ logicCommon.SessionManagerInterface = (*httpSessionManager)(nil)

func (h *httpSessionManager) GetPlayerBasicInfoBySessionId(sessionId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[platform] GetPlayerBasicInfoBySessionId error, sessionId:%d", sessionId)
	return nil
}

func (h *httpSessionManager) GetPlayerBasicInfoByUserId(userId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[platform] GetPlayerBasicInfoByUserId error, userId:%d", userId)
	return nil
}

func (h *httpSessionManager) GetPlayerBasicInfoByAccount(account string) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[platform] GetPlayerBasicInfoByAccount error, account:%s", account)
	return nil
}

type httpMessageSender struct {
}

var _ logicCommon.MessageSenderInterface = (*httpMessageSender)(nil)

func (h httpMessageSender) SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message) {
	logger.ErrorBySprintf("[platform] SendMessage error, msgId:%d", msgId)
}

func (h httpMessageSender) SendMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message) {

}

func (h httpMessageSender) Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int64) {
	logger.ErrorBySprintf("[platform] Broadcast error, msgId:%d", msgId)
}

func (h httpMessageSender) SendErrorMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[platform] SendErrorMessage error, msgId:%d", msgId)
}

func (h httpMessageSender) CloseSessionWithError(user logicCommon.UserBaseInterface, resp pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[platform] CloseSessionWithError error, resp:%d", resp)
}

func GetAuditServerAddr() (string, error) {
	return auditServerConfig.Addr, nil
}
