package sNet

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"github.com/gorilla/websocket"
	"net/http"
)

type NetConfig struct {
	Address       string `yaml:"address"`
	pingInterval  int64  `yaml:"pingInterval"`
	pongTimeout   int64  `yaml:"pongTimeout"`
	sendQueueSize int32  `yaml:"sendQueueSize"`
}

var pingInterval = int64(25 * 1000)
var pongTimeout = int64(60 * 1000)
var sendQueueSize = int32(256)

// WebsocketService 是 websocket 服务
type WebsocketService struct {
	addr        string
	upgrader    websocket.Upgrader
	srv         *http.Server
	idGenerator *tool.IdGenerator
	acceptor    serviceInterface.AcceptorInterface
	codec       serviceInterface.CodecInterface
	router      serviceInterface.RouterInterface
}

// NewNetService 返回默认 websocketServer
func NewNetService(config *NetConfig, serverId int32, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) *WebsocketService {
	if config.pingInterval > 0 {
		pingInterval = config.pingInterval * 1000
	}
	if config.pongTimeout > 0 {
		pongTimeout = config.pongTimeout * 1000
	}
	if config.sendQueueSize > 0 {
		sendQueueSize = config.sendQueueSize
	}

	logger.Info(fmt.Sprintf("[net] init server address:%s,pingInterval:%d,pongTimeout:%d", config.Address, pingInterval, pongTimeout))
	return &WebsocketService{
		addr: config.Address,
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
func (s *WebsocketService) Start() error {
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

func (s *WebsocketService) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("[net] upgrade failed: %v", err))
		return
	}

	c := newConn(conn, s.router, s.codec, s.idGenerator.NextId(), s.acceptor)
	s.acceptor.Accept(c)
	c.Start()
	logger.Info(fmt.Sprintf("[net] new connection: %d", c.GetID()))
}

// Shutdown 优雅停服
func (s *WebsocketService) Shutdown(ctx context.Context) error {
	// 先停止 http server（不再接收新连接）
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
		logger.Info(fmt.Sprintf("[net] ws server shutdown"))
	}
	// Router/Conn 会自行在 Context cancel 下关闭
	return nil
}
