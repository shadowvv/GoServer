package rpcController

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"google.golang.org/protobuf/proto"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var gateServerService *gameServerInfoService.GameServerInfoService

func InitGateRpc(serverInfoService *gameServerInfoService.GameServerInfoService, activity logicCommon.GameActivityServiceInterface) {
	gateServerService = serverInfoService
	activityService = activity
}

func RegisterGatewayRpcServer(s *grpc.Server) {
	rpcPb.RegisterGateServiceServer(s, &GatewayRpcServer{})
}

func SendMessageToGame(nodeId int32, msg *rpcPb.ForwardGameMessage) error {
	gameClientMu.RLock()
	client := gameClients[nodeId]
	gameClientMu.RUnlock()

	if client == nil {
		InitGameRpcClients(nodeId)
		gameClientMu.RLock()
		client = gameClients[nodeId]
		gameClientMu.RUnlock()
		if client == nil {
			logger.ErrorBySprintf("[rpc] send message to game error: nodeId:%d,sessionId:%d,msg:%+v", nodeId, msg.SessionId, msg)
			return fmt.Errorf("[rpc] send message to game error: nodeId:%d,sessionId:%d,msg:%+v", nodeId, msg.SessionId, msg)
		}
	}
	err := client.SendMessage(msg.SessionId, msg)
	if err != nil {
		InitGameRpcClients(nodeId)
		gameClientMu.RLock()
		retryClient := gameClients[nodeId]
		gameClientMu.RUnlock()
		if retryClient == nil {
			logger.ErrorBySprintf("[rpc] send message to game retry client nil: nodeId:%d,sessionId:%d,msg:%+v", nodeId, msg.SessionId, msg)
			return fmt.Errorf("[rpc] send message to game retry client nil: nodeId:%d,sessionId:%d,msg:%+v", nodeId, msg.SessionId, msg)
		}
		err = retryClient.SendMessage(msg.SessionId, msg)
		if err != nil {
			logger.ErrorBySprintf("[rpc] send message to game error: nodeId:%d,sessionId:%d,msg:%+v,err:%v", nodeId, msg.SessionId, msg, err)
			return err
		}
	}
	return nil
}

func OnReceiveBackwardClientMessage(resp *rpcPb.BackwardClientMessage) {
	if resp == nil {
		return
	}
	if resp.BroadcastType != 0 {
		messageSender.Broadcast(pb.MESSAGE_ID(resp.MsgId), resp, enum.BroadcastType(resp.BroadcastType), resp.BroadcastId)
		return
	}
	user := sessionManager.GetPlayerBasicInfoBySessionId(resp.GetSessionId())
	if user == nil {
		logger.ErrorBySprintf("[rpc] send message to game error user == nil: sessionId:%d,msgId:%d", resp.GetSessionId(), resp.MsgId)
		return
	}
	if user.GetSession() == nil {
		logger.ErrorBySprintf("[rpc] send message to game error user.GetSession() == nil: sessionId:%d,msgId:%d,userId:%d", resp.GetSessionId(), resp.MsgId, user.GetUserId())
		return
	}
	if user.GetSession().GetID() != resp.GetSessionId() {
		logger.ErrorBySprintf("[rpc] send message to game error user.GetSession().GetID() != resp.GetSessionId(): sessionId:%d,msgId:%d,userId:%d", resp.GetSessionId(), resp.MsgId, user.GetUserId())
		return
	}
	if resp.CloseSession {
		user.GetSession().SendAndClose(resp.MsgId, resp)
	} else {
		user.GetSession().Send(resp.MsgId, resp)
	}
}

func BroadcastOperationToGameNode(operation rpcPb.RPC_SERVER_OPERATION) {
	client := ServerNodeService.GetAllGameClient()
	for _, c := range client {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			_, err := c.NotifyServerOperationHandler(ctx, &rpcPb.NotifyOperationMessage{
				Operation: operation,
			})
			if err != nil {
				logger.ErrorBySprintf("BroadcastOperationToGameNode error: %v", err)
				return
			}
		}()
	}
}

type GatewayRpcServer struct {
	rpcPb.UnimplementedGateServiceServer
}

