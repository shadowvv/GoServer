package netService

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"github.com/gorilla/websocket"
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

const (
	defaultHeartbeatTimeout = 30 * time.Second
	defaultSendQueueSize    = int32(1024)
	defaultMaxMsgSize       = int32(256 * 1024)
	defaultWriteTimeout     = 5 * time.Second
	defaultMaxMsgPerSecond  = int32(200)
	defaultMaxBytePerSecond = int32(256 * 1024)
)

type sessionRuntimeConfig struct {
	heartbeatTimeout time.Duration
	sendQueueSize    int32
	maxMsgSize       int32
	writeTimeout     time.Duration
	maxMsgPerSecond  int32
	maxBytePerSecond int32
}

// WebsocketService websocket service
type WebsocketService struct {
	addr        string
	upgrader    websocket.Upgrader
	srv         *http.Server
	idGenerator *tool.IdGenerator
	sessionCfg  *sessionRuntimeConfig
	acceptor    serviceInterface.AcceptorInterface
	codec       serviceInterface.CodecInterface
	router      serviceInterface.RouterInterface
}

// NewNetService create websocket server
func NewNetService(config *NetConfig, nodeId int32, acceptorInterface serviceInterface.AcceptorInterface, codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) *WebsocketService {
	rawAddr := ""
	if config != nil {
		rawAddr = config.Address
	}
	return &WebsocketService{
		addr:       normalizeListenAddr(rawAddr),
		sessionCfg: normalizeSessionRuntimeConfig(config),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		idGenerator: tool.NewIdGenerator(int64(nodeId), 1),
		acceptor:    acceptorInterface,
		codec:       codec,
		router:      router,
	}
}

// Start start service
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
		remoteIP := r.RemoteAddr
		if host, _, splitErr := net.SplitHostPort(r.RemoteAddr); splitErr == nil {
			remoteIP = host
		}

		logger.ErrorBySprintf(
			"[net] websocket upgrade error: err=%v, remoteAddr=%s, remoteIP=%s, method=%s, uri=%s, host=%s, userAgent=%s, connection=%s, upgrade=%s, xForwardedFor=%s, xRealIP=%s",
			err,
			r.RemoteAddr,
			remoteIP,
			r.Method,
			r.RequestURI,
			r.Host,
			r.UserAgent(),
			r.Header.Get("Connection"),
			r.Header.Get("Upgrade"),
			r.Header.Get("X-Forwarded-For"),
			r.Header.Get("X-Real-IP"),
		)
		return
	}

	c := newSession(conn, s.router, s.codec, s.idGenerator.NextId(), s.acceptor, *s.sessionCfg)
	s.acceptor.Accept(c)
	if !c.IsActive() {
		return
	}
	c.Start()
}

// Shutdown graceful shutdown
func (s *WebsocketService) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
	}
	return nil
}

func normalizeSessionRuntimeConfig(config *NetConfig) *sessionRuntimeConfig {
	cfg := sessionRuntimeConfig{
		heartbeatTimeout: defaultHeartbeatTimeout,
		sendQueueSize:    defaultSendQueueSize,
		maxMsgSize:       defaultMaxMsgSize,
		writeTimeout:     defaultWriteTimeout,
		maxMsgPerSecond:  defaultMaxMsgPerSecond,
		maxBytePerSecond: defaultMaxBytePerSecond,
	}
	if config == nil {
		return &cfg
	}
	if config.HeartbeatTimeout > 0 {
		cfg.heartbeatTimeout = config.HeartbeatTimeout
	}
	if config.SendQueueSize > 0 {
		cfg.sendQueueSize = config.SendQueueSize
	}
	if config.MaxMsgSize > 0 {
		cfg.maxMsgSize = config.MaxMsgSize
	}
	if config.WriteTimeout > 0 {
		cfg.writeTimeout = config.WriteTimeout
	}
	if config.MaxMsgPerSecond > 0 {
		cfg.maxMsgPerSecond = config.MaxMsgPerSecond
	}
	if config.MaxBytePerSecond > 0 {
		cfg.maxBytePerSecond = config.MaxBytePerSecond
	}
	return &cfg
}

func normalizeListenAddr(rawAddr string) string {
	if rawAddr == "" {
		return ":0"
	}
	if parsed, err := url.Parse(rawAddr); err == nil && parsed.Host != "" {
		if _, port, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil && port != "" {
			return "0.0.0.0:" + port
		}
	}
	if strings.HasPrefix(rawAddr, ":") {
		return "0.0.0.0" + rawAddr
	}
	if _, port, err := net.SplitHostPort(rawAddr); err == nil && port != "" {
		return "0.0.0.0:" + port
	}
	lastColon := strings.LastIndex(rawAddr, ":")
	if lastColon >= 0 && lastColon < len(rawAddr)-1 {
		return "0.0.0.0:" + rawAddr[lastColon+1:]
	}
	return rawAddr
}
