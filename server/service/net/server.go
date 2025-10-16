package net

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/gorilla/websocket"
	"net/http"
)

// Server 是 websocket 服务
type Server struct {
	Addr     string
	Upgrader websocket.Upgrader
	Router   *Router
	srv      *http.Server
	Acceptor serviceInterface.AcceptorInterface
	codec    serviceInterface.CodecInterface
	RouterT  serviceInterface.RouterInterface
}

// NewServer 返回默认 Server
func NewServer(addr string, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) *Server {
	r := NewRouter()
	return &Server{
		Addr: addr,
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Router:   r,
		Acceptor: acceptorInterface,
		codec:    codec,
		RouterT:  router,
	}
}

// Start 启动服务，阻塞式
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.serveWS)
	s.srv = &http.Server{
		Addr:         s.Addr,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	logger.Info(fmt.Sprintf("ws server listen %s\n", s.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("upgrade failed: %v", err))
		return
	}

	c := newConn(conn, s.Router)
	c.Start()
}

// Handle 注册消息处理器
func (s *Server) Handle(msgID uint32, h HandlerFunc) {
	s.Router.Handle(msgID, h)
}

// Shutdown 优雅停服
func (s *Server) Shutdown(ctx context.Context) error {
	// 先停止 http server（不再接收新连接）
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
	}
	// Router/Conn 会自行在 Context cancel 下关闭
	return nil
}