func (g *GatewayRpcServer) SayHello(ctx context.Context, req *rpcPb.HelloReq) (*rpcPb.HelloResp, error) {
	logger.InfoWithSprintf("[gate] SayHello nodeInfo:%+v", req)
	return &rpcPb.HelloResp{}, nil
}

func (g *GatewayRpcServer) DeliverRechargeItem(ctx context.Context, req *rpcPb.DeliverRechargeItemReq) (*rpcPb.EmptyResp, error) {
	playerInfo := sessionManager.GetPlayerBasicInfoByUserId(req.UserId)
	if playerInfo == nil {
		g.deliverFailed(req)
		logger.ErrorBySprintf("[recharge] DeliverRechargeItem error: req:%v player is not online", req)
		return &rpcPb.EmptyResp{}, fmt.Errorf("[gateway] DeliverRechargeItem error: account:%s,serverId:%d", req.Account, req.ServerId)
	}
	if playerInfo.GetUserAccount() != req.Account || playerInfo.GetUserServerId() != req.ServerId {
		g.deliverFailed(req)
		logger.ErrorBySprintf("[recharge] DeliverRechargeItem error: req:%v", req)
		return &rpcPb.EmptyResp{}, fmt.Errorf("[gateway] DeliverRechargeItem error: account:%s,serverId:%d", req.Account, req.ServerId)
	}
	client, err := ServerNodeService.GetGameClientWithId(playerInfo.GetNodeId())
	if err != nil {
		g.deliverFailed(req)
		logger.ErrorBySprintf("[recharge] DeliverRechargeItem error: account:%s,serverId:%d,nodeId:%d,err:%v", req.Account, req.ServerId, playerInfo.GetNodeId(), err)
		return &rpcPb.EmptyResp{}, fmt.Errorf("[gateway] DeliverRechargeItem error: account:%s,serverId:%d,nodeId:%d,err:%v", req.Account, req.ServerId, playerInfo.GetNodeId(), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = client.DeliverRechargeItem(ctx, req)
	if err != nil {
		g.deliverFailed(req)
		logger.ErrorBySprintf("[recharge] DeliverRechargeItem error: account:%s,serverId:%d,nodeId:%d,err:%v", req.Account, req.ServerId, playerInfo.GetNodeId(), err)
		return &rpcPb.EmptyResp{}, fmt.Errorf("[gateway] DeliverRechargeItem error: account:%s,serverId:%d,nodeId:%d,err:%v", req.Account, req.ServerId, playerInfo.GetNodeId(), err)
	}
	logger.InfoWithSprintf("[recharge] DeliverRechargeItem success: account:%s,serverId:%d,nodeId:%d", req.Account, req.ServerId, playerInfo.GetNodeId())
	return &rpcPb.EmptyResp{}, nil
}

func (g *GatewayRpcServer) deliverFailed(req *rpcPb.DeliverRechargeItemReq) {
	orderEntity, err := easyDB.GetServerEntityByWhere[model.RechargeOrderEntity](map[string]interface{}{"order_id": req.OrderId})
	if orderEntity == nil || err != nil {
		logger.ErrorBySprintf("[recharge] DeliverFailed error: req:%v", req)
		return
	}
	orderEntity.Status = int32(enum.RECHARGE_ORDER_STATUS_FAILED)
	err = easyDB.UpdateServerEntity[model.RechargeOrderEntity](orderEntity, map[string]interface{}{"status": orderEntity.Status})
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[recharge] save order db error: %v,order:%v", err, orderEntity))
		return
	}
}

func (g *GatewayRpcServer) NotifyServerOperationHandler(ctx context.Context, req *rpcPb.NotifyOperationMessage) (*rpcPb.EmptyResp, error) {
	switch req.Operation {
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_SERVER_INFO:
		gateServerService.ReloadServerInfo()
		clients := ServerNodeService.GetAllGameClient()
		for nodeId, client := range clients {
			_, err := client.NotifyServerOperationHandler(context.Background(), req)
			if err != nil {
				logger.ErrorBySprintf("[gateway] NotifyServerOperationHandler req:%v error: %v,nodeId：%d", req, err, nodeId)
				continue
			}
		}
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_PLAYER_OUT:
		gatewaySessionManager := sessionManager.(*logicSessionManager.GatewaySessionManager)
		gatewaySessionManager.KickOutPlayer(req.OperationParam, func(player logicCommon.UserBaseInterface) {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
			messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PLAYER_IS_KICK_OUT)
		})
		nodeId, err := ServerNodeService.GetGameNodeIdByUserId(req.OperationParam)
		if err != nil {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d,err:%v]", req.OperationParam, err)
			return &rpcPb.EmptyResp{}, nil
		}
		client, err := ServerNodeService.GetGameClientWithId(nodeId)
		if err != nil {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d,err:%v]", req.OperationParam, err)
			return &rpcPb.EmptyResp{}, nil
		}
		_, err = client.NotifyServerOperationHandler(context.Background(), req)
		if err != nil {
			logger.ErrorBySprintf("[gateway] NotifyServerOperationHandler req:%v error: %v,nodeId：%d", req, err, nodeId)
		}

	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_SERVER_PLAYER_OFFLINE:
		gatewaySessionManager := sessionManager.(*logicSessionManager.GatewaySessionManager)
		gatewaySessionManager.KickOutServerPlayer(int32(req.OperationParam), func(player logicCommon.UserBaseInterface) {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
			messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PLAYER_IS_KICK_OUT)
		})
		broadcastToGameNode(req)
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_ALL_PLAYER_OFFLINE:
		gatewaySessionManager := sessionManager.(*logicSessionManager.GatewaySessionManager)
		gatewaySessionManager.KickOutAllPlayer(func(player logicCommon.UserBaseInterface) {
			logger.ErrorBySprintf("[platform] kick out node player Id:%d", player.GetUserId())
			messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PLAYER_IS_KICK_OUT)
		})
		broadcastToGameNode(req)
	case rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY_CONFIG:
		activityService.Reload()
	default:
		broadcastToGameNode(req)
	}
	return &rpcPb.EmptyResp{}, nil
}

