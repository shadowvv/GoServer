package rpcController

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/drop/GoServer/server/logic/gameServerInfoService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var gameSessionAcceptor serviceInterface.AcceptorInterface
var gameCodec serviceInterface.CodecInterface
var gameRouter serviceInterface.RouterInterface
var activityService logicCommon.GameActivityServiceInterface
var serverInfoService *gameServerInfoService.GameServerInfoService
var playerOfflineTimeout time.Duration

var gameRpcSenderMu sync.RWMutex
var gameRpcSender []logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]
var rankBoardClient *EasyRpcClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage]
var socialClient *EasyRpcClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage]

const rpcReplyTimeout = time.Second

var rpcTimeoutMu sync.Mutex
var rpcReplyTimeouts = make(map[string]*time.Timer)

func makeRpcTimeoutKey(channel string, userID int64, backMessageID int32) string {
	return fmt.Sprintf("%s:%d:%d", channel, userID, backMessageID)
}

func registerRpcReplyTimeout(channel string, userID int64, backMessageID int32) {
	if backMessageID <= 0 {
		return
	}
	key := makeRpcTimeoutKey(channel, userID, backMessageID)
	timer := time.AfterFunc(rpcReplyTimeout, func() {
		clearRpcReplyTimeout(channel, userID, backMessageID)
		player := sessionManager.GetPlayerBasicInfoByUserId(userID)
		if player == nil {
			return
		}
		pModel, ok := player.(*model.PlayerModel)
		if !ok || pModel == nil {
			return
		}
		messageSender.SendErrorMessage(pModel, pb.MESSAGE_ID(backMessageID), pb.ERROR_CODE_SYSTEM_IS_BUSY)
	})

	rpcTimeoutMu.Lock()
	if old := rpcReplyTimeouts[key]; old != nil {
		old.Stop()
	}
	rpcReplyTimeouts[key] = timer
	rpcTimeoutMu.Unlock()
}

func clearRpcReplyTimeout(channel string, userID int64, backMessageID int32) {
	if backMessageID <= 0 {
		return
	}
	key := makeRpcTimeoutKey(channel, userID, backMessageID)
	rpcTimeoutMu.Lock()
	timer := rpcReplyTimeouts[key]
	delete(rpcReplyTimeouts, key)
	rpcTimeoutMu.Unlock()
	if timer != nil {
		timer.Stop()
	}
}

func addGameRpcSender(sender logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]) {
	if sender == nil {
		return
	}
	gameRpcSenderMu.Lock()
	gameRpcSender = append(gameRpcSender, sender)
	gameRpcSenderMu.Unlock()
}

func removeGameRpcSender(sender logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]) {
	if sender == nil {
		return
	}
	gameRpcSenderMu.Lock()
	for i, s := range gameRpcSender {
		if s == sender {
			gameRpcSender = append(gameRpcSender[:i], gameRpcSender[i+1:]...)
			break
		}
	}
	gameRpcSenderMu.Unlock()
}

func randomGameRpcSender() logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage] {
	gameRpcSenderMu.RLock()
	defer gameRpcSenderMu.RUnlock()
	if len(gameRpcSender) == 0 {
		return nil
	}
	index := tool.RandInt(0, len(gameRpcSender)-1)
	return gameRpcSender[index]
}

func RegisterGameRpcServer(s *grpc.Server) {
	rpcPb.RegisterGameServiceServer(s, &GameRpcServer{})
}

func InitGameRpc(acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface, activity logicCommon.GameActivityServiceInterface, serverInfo *gameServerInfoService.GameServerInfoService, offlineTimeout time.Duration) {
	gameSessionAcceptor = acceptorInterface
	gameCodec = codec
	gameRouter = router
	activityService = activity
	serverInfoService = serverInfo
	playerOfflineTimeout = offlineTimeout
}

