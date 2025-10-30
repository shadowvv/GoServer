package sNet

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"reflect"
)

type Router struct {
	processors   map[uint32]serviceInterface.MessageProcessorInterface
	messageTypes map[uint32]uint32
	msgRegistry  map[uint32]reflect.Type
}

var _ serviceInterface.RouterInterface = (*Router)(nil)

// NewRouter
func NewRouter() *Router {
	return &Router{
		processors:   make(map[uint32]serviceInterface.MessageProcessorInterface),
		messageTypes: make(map[uint32]uint32),
		msgRegistry:  make(map[uint32]reflect.Type),
	}
}

func (r *Router) RegisterProcess(msgType, msgID uint32, msg proto.Message) {
	r.messageTypes[msgID] = msgType
	r.msgRegistry[msgID] = reflect.TypeOf(msg).Elem()

	logger.Info(fmt.Sprintf("[net] register msg msgType:%d,msgId:%d", msgType, msgID))
}

func (r *Router) RegisterProcessor(msgType uint32, processor serviceInterface.MessageProcessorInterface) {
	r.processors[msgType] = processor

	logger.Info(fmt.Sprintf("[net] register msg processor msgType:%d", msgType))
}

func (r *Router) Dispatch(connectionId int64, msgID uint32, msg proto.Message) {
	msgType, ok := r.messageTypes[msgID]
	if !ok {
		logger.Error(fmt.Sprintf("[net] unknown msgId:%d", msgID))
		return
	}
	processor, ok := r.processors[msgType]
	if !ok {
		logger.Error(fmt.Sprintf("[net] unknown msgType:%d", msgType))
		return
	}

	processor.Put(connectionId, msgID, msg)
	logger.Info(fmt.Sprintf("[net] dispatch msgId:%d", msgID))
}

func (r *Router) GetMessage(msgID uint32) proto.Message {
	t, ok := r.msgRegistry[msgID]
	if !ok {
		return nil
	}

	v := reflect.New(t).Interface()
	if m, ok := v.(proto.Message); ok {
		return m
	}
	return nil
}
