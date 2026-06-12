package ServerNodeService

import (
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"strconv"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/easyRpc"
	"github.com/drop/GoServer/server/service/etcd"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	HTTP_NODE      = string("/node/" + enum.NODE_TYPE_HTTP + "/")      // HTTP节点信息
	GATE_NODE      = string("/node/" + enum.NODE_TYPE_GATEWAY + "/")   // 网关节点信息
	GAME_NODE      = string("/node/" + enum.NODE_TYPE_GAME + "/")      // 游戏节点信息
	SOCIAL_NODE    = string("/node/" + enum.NODE_TYPE_SOCIAL + "/")    // 社交节点信息
	RANKBOARD_NODE = string("/node/" + enum.NODE_TYPE_RANKBOARD + "/") // 排行榜节点信息

	NODE_PORT_BEGIN         = 9000
	NODE_PORT_TYPE_INTERVAL = 10
)

// rpc配置
type RpcConfig struct {
	RpcStreamRoutineCount int32         `yaml:"rpcStreamRoutineCount"`
	RpcStreamBufferSize   int32         `yaml:"rpcStreamBufferSize"`
	RpcPingTimeSecond     time.Duration `yaml:"rpcPingTimeSecond"`
	RpcPongTimeoutSecond  time.Duration `yaml:"rpcPongTimeoutSecond"`
}

// 节点信息
type NodeInfo struct {
	NodeId      int32                 `json:"node_id"`      // 节点id
	NodeAddress string                `json:"node_address"` // 节点地址
	NodeType    enum.NodeType         `json:"node_type"`    // 节点类型
	NodeWeight  int32                 `json:"node_weight"`  // 节点权重
	NodeStatus  enum.GameServerStatus `json:"node_status"`  // 节点状态
	NodeParam   string                `json:"node_param"`   // 节点参数
}

//TODO: 后续优化节点状态和节点权重

var rpcHooker logicCommon.RPCServiceHooker // rpc服务钩子
var currentNodeInfo *NodeInfo              // 当前节点信息
var etcdService *etcd.EtcdService          // etcd服务
var rpcServer *grpc.Server                 // rpc服务
var leaseHolder *etcd.LeaseHolder          // 租约
var rpcCfg *RpcConfig                      // rpc配置
var watchCancels []context.CancelFunc

func InitNodeService(hooker logicCommon.RPCServiceHooker, nodeId int32, nodeType enum.NodeType, rpcConfig *RpcConfig, etcdConfig *etcd.Config, nodeParam string, proxyService ...func(s *grpc.Server)) {
	if nodeId > enum.MAX_NODE_ID || nodeId < enum.MIN_NODE_ID {
		panic("[platform] node id invalid")
	}
	rpcHooker = hooker
	rpcCfg = rpcConfig
	watchCancels = make([]context.CancelFunc, 0)
	easyRpc.SetClientPingPongTime(rpcCfg.RpcPingTimeSecond, rpcConfig.RpcPongTimeoutSecond)
	port := NODE_PORT_BEGIN + enum.GetNodeTypeIndex(nodeType)*NODE_PORT_TYPE_INTERVAL + nodeId
	localIp, err := tool.GetLocalIP()
	if err != nil {
		panic("[platform] get local ip error")
	}
	address := localIp + ":" + strconv.Itoa(int(port))

	currentNodeInfo = &NodeInfo{
		NodeId:      nodeId,
		NodeAddress: address,
		NodeType:    nodeType,
		NodeWeight:  100,
		NodeStatus:  enum.GAME_SERVER_STATUS_ONLINE,
		NodeParam:   nodeParam,
	}

	initEtcdService(etcdConfig, nodeType)
	startRpcServer("0.0.0.0:"+strconv.Itoa(int(port)), proxyService...)

	key := HTTP_NODE
	switch nodeType {
	case enum.NODE_TYPE_HTTP:
		key = HTTP_NODE
	case enum.NODE_TYPE_GATEWAY:
		key = GATE_NODE
	case enum.NODE_TYPE_GAME:
		key = GAME_NODE
	case enum.NODE_TYPE_SOCIAL:
		key = SOCIAL_NODE
	case enum.NODE_TYPE_RANKBOARD:
		key = RANKBOARD_NODE
	default:
		panic("[platform] etcd invalid node type")
	}
	key += strconv.Itoa(int(nodeId))
	data, err := json.Marshal(currentNodeInfo)
	if err != nil {
		logger.ErrorWithZapFields("[platform] etcd marshal node info error", zap.Error(err))
		panic("[platform] etcd marshal node info error")
	}
	holder, err := etcdService.RegisterNode(key, string(data), etcdConfig.TTL)
	if err != nil {
		panic("[platform] etcd register node error")
	}
	leaseHolder = holder
	logger.InfoWithSprintf("[platform] Register node %+v", leaseHolder)

	initRpcClient(nodeType)
}