func broadcastToGameNode(req *rpcPb.NotifyOperationMessage) {
	clients := ServerNodeService.GetAllGameClient()
	for nodeId, client := range clients {
		_, err := client.NotifyServerOperationHandler(context.Background(), req)
		if err != nil {
			logger.ErrorBySprintf("[gateway] NotifyServerOperationHandler req:%v error: %v,nodeId：%d", req, err, nodeId)
			continue
		}
	}
}

func (g *GatewayRpcServer) NotifyAllianceChange(ctx context.Context, req *rpcPb.NotifyAllianceChangeReq) (*rpcPb.EmptyResp, error) {
	gatewaySessionManager := sessionManager.(*logicSessionManager.GatewaySessionManager)
	player := gatewaySessionManager.GetPlayerByUserId(req.UserId)
	if player == nil {
		return &rpcPb.EmptyResp{}, nil
	}
	if player.GetSession() == nil {
		return &rpcPb.EmptyResp{}, nil
	}
	resp := &pb.PushAllianceChange{
		Oper:       pb.ALLIANCE_CHANGE_OPER(req.ChangeOper),
		AllianceId: req.AllianceId,
	}
	if resp.Oper == pb.ALLIANCE_CHANGE_OPER_ITEM_CHANGE {
		for _, v := range req.GetItems() {
			resp.ItemList = append(resp.ItemList, &pb.ItemBasicInfo{
				ItemId: v.ItemId,
				Count:  v.Count,
			})
		}
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		logger.ErrorBySprintf("[platform] Send error message error")
		return &rpcPb.EmptyResp{}, err
	}
	backMessage := &rpcPb.BackwardClientMessage{
		SessionId: player.GetSession().GetID(),
		MsgId:     int32(pb.MESSAGE_ID_PUSH_ALLIANCE_CHANGE),
		Payload:   data,
	}
	player.GetSession().Send(int32(pb.MESSAGE_ID_PUSH_ALLIANCE_CHANGE), backMessage)
	return &rpcPb.EmptyResp{}, nil
}
