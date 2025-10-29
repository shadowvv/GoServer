package sNet

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"reflect"
)

type Router struct {
	handlers    map[uint32]serviceInterface.MessageProcessorInterface
	msgRegistry map[uint32]reflect.Type
}

// NewRouter
func NewRouter() *Router {
	return &Router{
		handlers:    make(map[uint32]serviceInterface.MessageProcessorInterface),
		msgRegistry: make(map[uint32]reflect.Type),
	}
}

func (r *Router) RegisterProcess(msgID uint32, msg proto.Message, processor serviceInterface.MessageProcessorInterface) {
	r.msgRegistry[msgID] = reflect.TypeOf(msg).Elem()
	r.handlers[msgID] = processor
	logger.Info(fmt.Sprintf("[net] register msg id:%d", msgID))
}

func (r *Router) Dispatch(connectionId int64, msgID uint32, msg proto.Message) {
	processor, ok := r.handlers[msgID]
	if !ok {
		return
	}
	processor.Put(connectionId, msgID, msg)
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
