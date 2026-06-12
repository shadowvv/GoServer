package netService

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"

	"github.com/gorilla/websocket"
)

var pongWait = 30 * time.Second

type Session struct {
	id   int64
	conn *websocket.Conn
	meta sync.Map

	cfg sessionRuntimeConfig

	acceptor serviceInterface.AcceptorInterface
	codec    serviceInterface.CodecInterface
	router   serviceInterface.RouterInterface

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	sendQueue chan []byte

	lastActiveTime atomic.Int64
	active         atomic.Bool

	// Flood control
	lastMsgSec int64
	msgCount   int32
	msgBytes   int32
}

var _ serviceInterface.SessionInterface = (*Session)(nil)

func newSession(ws *websocket.Conn, router serviceInterface.RouterInterface, codec serviceInterface.CodecInterface, id int64, acceptor serviceInterface.AcceptorInterface, cfg sessionRuntimeConfig) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		id:        id,
		conn:      ws,
		cfg:       cfg,
		acceptor:  acceptor,
		codec:     codec,
		router:    router,
		ctx:       ctx,
		cancel:    cancel,
		sendQueue: make(chan []byte, int(cfg.sendQueueSize)),
	}
	s.lastActiveTime.Store(tool.UnixNowMilli())
	s.active.Store(true)
	logger.InfoWithSprintf("[net] create new sessionId:%d", s.id)
	return s
}

func (s *Session) Start() {
	s.conn.SetReadLimit(int64(s.cfg.maxMsgSize))
	_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go s.readPump()
	go s.writePump()
	go s.heartbeat()
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.cancel()
		s.active.Store(false)
		s.acceptor.OnSessionClose(s, false)
		_ = s.conn.Close()
	})
}

func (s *Session) Send(msgId int32, message proto.Message) {
	if !s.IsActive() {
		logger.ErrorBySprintf("[net] send message sessionId:%d is closed. msgId:%d", s.id, msgId)
		return
	}

	frame, err := s.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorBySprintf("[net] send messageId:%d sessionId:%d marshal error: %v", msgId, s.id, err)
		return
	}
	if int32(len(frame)) > s.cfg.maxMsgSize {
		logger.ErrorBySprintf("[net] sessionId:%d send messageId:%d msg too large: %d", s.id, msgId, len(frame))
		return
	}

	select {
	case s.sendQueue <- frame:
		return
	default:
		logger.ErrorBySprintf("[net] sessionId:%d send messageId:%d queue full", s.id, msgId)
	}
}

func (s *Session) SendAndClose(msgId int32, message proto.Message) {
	if !s.IsActive() {
		logger.ErrorBySprintf("[net] send message sessionId:%d is closed. msgId:%d", s.id, msgId)
		return
	}

	frame, err := s.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorBySprintf("[net] send messageId:%d sessionId:%d marshal error: %v", msgId, s.id, err)
		return
	}
	if int32(len(frame)) > s.cfg.maxMsgSize {
		logger.ErrorBySprintf("[net] sessionId:%d send messageId:%d msg too large: %d", s.id, msgId, len(frame))
		return
	}

	select {
	case s.sendQueue <- frame:
		s.SetMeta("close", true)
		return
	default:
		logger.ErrorBySprintf("[net] sessionId:%d send messageId:%d queue full", s.id, msgId)
	}
}

// 含洪水检测
func (s *Session) readPump() {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorBySprintf("[net] readPump panic: %v,sessionId:%d", r, s.id)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		typ, data, err := s.conn.ReadMessage()
		if err != nil {
			logger.ErrorBySprintf("[net] ws read message sessionId:%d, error: %v", s.GetID(), err)
			s.Close()
			return
		}
		if typ != websocket.BinaryMessage {
			logger.ErrorBySprintf("[net] non-binary message type=%d sessionId=%d", typ, s.id)
			continue
		}
		if len(data) < 4 {
			logger.ErrorBySprintf("[net] msg too short sessionId:%d", s.id)
			continue
		}

		// Flood control
		nowSec := tool.UnixNow()
		if s.lastMsgSec != nowSec {
			s.lastMsgSec = nowSec
			s.msgCount = 0
			s.msgBytes = 0
		}
		s.msgCount++
		s.msgBytes += int32(len(data))
		if s.msgCount > s.cfg.maxMsgPerSecond || s.msgBytes > s.cfg.maxBytePerSecond {
			logger.ErrorBySprintf("[net] sessionId:%d flood detect cnt=%d bytes=%d", s.id, s.msgCount, s.msgBytes)
			s.Close()
			return
		}

		msgID := binary.BigEndian.Uint32(data[:4])

		msg := s.router.GetMessage(int32(msgID))
		if msg == nil {
			logger.ErrorBySprintf("[net] read msg sessionId:%d unknown msgId:%d", s.id, msgID)
			continue
		}

		if err = s.codec.Unmarshal(data, msg); err != nil {
			logger.ErrorBySprintf("[net] read msg sessionId:%d msgId:%d unmarshal error: %v", s.id, msgID, err)
			continue
		}
		s.router.Dispatch(s, int32(msgID), msg)
		s.lastActiveTime.Store(tool.UnixNowMilli())
	}
}

// 批量写 + pool + panic recovery
func (s *Session) writePump() {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorBySprintf("[net] writePump panic: %v,sessionId:%d", r, s.id)
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case data := <-s.sendQueue:
			err := s.conn.SetWriteDeadline(time.Now().Add(s.cfg.writeTimeout))
			if err != nil {
				logger.ErrorBySprintf("[net] ws set write deadline sessionId:%d, error: %v", s.GetID(), err)
				s.Close()
				return
			}
			if err = s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				logger.ErrorBySprintf("[net] ws write message sessionId:%d, error: %v", s.GetID(), err)
				s.Close()
				return
			}
			if meta, ok := s.GetMeta("close"); ok && meta.(bool) {
				logger.InfoWithSprintf("[net] sessionId:%d close", s.id)
				s.Close()
				return
			}
		case <-ticker.C:
			if err := s.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(s.cfg.writeTimeout)); err != nil {
				logger.ErrorBySprintf("[net] ws ping sessionId:%d, error: %v", s.GetID(), err)
				s.Close()
				return
			}
		}
	}
}

func (s *Session) heartbeat() {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorBySprintf("[net] heartbeat panic: %v, sessionId:%d", r, s.id)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			inactive := time.Duration(tool.UnixNowMilli()-s.lastActiveTime.Load()) * time.Millisecond
			if inactive > s.cfg.heartbeatTimeout {
				logger.InfoWithSprintf("[net] sessionId:%d timeout", s.id)
				s.Close()
				return
			}
		}
	}
}

func (s *Session) IsActive() bool { return s.active.Load() }

func (s *Session) GetID() int64 { return s.id }

func (s *Session) SetMeta(k string, v interface{}) { s.meta.Store(k, v) }

func (s *Session) GetMeta(k string) (interface{}, bool) { return s.meta.Load(k) }

func (s *Session) DelMeta(k string) { s.meta.Delete(k) }