func OnReceiveBackwardRankBoardMessage(message *rpcPb.BackwardRankBoardMessage) {
	clearRpcReplyTimeout("rank", message.UserId, message.BackMessageId)
	player := sessionManager.GetPlayerBasicInfoByUserId(message.UserId)
	if player == nil {
		logger.ErrorBySprintf("[rpc] unknown userId:%d", message.UserId)
		return
	}
	pModel := player.(*model.PlayerModel)
	if pModel == nil {
		logger.ErrorBySprintf("[rpc] unknown userId:%d", message.UserId)
		return
	}
	if message.ErrorCode != int32(pb.ERROR_CODE_SUCCESS) {
		messageSender.SendErrorMessage(pModel, pb.MESSAGE_ID(message.BackMessageId), pb.ERROR_CODE(message.ErrorCode))
		return
	}
	// ------------------ 路由 & 反序列化 ------------------
	gameMessage := gameRouter.GetMessage(int32(message.MsgId))
	if gameMessage == nil {
		logger.ErrorBySprintf("[net] unknown msgId:%d", message.MsgId)
		return
	}

	if err := gameCodec.Unmarshal(message.Payload, gameMessage); err != nil {
		logger.ErrorBySprintf("[net] unmarshal failed msgId:%d err:%v", message.MsgId, err)
		return
	}
	// ------------------ 投递到逻辑层 ------------------
	gameRouter.Dispatch(player.GetSession(), int32(message.MsgId), gameMessage)
}

func OnReceiveBackwardSocialMessage(message *rpcPb.BackwardSocialMessage) {
	clearRpcReplyTimeout("social", message.UserId, message.BackMessageId)
	player := sessionManager.GetPlayerBasicInfoByUserId(message.UserId)
	if player == nil {
		logger.ErrorBySprintf("[rpc] social unknown userId:%d", message.UserId)
		return
	}
	pModel := player.(*model.PlayerModel)
	if pModel == nil {
		logger.ErrorBySprintf("[rpc] social invalid player userId:%d", message.UserId)
		return
	}
	if message.ErrorCode != int32(pb.ERROR_CODE_SUCCESS) {
		if message.BackMessageId > 0 {
			messageSender.SendErrorMessage(pModel, pb.MESSAGE_ID(message.BackMessageId), pb.ERROR_CODE(message.ErrorCode))
		}
		return
	}
	gameMessage := gameRouter.GetMessage(int32(message.MsgId))
	if gameMessage == nil {
		logger.ErrorBySprintf("[rpc] social unknown msgId:%d", message.MsgId)
		return
	}
	if err := gameCodec.Unmarshal(message.Payload, gameMessage); err != nil {
		logger.ErrorBySprintf("[rpc] social unmarshal failed msgId:%d err:%v", message.MsgId, err)
		return
	}
	gameRouter.Dispatch(player.GetSession(), int32(message.MsgId), gameMessage)
}

