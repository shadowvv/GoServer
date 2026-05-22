package rpcController

import (
	"context"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/grpc"
)

var dispatcherService serviceInterface.DispatchInterface
var sessionManager logicCommon.SessionManagerInterface
var messageSender logicCommon.MessageSenderInterface
var config *ServerNodeService.RpcConfig

var gameClientMu sync.RWMutex
var gameClients = make(map[int32]*EasyRpcClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage])

func InitRpcController(nodeType enum.NodeType, rpcConfig *ServerNodeService.RpcConfig, dispatcher serviceInterface.DispatchInterface, sessionMgr logicCommon.SessionManagerInterface, sender logicCommon.MessageSenderInterface) {
	dispatcherService = dispatcher
	sessionManager = sessionMgr
	messageSender = sender
	config = rpcConfig

	switch nodeType {
	case enum.NODE_TYPE_HTTP:
		InitGatewayRpcClients()
	case enum.NODE_TYPE_GATEWAY:
		InitAllGameClients()
	case enum.NODE_TYPE_GAME:
		InitRankBoardRpcClients()
		InitSocialRpcClients()
		InitGatewayRpcClients()
	case enum.NODE_TYPE_RANKBOARD:
	case enum.NODE_TYPE_SOCIAL:
		InitRankBoardRpcClients()
	default:
		panic("[platform] init rpc client invalid node type")
	}
}

func InitGatewayRpcClients() {
	_, err := ServerNodeService.GetGatewayClient()
	if err != nil {
		logger.ErrorBySprintf("[rpc] init gateway node client error:%+v", err)
		return
	}
}

func InitGameRpcClients(nodeId int32) {
	gameClientMu.Lock()
	defer gameClientMu.Unlock()

	if old := gameClients[nodeId]; old != nil {
		old.Close()
	}

	c := NewGameNodeClient(
		nodeId,
		func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage], error) {
			client, err := ServerNodeService.GetGameClientWithId(nodeId)
			if err != nil {
				return nil, err
			}
			return client.ForwardGameMessageHandler(context.Background())
		},
		config.RpcStreamRoutineCount,
		config.RpcStreamBufferSize,
		OnReceiveBackwardClientMessage,
	)

	gameClients[nodeId] = c
	logger.InfoWithSprintf("[rpc] init game node client success (auto-reconnect): nodeId:%d", nodeId)
}

func RemoveGatewayRpcGameClient(nodeId int32) {
	gameClientMu.Lock()
	defer gameClientMu.Unlock()

	client := gameClients[nodeId]
	if client != nil {
		client.Close()
	}
	delete(gameClients, nodeId)
}

func InitAllGameClients() {
	for nodeId := range ServerNodeService.GetAllGameClient() {
		InitGameRpcClients(nodeId)
	}
}

func InitRankBoardRpcClients() {
	clientFunc := func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage], error) {
		client, err := ServerNodeService.GetRankBoardClient()
		if err != nil || client == nil {
			return nil, err
		}
		return client.ForwardRankBoardMessageHandler(context.Background())
	}

	rankBoardClient = NewRankBoardNodeClient(
		clientFunc,
		config.RpcStreamRoutineCount,
		config.RpcStreamBufferSize,
		OnReceiveBackwardRankBoardMessage,
	)

	logger.InfoWithSprintf("[rpc] init rankBoard node client success (auto-reconnect)")
}

func InitSocialRpcClients() {
	if !IsSocialRpcEnabled() {
		logger.InfoWithSprintf("[rpc] social rpc disabled, skip init social node client")
		return
	}

	clientFunc := func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage], error) {
		client, err := ServerNodeService.GetSocialClient()
		if err != nil || client == nil {
			return nil, err
		}
		return client.ForwardSocialMessageHandler(context.Background())
	}

	socialClient = NewSocialNodeClient(
		clientFunc,
		config.RpcStreamRoutineCount,
		config.RpcStreamBufferSize,
		OnReceiveBackwardSocialMessage,
	)
	logger.InfoWithSprintf("[rpc] init social node client success (auto-reconnect)")
}
