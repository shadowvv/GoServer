package ServerNodeService

import (
	"slices"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/logger"
)

var (
	nodeCache = struct {
		sync.RWMutex
		nodes map[enum.NodeType]map[int32]*NodeInfo // nodeId -> NodeInfo
	}{
		nodes: make(map[enum.NodeType]map[int32]*NodeInfo),
	}
)

func addNode(nodeType enum.NodeType, id int32, node *NodeInfo) {
	nodeCache.Lock()
	defer nodeCache.Unlock()

	if nodeCache.nodes[nodeType] == nil {
		nodeCache.nodes[nodeType] = make(map[int32]*NodeInfo)
	}
	if nodeCache.nodes[nodeType][id] != nil {
		logger.InfoWithSprintf("[platform] Node key:%s id:%d update data", nodeType, id)
	}
	nodeCache.nodes[nodeType][id] = node
	logger.InfoWithSprintf("[platform] Add node key:%s id:%d", nodeType, id)
}

func removeNode(nodeType enum.NodeType, id int32) *NodeInfo {
	nodeCache.Lock()
	defer nodeCache.Unlock()

	if nodeCache.nodes[nodeType] == nil {
		logger.ErrorBySprintf("[platform] Node key:%s id:%d not exist", nodeType, id)
		return nil
	}
	node := nodeCache.nodes[nodeType][id]
	if node == nil {
		logger.ErrorBySprintf("[platform] Node key:%s id:%d not exist", nodeType, id)
		return nil
	}
	delete(nodeCache.nodes[nodeType], id)
	logger.InfoWithSprintf("[platform] Remove node key:%s id:%d", nodeType, id)
	return node
}

func listNodes(nodeType enum.NodeType) []*NodeInfo {
	nodeCache.RLock()
	defer nodeCache.RUnlock()

	res := make([]*NodeInfo, 0, len(nodeCache.nodes[nodeType]))
	if nodeCache.nodes[nodeType] == nil {
		return res
	}
	for _, v := range nodeCache.nodes[nodeType] {
		res = append(res, v)
	}
	slices.SortFunc(res, func(a, b *NodeInfo) int {
		return int(a.NodeId - b.NodeId)
	})
	return res
}

func getNode(nodeType enum.NodeType, id int32) *NodeInfo {
	nodeCache.RLock()
	defer nodeCache.RUnlock()

	if nodeCache.nodes[nodeType] == nil {
		return nil
	}
	return nodeCache.nodes[nodeType][id]
}
