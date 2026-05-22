package logicRouter

import (
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"reflect"
)

type GatewayRouter struct {
	dispatcher   serviceInterface.DispatchInterface
	messageTypes map[int32]int32
	msgRegistry  map[int32]reflect.Type
	msgIsEmpty   map[int32]bool
}

var _ serviceInterface.RouterInterface = (*GatewayRouter)(nil)

func NewGatewayRouter(dispatcher serviceInterface.DispatchInterface) *GatewayRouter {
	return &GatewayRouter{
		dispatcher:   dispatcher,
		messageTypes: make(map[int32]int32),
		msgRegistry:  make(map[int32]reflect.Type),
		msgIsEmpty:   make(map[int32]bool),
	}
}

func (r *GatewayRouter) RegisterMessage(msgType int32, msgID int32, msg proto.Message) {
	r.messageTypes[msgID] = msgType
	r.msgRegistry[msgID] = reflect.TypeOf(&rpcPb.ClientReq{}).Elem()
	r.msgIsEmpty[msgID] = msg.ProtoReflect().Descriptor().Fields().Len() == 0

	logger.InfoWithSprintf("[platform] register msg msgType:%d,msgId:%d", msgType, msgID)
}

func (r *GatewayRouter) Dispatch(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	msgType, ok := r.messageTypes[msgID]
	if !ok {
		logger.ErrorBySprintf("[platform] unknown msgId:%d,sessionId:%d", msgID, session.GetID())
		return
	}
	r.dispatcher.DispatchGameMessage(session, msgID, msgType, msg)
}

func (r *GatewayRouter) GetMessage(msgID int32) proto.Message {
	t, ok := r.msgRegistry[msgID]
	if !ok {
		logger.ErrorBySprintf("[platform] unknown msgId:%d", msgID)
		return nil
	}

	v := reflect.New(t).Interface()
	if m, ok := v.(proto.Message); ok {
		return m
	}
	logger.ErrorBySprintf("[platform] unknown msgId:%d", msgID)
	return nil
}

func (r *GatewayRouter) IsEmpty(msgID int32) bool {
	return r.msgIsEmpty[msgID]
}
