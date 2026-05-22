package ServerNodeService

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/etcd"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

type nodeReconnectState struct {
	NeedReconnect bool
	LastLease     int64
}

var reconnectState = struct {
	sync.Mutex
	states map[enum.NodeType]map[int32]*nodeReconnectState
}{
	states: make(map[enum.NodeType]map[int32]*nodeReconnectState),
}

func getReconnectState(nodeType enum.NodeType, nodeID int32) *nodeReconnectState {
	if reconnectState.states[nodeType] == nil {
		reconnectState.states[nodeType] = make(map[int32]*nodeReconnectState)
	}
	state := reconnectState.states[nodeType][nodeID]
	if state == nil {
		state = &nodeReconnectState{}
		reconnectState.states[nodeType][nodeID] = state
	}
	return state
}

func markReconnectNeeded(nodeType enum.NodeType, nodeID int32) {
	reconnectState.Lock()
	defer reconnectState.Unlock()
	state := getReconnectState(nodeType, nodeID)
	state.NeedReconnect = true
}

func shouldReconnectByState(nodeType enum.NodeType, nodeID int32, lease int64) bool {
	reconnectState.Lock()
	defer reconnectState.Unlock()

	state := getReconnectState(nodeType, nodeID)
	needReconnect := state.NeedReconnect

	if state.LastLease != 0 && lease != 0 && state.LastLease != lease {
		needReconnect = true
	}

	state.NeedReconnect = false
	state.LastLease = lease
	return needReconnect
}

func handleNodeWatch(ev etcd.WatchEvent) {
	switch ev.Type {
	case "put":
		node := &NodeInfo{}
		if err := json.Unmarshal([]byte(ev.Value), node); err != nil {
			logger.ErrorWithZapFields("[platform] etcd watch unmarshal nodeInfo error", zap.Error(err))
			return
		}

		reconnectByState := shouldReconnectByState(node.NodeType, node.NodeId, ev.Lease)
		currentNode := getNode(node.NodeType, node.NodeId)
		if currentNode != nil {
			shouldReconnect := reconnectByState
			if currentNode.NodeAddress != node.NodeAddress {
				shouldReconnect = true
			}
			if currentNode.NodeStatus == enum.GAME_SERVER_STATUS_ONLINE && node.NodeStatus == enum.GAME_SERVER_STATUS_MAINTAIN {
				rpcHooker.OnNodeDisconnect(currentNode.NodeId, currentNode.NodeType, currentNode.NodeAddress)
				addNode(node.NodeType, node.NodeId, node)
				return
			}
			if currentNode.NodeStatus == enum.GAME_SERVER_STATUS_MAINTAIN && node.NodeStatus == enum.GAME_SERVER_STATUS_ONLINE {
				shouldReconnect = true
			}
			if shouldReconnect {
				rpcHooker.OnNodeConnect(node.NodeId, node.NodeType)
			}
			addNode(node.NodeType, node.NodeId, node)
		} else {
			addNode(node.NodeType, node.NodeId, node)
			rpcHooker.OnNodeConnect(node.NodeId, node.NodeType)
		}

	case "delete":
		parts := strings.Split(ev.Key, "/")
		if len(parts) < 4 {
			logger.ErrorBySprintf("[platform] etcd parse node key error, key:%s", ev.Key)
			return
		}
		nodeType := enum.NodeType(parts[2])
		id, err := strconv.Atoi(parts[3])
		if err != nil {
			logger.ErrorWithZapFields("[platform] etcd parse node id error", zap.Error(err))
			return
		}

		markReconnectNeeded(nodeType, int32(id))

		node, err := GetNodeById(nodeType, int32(id))
		if err != nil {
			logger.ErrorWithZapFields("[platform] etcd get node error", zap.Error(err))
		}
		if node != nil {
			rpcHooker.OnNodeDisconnect(int32(id), nodeType, node.NodeAddress)
		}
	}
}

func onGameNodeChange(ev etcd.WatchEvent) {
	handleNodeWatch(ev)
}

func onGateNodeChange(ev etcd.WatchEvent)   { handleNodeWatch(ev) }
func onHttpNodeChange(ev etcd.WatchEvent)   { handleNodeWatch(ev) }
func onSocialNodeChange(ev etcd.WatchEvent) { handleNodeWatch(ev) }
func onRankNodeChange(ev etcd.WatchEvent)   { handleNodeWatch(ev) }