func SendMessageToRankBoard(userId int64, rankId string, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error {
	return sendMessageToRankBoard(userId, rankId, backMessageId, msgId, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ID_NONE, req)
}

func SendMessageToRankBoardWithRespMsgId(userId int64, rankId string, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, respMsgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error {
	return sendMessageToRankBoard(userId, rankId, backMessageId, msgId, respMsgId, req)
}

func sendMessageToRankBoard(userId int64, rankId string, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, respMsgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error {
	data, err := gameCodec.Marshal(int32(msgId), req)
	if err != nil {
		logger.ErrorBySprintf("[rpc] marshal rankBoard message error:%+v", err)
		return err
	}
	msg := &rpcPb.ForwardRankBoardMessage{
		UserId:        userId,
		MsgId:         msgId,
		BackMessageId: backMessageId,
		RankId:        rankId,
		Payload:       data,
		RespMsgId:     respMsgId,
	}

	id := tool.Hash(msg.RankId)
	if rankBoardClient == nil {
		logger.ErrorBySprintf("[rpc] unknown rankBoardClient")
		return fmt.Errorf("[rpc] unknown rankBoardClient")
	}
	registerRpcReplyTimeout("rank", userId, backMessageId)
	defer func() {
		if err != nil {
			clearRpcReplyTimeout("rank", userId, backMessageId)
		}
	}()
	err = rankBoardClient.SendMessage(int64(id), msg)
	if err != nil {
		InitRankBoardRpcClients()
		err = rankBoardClient.SendMessage(int64(id), msg)
		if err != nil {
			logger.ErrorBySprintf("[rpc] send rankBoard message error:%+v", err)
			return err
		}
	}
	return nil
}

const BLOCK_SOCIAL = true

func IsSocialRpcEnabled() bool {
	return !BLOCK_SOCIAL
}

func SendMessageToSocial(userId int64, allianceId int64, backMessageId int32, msgId rpcPb.RPC_MESSAGE_ID, req proto.Message) error {
	if BLOCK_SOCIAL {
		return nil
	}
	data, err := gameCodec.Marshal(int32(msgId), req)
	if err != nil {
		logger.ErrorBySprintf("[rpc] marshal social message error:%+v", err)
		return err
	}
	effectiveAllianceId := allianceId
	if effectiveAllianceId <= 0 {
		effectiveAllianceId = extractSocialAllianceID(req)
	}

	msg := &rpcPb.ForwardSocialMessage{
		UserId:        userId,
		MsgId:         msgId,
		BackMessageId: backMessageId,
		AllianceId:    effectiveAllianceId,
		Payload:       data,
	}
	if socialClient == nil {
		logger.ErrorBySprintf("[rpc] social client is nil")
		return fmt.Errorf("[rpc] social client is nil")
	}
	registerRpcReplyTimeout("social", userId, backMessageId)
	defer func() {
		if err != nil {
			clearRpcReplyTimeout("social", userId, backMessageId)
		}
	}()
	shardKey := effectiveAllianceId
	if shardKey < 0 {
		shardKey = -shardKey
	}
	err = socialClient.SendMessage(shardKey, msg)
	if err != nil {
		InitSocialRpcClients()
		err = socialClient.SendMessage(shardKey, msg)
		if err != nil {
			logger.ErrorBySprintf("[rpc] send social message error:%+v", err)
			return err
		}
	}
	return nil
}

func extractSocialAllianceID(req proto.Message) int64 {
	switch r := req.(type) {
	case *rpcPb.ChangeAllianceBasicInfoReq:
		return r.GetAllianceId()
	case *rpcPb.GetAllianceInfoReq:
		return r.GetAllianceId()
	case *rpcPb.ApplyAllianceReq:
		return r.GetAllianceId()
	case *rpcPb.KickAllianceMemberReq:
		return r.GetAllianceId()
	case *rpcPb.QuitAllianceReq:
		return r.GetAllianceId()
	case *rpcPb.ChangeMemberPositionReq:
		return r.GetAllianceId()
	default:
		return 0
	}
}

func BroadcastMessageToGateway(msg *rpcPb.BackwardClientMessage) error {
	sender := randomGameRpcSender()
	if sender == nil {
		logger.ErrorBySprintf("[rpc] no available gateway stream sender")
		return fmt.Errorf("[rpc] no available gateway stream sender")
	}
	return sender.Send(msg)
}

type GameRpcServer struct {
	rpcPb.UnimplementedGameServiceServer
}

func (g *GameRpcServer) SayHello(ctx context.Context, req *rpcPb.HelloReq) (*rpcPb.HelloResp, error) {
	logger.InfoWithSprintf("[gate] SayHello nodeInfo:%+v", req)
	return &rpcPb.HelloResp{}, nil
}

func (g *GameRpcServer) NotifyServerOperationHandler(ctx context.Context, req *rpcPb.NotifyOperationMessage) (*rpcPb.EmptyResp, error) {
	switch req.Operation {
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY_CONFIG:
		activityService.Reload()
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY:
		activityService.Reload()
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_ANNOUNCE_INFO_UPDATE:
		serverInfoService.ReloadAnnounce()
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_SERVER_INFO:
		serverInfoService.ReloadServerInfo()
	}

	return &rpcPb.EmptyResp{}, nil
}

func (g *GameRpcServer) ForwardGameMessageHandler(stream grpc.BidiStreamingServer[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]) error {

	// 1️⃣ stream 生命周期 = gateway 连接生命周期
	ctx := stream.Context()
	rpcSender := NewGrpcSender[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage](stream, 100)
	if rpcSender == nil {
		return fmt.Errorf("[grpc][stream] init rpc sender failed")
	}
	addGameRpcSender(rpcSender)
	defer func() {
		removeGameRpcSender(rpcSender)
		rpcSender.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			// gateway 主动断开 / 超时
			err := ctx.Err()
			st, _ := status.FromError(err)

			logger.ErrorBySprintf("[grpc][stream] ctx done code=%s err=%v", st.Code(), err)
			return err

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				// 正常关闭
				logger.InfoWithSprintf("[net] gateway stream EOF")
				return nil
			}
			if err != nil {
				logger.ErrorWithZapFields(fmt.Sprintf("[net] recv failed: %v", err))
				return err
			}
			if msg == nil {
				continue
			}

			// ------------------ 解包 ------------------
			if len(msg.Payload) < 4 {
				logger.ErrorWithZapFields("[net] invalid payload length")
				continue
			}

			msgID := binary.BigEndian.Uint32(msg.Payload[:4])
			payload := msg.Payload[4:]

			// ------------------ Session 管理 ------------------
			gSession := gameSessionAcceptor.GetSessionById(msg.SessionId)
			var realSession *logicSessionManager.GameSession
			// Login：创建 session，并把 stream 绑定进去
			if msgID == uint32(pb.MESSAGE_ID_LOGIN_REQ) {
				if gSession == nil {
					realSession = logicSessionManager.NewGameSession(gameCodec, msg.SessionId, msg.PlayerId, gameSessionAcceptor, playerOfflineTimeout)
					realSession.Start()
					// ⭐ 关键：绑定 stream（用于 Send）
					realSession.BindSender(rpcSender)
					gSession = realSession
					gameSessionAcceptor.Accept(gSession)
				}
			}

			if gSession == nil {
				logger.ErrorBySprintf("[net] unknown sessionId:%d", msg.SessionId)
				continue
			}
			realSession = gSession.(*logicSessionManager.GameSession)
			if realSession == nil {
				logger.ErrorBySprintf("[net] unknown sessionId:%d", msg.SessionId)
				continue
			}

			// ------------------ 路由 & 反序列化 ------------------
			gameMessage := gameRouter.GetMessage(int32(msgID))
			if gameMessage == nil {
				logger.ErrorBySprintf("[net] unknown msgId:%d sessionId:%d", msgID, msg.SessionId)
				continue
			}

			if err := gameCodec.Unmarshal(payload, gameMessage); err != nil {
				logger.ErrorBySprintf("[net] unmarshal failed sessionId:%d msgId:%d err:%v", msg.SessionId, msgID, err)
				continue
			}
			// ------------------ 投递到逻辑层 ------------------
			gameRouter.Dispatch(realSession, int32(msgID), gameMessage)
			realSession.LastActiveTime.Store(tool.UnixNowMilli())
		}
	}
}

func (g *GameRpcServer) DeliverRechargeItem(ctx context.Context, req *rpcPb.DeliverRechargeItemReq) (*rpcPb.EmptyResp, error) {
	logger.InfoWithSprintf("[recharge] DeliverRechargeItem req:%+v", req)
	dispatcherService.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MSG_DELIVER_RECHARGE_ITEM, req.UserId, req, 0, 0, nil)
	return &rpcPb.EmptyResp{}, nil
}
