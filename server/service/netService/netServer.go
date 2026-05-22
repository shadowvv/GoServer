package netService

import (
	"context"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"
)

type NetConfig struct {
	Address          string        `yaml:"address"`
	HeartbeatTimeout time.Duration `yaml:"heartbeatTimeout"`
	SendQueueSize    int32         `yaml:"sendQueueSize"`
	MaxMsgSize       int32         `yaml:"maxMsgSize"`
	WriteTimeout     time.Duration `yaml:"writeTimeout"`
	MaxMsgPerSecond  int32         `yaml:"maxMsgPerSecond"`
	MaxBytePerSecond int32         `yaml:"maxBytePerSecond"`
}

// 心跳超时时间
var heartbeatTimeout = 30 * time.Second

// 发送队列大小
var sendQueueSize = int32(1024)

// 最大消息长度
var maxMsgSize = int32(256 * 1024)

// 写超时时间
var writeTimeout = 5 * time.Second

// 每秒最大包数
var maxMsgPerSecond = int32(200)

// 每秒最大字节数
var maxBytePerSecond = int32(256 * 1024)

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
func NewNetService(config *NetConfig, nodeId int32, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) *WebsocketService {

	heartbeatTimeout = config.HeartbeatTimeout
	sendQueueSize = config.SendQueueSize
	maxMsgSize = config.MaxMsgSize
	writeTimeout = config.WriteTimeout
	maxMsgPerSecond = config.MaxMsgPerSecond
	maxBytePerSecond = config.MaxBytePerSecond

	port := strings.Split(config.Address, ":")[2]
	return &WebsocketService{
		addr: "0.0.0.0:" + port,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		idGenerator: tool.NewIdGenerator(int64(nodeId), 1),

		acceptor: acceptorInterface,
		codec:    codec,
		router:   router,
	}
}

// Start 启动服务，阻塞式
func (s *WebsocketService) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveWS)
	s.srv = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	return s.srv.ListenAndServe()
}

func (s *WebsocketService) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorBySprintf("[net] websocket upgrade error: %v", err)
		return
	}

	c := newSession(conn, s.router, s.codec, s.idGenerator.NextId(), s.acceptor)
	s.acceptor.Accept(c)
	c.Start()
}

// Shutdown 优雅停服
func (s *WebsocketService) Shutdown(ctx context.Context) error {
	// 先停止 http server（不再接收新连接）
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
	}
	// Router/Session 会自行在 Context cancel 下关闭
	return nil
}