func initEtcdService(etcdConfig *etcd.Config, nodeType enum.NodeType) {
	data, _ := json.MarshalIndent(etcdConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init etcd service config:%s", string(data))
	service, err := etcd.NewEtcdService(etcdConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] init etcd service error", zap.Error(err))
		panic(err)
	}
	etcdService = service

	switch nodeType {
	case enum.NODE_TYPE_HTTP:
		cancel := service.WatchNodes(GATE_NODE, onGateNodeChange)
		watchCancels = append(watchCancels, cancel)
		initAllNode(GATE_NODE)
		logger.InfoWithSprintf("[platform] Init etcd service watch node %s", GATE_NODE)
	case enum.NODE_TYPE_GATEWAY:
		cancel := service.WatchNodes(GAME_NODE, onGameNodeChange)
		watchCancels = append(watchCancels, cancel)
		initAllNode(GAME_NODE)
		logger.InfoWithSprintf("[platform] Init etcd service watch node %s", GAME_NODE)
	case enum.NODE_TYPE_GAME:
		cancel1 := service.WatchNodes(GATE_NODE, onGateNodeChange)
		cancel2 := service.WatchNodes(SOCIAL_NODE, onSocialNodeChange)
		cancel3 := service.WatchNodes(RANKBOARD_NODE, onRankNodeChange)
		watchCancels = append(watchCancels, cancel1, cancel2, cancel3)
		initAllNode(GATE_NODE, SOCIAL_NODE, RANKBOARD_NODE)
		logger.InfoWithSprintf("[platform] Init etcd service watch node %s %s %s", GATE_NODE, SOCIAL_NODE, RANKBOARD_NODE)
	case enum.NODE_TYPE_SOCIAL:
		cancel1 := service.WatchNodes(GATE_NODE, onGateNodeChange)
		cancel2 := service.WatchNodes(GAME_NODE, onGameNodeChange)
		cancel3 := service.WatchNodes(RANKBOARD_NODE, onRankNodeChange)
		watchCancels = append(watchCancels, cancel1, cancel2, cancel3)
		initAllNode(GAME_NODE, GATE_NODE, RANKBOARD_NODE)
		logger.InfoWithSprintf("[platform] Init etcd service watch node %s %s %s", GAME_NODE, GATE_NODE, RANKBOARD_NODE)
	case enum.NODE_TYPE_RANKBOARD:
		cancel1 := service.WatchNodes(GATE_NODE, onGateNodeChange)
		cancel2 := service.WatchNodes(GAME_NODE, onGameNodeChange)
		watchCancels = append(watchCancels, cancel1, cancel2)
		initAllNode(GAME_NODE, GATE_NODE)
		logger.InfoWithSprintf("[platform] Init etcd service watch node %s %s", GAME_NODE, GATE_NODE)
	default:
		panic("[platform] init etcd server invalid node type")
	}
}

func initAllNode(key ...string) {
	for _, k := range key {
		nodes, _, err := etcdService.GetOnlineNodes(k)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get online nodes error", zap.Error(err))
			panic(err)
		}
		for _, v := range nodes {
			nodeInfo := &NodeInfo{}
			err := json.Unmarshal([]byte(v), nodeInfo)
			if err != nil {
				logger.ErrorWithZapFields("[platform] unmarshal node info error", zap.Error(err))
			}
			addNode(nodeInfo.NodeType, nodeInfo.NodeId, nodeInfo)
		}
	}
}

func startRpcServer(address string, service ...func(s *grpc.Server)) {
	serverOption := &easyRpc.ServerOptions{
		Address: address,
		Options: []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(UnaryServerRecovery(), UnaryServerLogging()),
			grpc.ChainStreamInterceptor(StreamServerRecovery(), StreamServerLogging()),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				Time:    rpcCfg.RpcPingTimeSecond,
				Timeout: rpcCfg.RpcPongTimeoutSecond,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             15 * time.Second,
				PermitWithoutStream: true,
			}),
		},
	}
	server, err := easyRpc.NewServer(serverOption, service...)
	if err != nil {
		logger.ErrorWithZapFields("[platform] init rpc server error", zap.Error(err))
		panic(err)
	}
	rpcServer = server
	logger.InfoWithSprintf("[platform] Start rpc server %s", address)
}

