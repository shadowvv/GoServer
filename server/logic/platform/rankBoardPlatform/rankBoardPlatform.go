package rankBoardPlatform

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
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

type rankBoardRpcHooker struct {
}

var _ logicCommon.RPCServiceHooker = (*rankBoardRpcHooker)(nil)

func (r *rankBoardRpcHooker) OnNodeConnect(id int32, nodeType enum.NodeType) {
	if nodeType == enum.NODE_TYPE_GAME {
		ServerNodeService.InitGameRpcClient(id)
	}
}

func (r *rankBoardRpcHooker) OnNodeDisconnect(id int32, nodeType enum.NodeType, nodeAddress string) {
	easyRpc.CloseConnect(nodeAddress)
	if nodeType == enum.NODE_TYPE_GAME {
		logger.InfoWithSprintf("[platform] node disconnect, nodeId:%d, nodeType:%s", id, nodeType)
	}
}

type rankBoardSessionManager struct {
}

var _ logicCommon.SessionManagerInterface = (*rankBoardSessionManager)(nil)

func (r rankBoardSessionManager) GetPlayerBasicInfoBySessionId(sessionId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[platform] GetPlayerBasicInfoBySessionId error, sessionId:%d", sessionId)
	return nil
}

func (r rankBoardSessionManager) GetPlayerBasicInfoByUserId(userId int64) logicCommon.UserBaseInterface {
	logger.ErrorBySprintf("[platform] GetPlayerBasicInfoByUserId error, userId:%d", userId)
	return nil
}

type rankBoardMessageSender struct {
}

var _ logicCommon.MessageSenderInterface = (*rankBoardMessageSender)(nil)

func (r rankBoardMessageSender) SendMessageByPlayerId(playerId int64, msgId pb.MESSAGE_ID, message proto.Message) {
	logger.ErrorBySprintf("[platform] SendMessage error, msgId:%d, playerId:%v", msgId, playerId)
}

func (r rankBoardMessageSender) SendMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, message proto.Message) {
	logger.ErrorBySprintf("[platform] SendMessage error, msgId:%d, player:%v", msgId, player)
}

func (r rankBoardMessageSender) Broadcast(msgId pb.MESSAGE_ID, msg proto.Message, broadcastType enum.BroadcastType, typeId int32) {
	logger.ErrorBySprintf("[platform] Broadcast error, msgId:%d, broadcastType:%d, typeId:%d", msgId, broadcastType, typeId)
}

func (r rankBoardMessageSender) SendErrorMessage(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[platform] SendErrorMessage error, msgId:%d, player:%v, errorCode:%d", msgId, player, errorCode)
}

func (r rankBoardMessageSender) CloseSessionWithError(player logicCommon.UserBaseInterface, msgId pb.MESSAGE_ID, errorCode pb.ERROR_CODE) {
	logger.ErrorBySprintf("[platform] CloseSessionWithError error, msgId:%d, player:%v, errorCode:%d", msgId, player, errorCode)
}

var mailIdGenerator *tool.IdGenerator
var dbPoolManager *dbPool.DBPoolManager
var router serviceInterface.RouterInterface
var dispatcher *dispatcherService.Dispatcher
var sessionManager logicCommon.SessionManagerInterface
var messageSender logicCommon.MessageSenderInterface
var serverInfoService *gameServerInfoService.GameServerInfoService

func BootRankBoardPlatform() {
	cfg := platform.BootBasicService()

	rankDB, err := dbService.InitMySQL(cfg.RankDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetRankDB(rankDB)
	gameDB, err := dbService.InitMySQL(cfg.GameDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init game db error", zap.Error(err))
		panic("[platform] Init game db error")
	}
	dbPoolManager = dbPool.NewDBPoolManager(gameDB)
	easyDB.SetGameDBPool(dbPoolManager)
	serverDB, err := dbService.InitMySQL(cfg.ServerDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetServerDB(serverDB)
	// 初始化redis
	data, _ := json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init redis error", zap.Error(err))
		panic("[platform] Init redis error")
	}

	sessionManager = &rankBoardSessionManager{}

	dispatcher = dispatcherService.NewDispatcher(router, sessionManager, tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_INNER_MESSAGE)), enum.NodeType(nodeConfig.NodeType), cfg.RunConfig.MessageProcessConfig)
	router = logicRouter.NewGameRouter(dispatcher)
	logger.InfoWithSprintf("[platform] init message dispatcher")

	messageSender = &rankBoardMessageSender{}
	rpcController.InitRankProxy(logicCodec.NewGameCodec(), router)
	// 节点RPC服务
	ServerNodeService.InitNodeService(&rankBoardRpcHooker{}, nodeConfig.NodeId, enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, cfg.EtcdConfig, "", rpcController.RegisterRankBoardRpcServer)
	rpcController.InitRpcController(enum.NodeType(nodeConfig.NodeType), cfg.RpcConfig, dispatcher, sessionManager, messageSender)

	gameConfig.LoadAllConfig()
	serverInfoService = gameServerInfoService.NewGameServerInfoService()
	logger.InfoWithSprintf("[platform] init server info service")
	logger.InfoWithSprintf("[platform] init node proxy service")

	mailIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_RANK_BOARD_MAIL))

	logger.InfoWithSprintf("[platform] Boot platform success !!!")
}

