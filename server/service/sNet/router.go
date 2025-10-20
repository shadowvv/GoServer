package sNet

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"reflect"
)

// Router 简单的 msgID -> Handler 映射，支持中间件链
type Router struct {
	handlers    map[uint32]serviceInterface.HandlerFunc
	msgRegistry map[uint32]reflect.Type
}

// NewRouter
func NewRouter() *Router {
	return &Router{
		handlers:    make(map[uint32]serviceInterface.HandlerFunc),
		msgRegistry: make(map[uint32]reflect.Type),
	}
}

func (r *Router) Register(msgID uint32, msg *proto.Message, h serviceInterface.HandlerFunc) {
	r.msgRegistry[msgID] = reflect.TypeOf(msg).Elem()
	r.handlers[msgID] = h
}

func (r *Router) Dispatch(msgID uint32, msg *proto.Message) {
	h, ok := r.handlers[msgID]
	if !ok {
		return
	}
	h(msgID, msg)
}

func (r *Router) GetMessage(msgID uint32) *proto.Message {
	t, ok := r.msgRegistry[msgID]
	if !ok {
		return nil
	}

	v := reflect.New(t).Interface()
	if m, ok := v.(proto.Message); ok {
		return &m
	}
	return nil
}