func initRpcClient(nodeType enum.NodeType) {
	switch nodeType {
	case enum.NODE_TYPE_HTTP:
		InitGateRpcClient()
	case enum.NODE_TYPE_GATEWAY:
		initAllGameRpcClient()
	case enum.NODE_TYPE_GAME:
		InitGateRpcClient()
		InitRankBoardRpcClient()
		InitSocialRpcClient()
	case enum.NODE_TYPE_SOCIAL:
		initAllGameRpcClient()
		InitRankBoardRpcClient()
	case enum.NODE_TYPE_RANKBOARD:
		initAllGameRpcClient()
	default:
		panic("[platform] init rpc client invalid node type")
	}
}

func InitGateRpcClient() {
	gateInfos := listNodes(enum.NODE_TYPE_GATEWAY)
	if len(gateInfos) == 0 {
		logger.ErrorWithZapFields("[platform] no gate node online")
		return
	}
	for _, v := range gateInfos {
		conn, err := easyRpc.GetClientConn(v.NodeAddress)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get gate node client error", zap.Error(err))
			return
		}
		gatewayClient := rpcPb.NewGateServiceClient(conn)
		_, _ = gatewayClient.SayHello(context.Background(), &rpcPb.HelloReq{NodeId: currentNodeInfo.NodeId, NodeType: string(currentNodeInfo.NodeType), Address: currentNodeInfo.NodeAddress})
		logger.InfoWithSprintf("[platform] Init gate node client %v", v)
	}
}

func InitGameRpcClient(id int32) {
	gameInfo, err := GetNodeById(enum.NODE_TYPE_GAME, id)
	if err != nil {
		logger.ErrorWithZapFields("[platform] get game node error", zap.Error(err))
		return
	}
	conn, err := easyRpc.GetClientConn(gameInfo.NodeAddress)
	if err != nil {
		logger.ErrorWithZapFields("[platform] get game node client error", zap.Error(err))
		return
	}
	gameClient := rpcPb.NewGameServiceClient(conn)
	_, _ = gameClient.SayHello(context.Background(), &rpcPb.HelloReq{NodeId: currentNodeInfo.NodeId, NodeType: string(currentNodeInfo.NodeType), Address: currentNodeInfo.NodeAddress})
	logger.InfoWithSprintf("[platform] Init game node client %v", gameInfo)
}

func initAllGameRpcClient() {
	gameInfos := listNodes(enum.NODE_TYPE_GAME)
	if len(gameInfos) == 0 {
		logger.ErrorWithZapFields("[platform] no game node online")
		return
	}
	for _, v := range gameInfos {
		conn, err := easyRpc.GetClientConn(v.NodeAddress)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get game node client error", zap.Error(err))
			continue
		}
		gameClient := rpcPb.NewGameServiceClient(conn)
		_, _ = gameClient.SayHello(context.Background(), &rpcPb.HelloReq{NodeId: currentNodeInfo.NodeId, NodeType: string(currentNodeInfo.NodeType), Address: currentNodeInfo.NodeAddress})
		logger.InfoWithSprintf("[platform] Init game node client %v", v)
	}
}

func InitRankBoardRpcClient() {
	rankInfos := listNodes(enum.NODE_TYPE_RANKBOARD)
	if len(rankInfos) == 0 {
		logger.ErrorWithZapFields("[platform] no rank node online")
		return
	}
	for _, v := range rankInfos {
		conn, err := easyRpc.GetClientConn(v.NodeAddress)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get rank node client error", zap.Error(err))
			continue
		}
		rankClient := rpcPb.NewRankServiceClient(conn)
		_, _ = rankClient.SayHello(context.Background(), &rpcPb.HelloReq{NodeId: currentNodeInfo.NodeId, NodeType: string(currentNodeInfo.NodeType), Address: currentNodeInfo.NodeAddress})
		logger.InfoWithSprintf("[platform] Init rank node client %v", v)
	}
}

