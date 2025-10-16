package net

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
	"sync"
)

// MiddlewareFunc 可选的中间件（链式调用）
type MiddlewareFunc func(next serviceInterface.HandlerFunc) serviceInterface.HandlerFunc

// Router 简单的 msgID -> Handler 映射，支持中间件链
type Router struct {
	mu       sync.RWMutex
	handlers map[uint32]serviceInterface.HandlerFunc
	mw       []MiddlewareFunc
}

// NewRouter
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]serviceInterface.HandlerFunc),
	}
}

// Use 注册中间件（后注册的先执行）
func (r *Router) Use(m MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mw = append(r.mw, m)
}

// Handle 注册 handler
func (r *Router) Handle(msgID uint32, h serviceInterface.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 应用中间件链（从后到前包裹）
	final := h
	for i := len(r.mw) - 1; i >= 0; i-- {
		final = r.mw[i](final)
	}
	r.handlers[msgID] = final
}

// Dispatch 处理消息（非阻塞、同步调用 handler——handler 内可另起 goroutine）
func (r *Router) Dispatch(msgID uint32, c *Conn, payload []byte) {
	r.mu.RLock()
	h, ok := r.handlers[msgID]
	r.mu.RUnlock()
	if !ok {
		// no handler, ignore or log
		return
	}
	// 以同步方式调用 handler；如果 handler 需要并发，可自身启动 goroutine
	h(c, payload)
}