func RegisterRankBoardMessageHandler(messageId int32, h logicCommon.RankBoardMessageHandler) {
	dispatcher.RegisterRankBoardMessageHandler(messageId, h)
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

func GetServerInfoService() *gameServerInfoService.GameServerInfoService {
	return serverInfoService
}

func SendErrorMessageBySession(session serviceInterface.SessionInterface, messageId rpcPb.RPC_MESSAGE_ID, errorCode pb.ERROR_CODE) {
	rankBoardSession := session.(*logicSessionManager.RankBoardSession)
	if rankBoardSession == nil {
		return
	}
	rankBoardSession.ErrorCode = int32(errorCode)
	session.Send(int32(messageId), &rpcPb.EmptyResp{})
}

func SendMessageBySession(session serviceInterface.SessionInterface, messageId rpcPb.RPC_MESSAGE_ID, msg proto.Message) {
	session.Send(int32(messageId), msg)
}

func SendRankBoardRewardMail(mailTemplateId int32, playerId int64, items []*gameConfig.ItemConfig, rank int32) {
	cfg := gameConfig.GetMailContentCfg(mailTemplateId)
	if cfg == nil {
		logger.ErrorBySprintf("mail template not found: %v,playerId:%d", mailTemplateId, playerId)
		return
	}
	expireTime := int64(0)
	if cfg.MailExpTime > 0 {
		expireTime = tool.UnixNow() + int64(cfg.MailExpTime)*3600
	}
	mailID := mailIdGenerator.NextId()

	rankString := strconv.Itoa(int(rank))
	mailItems := make([]*mail.MailAttachmentItem, 0)
	for _, item := range items {
		mailItems = append(mailItems, &mail.MailAttachmentItem{
			ID:   item.ID,
			Num:  int32(item.Num),
			Type: int32(mail.AttachmentItemTypeItem),
		})
	}
	playerMail := &mail.Mail{
		MailID:        mailID,
		UserID:        playerId,
		MailType:      cfg.MailType,
		Title:         strconv.FormatInt(int64(cfg.MailTitle), 10),
		Content:       strconv.FormatInt(int64(cfg.MailWords), 10),
		ContentParams: []string{rankString},
		SenderID:      0,
		SenderName:    strconv.FormatInt(int64(cfg.SendName), 10),
		TemplateID:    cfg.ID,
		Status:        mail.MailStatusUnread,
		IsConvenient:  cfg.IsConvenient,
		Items:         mailItems,
		ExpireTime:    expireTime,
		SendTime:      tool.UnixNow(),
	}

	entity := mail.MailToEntity(playerMail)
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		logger.ErrorBySprintf("create player mail error: %v,entity:%v", err, entity)
		return
	}

	// 写入 Redis 通知游戏服刷新该玩家邮件缓存
	if dbService.RDB != nil {
		_ = dbService.RDB.SAdd(context.Background(), enum.REDIS_MAIL_REFRESH_USERS, strconv.FormatInt(playerId, 10)).Err()
	}
}

func SendRankBoardAllianceRewardMail(mailTemplateId int32, serverId int32, allianceId int64, items []*gameConfig.ItemConfig) error {
	if allianceId <= 0 {
		return nil
	}
	cfg := gameConfig.GetMailContentCfg(mailTemplateId)
	if cfg == nil {
		logger.ErrorBySprintf("mail template not found: %v,allianceId:%d", mailTemplateId, allianceId)
		return nil
	}
	expireTime := int64(0)
	if cfg.MailExpTime > 0 {
		expireTime = tool.UnixNow() + int64(cfg.MailExpTime)*3600
	}
	serverMailID := mailIdGenerator.NextId()

	mailItems := make([]*mail.MailAttachmentItem, 0, len(items))
	for _, item := range items {
		mailItems = append(mailItems, &mail.MailAttachmentItem{
			ID:   item.ID,
			Num:  int32(item.Num),
			Type: int32(mail.AttachmentItemTypeItem),
		})
	}
	allianceMail := &mail.ServerMail{
		ServerMailID: serverMailID,
		MailType:     cfg.MailType,
		Title:        strconv.FormatInt(int64(cfg.MailTitle), 10),
		Content:      strconv.FormatInt(int64(cfg.MailWords), 10),
		TemplateID:   cfg.ID,
		ServerID:     serverId,
		AllianceID:   allianceId,
		UnlockList:   []int32{},
		IsConvenient: cfg.IsConvenient,
		Items:        mailItems,
		SendTime:     tool.UnixNow(),
		ExpireTime:   expireTime,
		Status:       mail.ServerMailStatusSent,
		CreatedBy:    "rank_board_settle",
	}

	entity := mail.ServerMailToEntity(allianceMail)
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		logger.ErrorBySprintf("create alliance rank reward mail error: %v,entity:%v", err, entity)
		return err
	}

	if dbService.RDB != nil {
		members, err := easyDB.GetPlayerEntitiesByWhere[model.AllianceMemberEntity](map[string]interface{}{"alliance_id": allianceId})
		if err != nil {
			logger.ErrorBySprintf("load alliance members for mail refresh failed, allianceId:%d, err:%v", allianceId, err)
			return nil
		}
		for _, member := range members {
			if member == nil || member.UserId <= 0 {
				continue
			}
			_ = dbService.RDB.SAdd(context.Background(), enum.REDIS_MAIL_REFRESH_USERS, strconv.FormatInt(member.UserId, 10)).Err()
		}
	}
	return nil
}
