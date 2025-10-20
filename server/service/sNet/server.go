package sNet

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
)

// websocketServer 是 websocket 服务
type websocketServer struct {
	addr        string
	upgrader    websocket.Upgrader
	srv         *http.Server
	idGenerator *tool.IdGenerator
	acceptor    serviceInterface.AcceptorInterface
	codec       serviceInterface.CodecInterface
	router      serviceInterface.RouterInterface
}

// NewServer 返回默认 websocketServer
func NewServer(addr string, serverId int32, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) *websocketServer {
	return &websocketServer{
		addr: addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		idGenerator: tool.NewIdGenerator(int64(serverId), 1),

		acceptor: acceptorInterface,
		codec:    codec,
		router:   router,
	}
}

// Start 启动服务，阻塞式
func (s *websocketServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.serveWS)
	s.srv = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	logger.Info(fmt.Sprintf("[net] ws server listen %s\n", s.addr))
	return s.srv.ListenAndServe()
}

func (s *websocketServer) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("[net] upgrade failed: %v", err))
		return
	}

	c := newConn(conn, s.router, s.codec, s.idGenerator.NextId())
	s.acceptor.Accept(c)
	c.Start()
	logger.Info(fmt.Sprintf("[net] new connection: %d", c.GetID()))
}

// Register 注册消息处理器
func (s *websocketServer) Register(msgID uint32, msg *proto.Message, h serviceInterface.HandlerFunc) {
	s.router.Register(msgID, msg, h)
	logger.Info(fmt.Sprintf("[net] register msg id:%d", msgID))
}

// Shutdown 优雅停服
func (s *websocketServer) Shutdown(ctx context.Context) error {
	// 先停止 http server（不再接收新连接）
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
		logger.Info(fmt.Sprintf("[net] ws server shutdown"))
	}
	// Router/Conn 会自行在 Context cancel 下关闭
	return nil
}
