package dispatcherService

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type Dispatcher struct {
	router                serviceInterface.RouterInterface
	gatewayMsgProcessor   *GatewayMessageProcessor
	loginMsgProcessor     *LoginMessageProcessor
	sceneMsgProcessor     *SceneMessageProcessor
	rankBoardMsgProcessor *RankBoardMessageProcessor
	allianceMsgProcessor  *AllianceMessageProcessor
	innerMessageManager   *InnerTaskFutureManager
	innerTaskIdGenerator  *tool.IdGenerator
	nodeType              enum.NodeType
}

var _ serviceInterface.DispatchInterface = (*Dispatcher)(nil)

// TODO: idGenerator当前无用，之后
func NewDispatcher(router serviceInterface.RouterInterface, sessionManager logicCommon.SessionManagerInterface, idGenerator *tool.IdGenerator, nodeType enum.NodeType, processConfig map[string]*nodeConfig.MessageProcessConfig) *Dispatcher {
	dispatcher := &Dispatcher{}
	dispatcher.router = router
	dispatcher.innerTaskIdGenerator = idGenerator
	dispatcher.nodeType = nodeType
	switch nodeType {
	case enum.NODE_TYPE_HTTP:
	case enum.NODE_TYPE_GATEWAY:
		config := processConfig["gateway"]
		if config == nil {
			panic("[dispatcher] gateway process config is nil")
		}
		dispatcher.gatewayMsgProcessor = NewGatewayMessageProcess(sessionManager, config)
	case enum.NODE_TYPE_GAME:
		loginConfig := processConfig["login"]
		if loginConfig == nil {
			panic("[dispatcher] login process config is nil")
		}
		dispatcher.loginMsgProcessor = NewLoginMessageProcessor(loginConfig)
		dispatcher.sceneMsgProcessor = NewSceneMessageProcessor(sessionManager)
		dispatcher.innerMessageManager = NewInnerMessageFutureManager(dispatcher)
	case enum.NODE_TYPE_RANKBOARD:
		config := processConfig["rankBoard"]
		if config == nil {
			panic("[dispatcher] login rankBoard config is nil")
		}
		dispatcher.rankBoardMsgProcessor = NewRankBoardMessageProcessor(config)
	case enum.NODE_TYPE_SOCIAL:
		config := processConfig["alliance"]
		if config == nil {
			panic("[dispatcher] alliance process config is nil")
		}
		dispatcher.allianceMsgProcessor = NewAllianceMessageProcessor(config)
	default:
		panic(fmt.Sprintf("[dispatcher] node type error type:%s", nodeType))
	}
	return dispatcher
}

func (d *Dispatcher) DispatchGameMessage(session serviceInterface.SessionInterface, msgID, msgType int32, msg proto.Message) {
	if d.nodeType == enum.NODE_TYPE_GATEWAY {
		d.gatewayMsgProcessor.PushMessage(session, msgType, msg)
	} else if d.nodeType == enum.NODE_TYPE_GAME {
		if msgType == int32(enum.MSG_TYPE_LOGIN) {
			d.loginMsgProcessor.PushMessage(session, msgID, msg)
		} else if msgType == int32(enum.MSG_TYPE_PLAYER) {
			d.sceneMsgProcessor.PushMessage(session, msgID, msg)
		}
	} else if d.nodeType == enum.NODE_TYPE_RANKBOARD {
		d.rankBoardMsgProcessor.PushMessage(session, msgID, msg)
	} else if d.nodeType == enum.NODE_TYPE_SOCIAL {
		d.allianceMsgProcessor.PushMessage(session, msgID, msg)
	}
}

func (d *Dispatcher) DispatchInnerMessageTask(reqType enum.InnerMessageType, msgId enum.InnerMessageId, reqId int64, parameter any, respType enum.InnerMessageType, respId int64, respCallback serviceInterface.InnerTaskResult) {
	t := &InnerTask{
		Id:           d.innerTaskIdGenerator.NextId(),
		MessageId:    msgId,
		ReqType:      reqType,
		ReqId:        reqId,
		ReqParameter: parameter,
		RespType:     respType,
		RespId:       respId,
		RespCallback: respCallback,
		err:          nil,
		done:         make(chan struct{}),
	}
	d.DispatchInnerTask(t)
}

func (d *Dispatcher) DispatchInnerTask(task serviceInterface.InnerTaskInterface) {
	t := task.(*InnerTask)
	if t.ReqType == enum.INNER_MSG_TYPE_PLAYER {
		d.sceneMsgProcessor.PushPlayerInnerTask(t)
	} else if t.ReqType == enum.INNER_MSG_TYPE_SCENE {
		d.sceneMsgProcessor.PushSceneInnerTask(t)
	}
	d.innerMessageManager.AddTask(t)
}

func (d *Dispatcher) DispatchInnerTaskResp(task serviceInterface.InnerTaskInterface, respHandler func()) {
	t := task.(*InnerTask)
	if t.RespType == enum.INNER_MSG_TYPE_PLAYER {
		d.sceneMsgProcessor.PushPlayerInnerResp(t, respHandler)
	} else if t.RespType == enum.INNER_MSG_TYPE_SCENE {
		d.sceneMsgProcessor.PushSceneInnerResp(t, respHandler)
	}
}

func (d *Dispatcher) RegisterLoginMessageHandler(msgID int32, h logicCommon.LoginMessageHandler) {
	if d.nodeType != enum.NODE_TYPE_GAME {
		return
	}
	d.loginMsgProcessor.RegisterLoginMessageHandler(msgID, h)
}

func (d *Dispatcher) RegisterPlayerMessageHandler(msgID int32, h logicCommon.PlayerMessageHandler, function enum.FunctionIdEnum) {
	if d.nodeType != enum.NODE_TYPE_GAME {
		return
	}
	d.sceneMsgProcessor.RegisterPlayerMessageHandler(msgID, h, function)
}

func (d *Dispatcher) RegisterGatewayMessageHandler(msgType enum.MessageType, h logicCommon.GatewayMessageHandler) {
	d.gatewayMsgProcessor.RegisterGatewayMessageHandler(int32(msgType), h)
}

func (d *Dispatcher) RegisterRankBoardMessageHandler(id int32, h logicCommon.RankBoardMessageHandler) {
	d.rankBoardMsgProcessor.RegisterRankBoardMessageHandler(id, h)
}

func (d *Dispatcher) RegisterAllianceMessageHandler(id int32, h logicCommon.AllianceMessageHandler) {
	if d.nodeType != enum.NODE_TYPE_SOCIAL {
		return
	}
	d.allianceMsgProcessor.RegisterAllianceMessageHandler(id, h)
}

func (d *Dispatcher) RegisterInnerMessageHandler(msgType enum.InnerMessageType, msgId int32, h logicCommon.InnerMessageHandler) {
	if d.nodeType != enum.NODE_TYPE_GAME {
		return
	}
	if msgType == enum.INNER_MSG_TYPE_PLAYER {
		d.sceneMsgProcessor.RegisterInnerMessageHandler(msgId, h)
	}
}

func (d *Dispatcher) RegisterSceneProcessor(sceneId int32, processor logicCommon.SingleSceneProcessor) {
	d.sceneMsgProcessor.RegisterSceneProcessor(sceneId, processor)
}
