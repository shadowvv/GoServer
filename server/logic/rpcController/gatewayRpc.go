package rpcController

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/gameServerInfoService"
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

func InitGateRpc(serverInfoService *gameServerInfoService.GameServerInfoService) {
	gateServerService = serverInfoService
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
		logger.ErrorBySprintf("[rpc] send message to game error user == nil: sessionId:%d,msg:%+v,user:%+v", resp.GetSessionId(), resp, user)
		return
	}
	if user.GetSession() == nil {
		logger.ErrorBySprintf("[rpc] send message to game error user.GetSession() == nil: sessionId:%d,msg:%+v,user:%+v", resp.GetSessionId(), resp, user)
		return
	}
	if user.GetSession().GetID() != resp.GetSessionId() {
		logger.ErrorBySprintf("[rpc] send message to game error user.GetSession().GetID() != resp.GetSessionId(): sessionId:%d,msg:%+v,user:%+v", resp.GetSessionId(), resp, user)
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
	default:
		clients := ServerNodeService.GetAllGameClient()
		for nodeId, client := range clients {
			_, err := client.NotifyServerOperationHandler(context.Background(), req)
			if err != nil {
				logger.ErrorBySprintf("[gateway] NotifyServerOperationHandler req:%v error: %v,nodeId：%d", req, err, nodeId)
				continue
			}
		}
	}
	return &rpcPb.EmptyResp{}, nil
}

func (g *GatewayRpcServer) GmKickPlayer(ctx context.Context, req *rpcPb.GmKickPlayerReq) (*rpcPb.EmptyResp, error) {
	gatewaySessionManager := sessionManager.(*logicSessionManager.GatewaySessionManager)
	gatewaySessionManager.KickOutPlayer(req.UserId, req.ServerId)
	return &rpcPb.EmptyResp{}, nil
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