func InitSocialRpcClient() {
	socialInfos := listNodes(enum.NODE_TYPE_SOCIAL)
	if len(socialInfos) == 0 {
		logger.ErrorWithZapFields("[platform] no social node online")
		return
	}
	for _, v := range socialInfos {
		conn, err := easyRpc.GetClientConn(v.NodeAddress)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get social node client error", zap.Error(err))
			continue
		}
		socialClient := rpcPb.NewSocialServiceClient(conn)
		_, _ = socialClient.SayHello(context.Background(), &rpcPb.HelloReq{NodeId: currentNodeInfo.NodeId, NodeType: string(currentNodeInfo.NodeType), Address: currentNodeInfo.NodeAddress})
		logger.InfoWithSprintf("[platform] Init social node client %v", v)
	}
}

func GetGateWayOpenAddress() (string, error) {
	gateInfo, err := randomNode(enum.NODE_TYPE_GATEWAY)
	if err != nil {
		logger.ErrorWithZapFields("[platform] get gateway node error", zap.Error(err))
		return "", err
	}
	return gateInfo.NodeParam, nil
}

func GetGameNodeIdByUserId(userId int64) (int32, error) {
	nodes := listNodes(enum.NODE_TYPE_GAME)
	num := len(nodes)
	if num == 0 {
		return 0, errors.New("no node available")
	}
	hash := crc32.ChecksumIEEE([]byte(strconv.FormatInt(userId, 10)))
	index := int64(hash % uint32(num))
	logger.InfoWithSprintf("[platform] get game nodeId:%d,userId:%d,nodeNum:%d", nodes[index].NodeId, userId, num)
	return nodes[index].NodeId, nil
}

func randomNode(nodeType enum.NodeType) (*NodeInfo, error) {
	nodes := listNodes(nodeType)
	num := len(nodes)
	if num == 0 {
		return nil, errors.New("no node available")
	}
	return nodes[tool.RandInt(0, num-1)], nil
}

func GetNodeById(nodeType enum.NodeType, id int32) (*NodeInfo, error) {
	nodes := listNodes(nodeType)
	for _, node := range nodes {
		if node.NodeId == id {
			return node, nil
		}
	}
	return nil, errors.New("no node available")
}

func GetGatewayClient() (rpcPb.GateServiceClient, error) {
	node, err := randomNode(enum.NODE_TYPE_GATEWAY)
	if err != nil {
		return nil, err
	}
	conn, err := easyRpc.GetClientConn(node.NodeAddress)
	if err != nil {
		return nil, err
	}
	return rpcPb.NewGateServiceClient(conn), nil
}

func GetAllGameClient() map[int32]rpcPb.GameServiceClient {
	nodes := listNodes(enum.NODE_TYPE_GAME)
	clients := make(map[int32]rpcPb.GameServiceClient, len(nodes))
	for _, node := range nodes {
		conn, err := easyRpc.GetClientConn(node.NodeAddress)
		if err != nil {
			logger.ErrorWithZapFields("[platform] get game node client error", zap.Error(err))
			continue
		}
		clients[node.NodeId] = rpcPb.NewGameServiceClient(conn)
	}
	return clients
}

func GetGameClientWithId(id int32) (rpcPb.GameServiceClient, error) {
	node, err := GetNodeById(enum.NODE_TYPE_GAME, id)
	if err != nil {
		return nil, err
	}
	conn, err := easyRpc.GetClientConn(node.NodeAddress)
	if err != nil {
		return nil, err
	}
	return rpcPb.NewGameServiceClient(conn), nil
}

func GetRankBoardClient() (rpcPb.RankServiceClient, error) {
	node, err := randomNode(enum.NODE_TYPE_RANKBOARD)
	if err != nil {
		return nil, err
	}
	conn, err := easyRpc.GetClientConn(node.NodeAddress)
	if err != nil {
		return nil, err
	}
	// 额外检查：防止 GetClientConn 返回 nil conn 但不返回 error
	if conn == nil {
		return nil, errors.New("get client conn is nil")
	}
	return rpcPb.NewRankServiceClient(conn), nil
}

func GetSocialClient() (rpcPb.SocialServiceClient, error) {
	node, err := randomNode(enum.NODE_TYPE_SOCIAL)
	if err != nil {
		return nil, err
	}
	conn, err := easyRpc.GetClientConn(node.NodeAddress)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, errors.New("get client conn is nil")
	}
	return rpcPb.NewSocialServiceClient(conn), nil
}
