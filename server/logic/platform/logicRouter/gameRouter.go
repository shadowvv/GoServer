package logicRouter

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"reflect"
)

type GameRouter struct {
	dispatcher   serviceInterface.DispatchInterface
	messageTypes map[int32]int32
	msgRegistry  map[int32]reflect.Type
	msgIsEmpty   map[int32]bool
}

var _ serviceInterface.RouterInterface = (*GameRouter)(nil)

func NewGameRouter(dispatcher serviceInterface.DispatchInterface) *GameRouter {
	return &GameRouter{
		messageTypes: make(map[int32]int32),
		msgRegistry:  make(map[int32]reflect.Type),
		msgIsEmpty:   make(map[int32]bool),
		dispatcher:   dispatcher,
	}
}

func (r *GameRouter) RegisterMessage(msgType int32, msgID int32, msg proto.Message) {
	if _, ok := r.messageTypes[msgID]; ok {
		panic(fmt.Sprintf("[net] duplicate msgId:%d", msgID))
	}
	r.messageTypes[msgID] = msgType
	r.msgRegistry[msgID] = reflect.TypeOf(msg).Elem()
	r.msgIsEmpty[msgID] = msg.ProtoReflect().Descriptor().Fields().Len() == 0

	logger.InfoWithSprintf("[net] register msg msgType:%d,msgId:%d", msgType, msgID)
}

func (r *GameRouter) Dispatch(session serviceInterface.SessionInterface, msgID int32, msg proto.Message) {
	msgType, ok := r.messageTypes[msgID]
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] unknown msgId:%d,sessionId:%d", msgID, session.GetID()))
		return
	}
	r.dispatcher.DispatchGameMessage(session, msgID, msgType, msg)
}

func (r *GameRouter) GetMessage(msgID int32) proto.Message {
	t, ok := r.msgRegistry[msgID]
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] unknown msgId:%d", msgID))
		return nil
	}

	v := reflect.New(t).Interface()
	if m, ok := v.(proto.Message); ok {
		return m
	}
	logger.ErrorWithZapFields(fmt.Sprintf("[net] unknown msgId:%d", msgID))
	return nil
}

func (r *GameRouter) IsEmpty(msgID int32) bool {
	return r.msgIsEmpty[msgID]
}
