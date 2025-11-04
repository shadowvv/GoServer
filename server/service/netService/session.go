package netService

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Session 包装 websocket.Session
type Session struct {
	id   int64
	conn *websocket.Conn
	meta sync.Map

	acceptor serviceInterface.AcceptorInterface
	codec    serviceInterface.CodecInterface
	router   serviceInterface.RouterInterface

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	sendQueue chan []byte
}

var _ serviceInterface.SessionInterface = (*Session)(nil)

// newSession
func newSession(ws *websocket.Conn, router serviceInterface.RouterInterface, codec serviceInterface.CodecInterface, id int64, acceptor serviceInterface.AcceptorInterface) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		id:        id,
		conn:      ws,
		acceptor:  acceptor,
		codec:     codec,
		router:    router,
		ctx:       ctx,
		cancel:    cancel,
		sendQueue: make(chan []byte, sendQueueSize),
	}
	// pong handler updates deadline
	//err := ws.SetReadDeadline(time.Now().Add(time.Duration(pongTimeout)))
	//if err != nil {
	//	logger.Error(fmt.Sprintf("[net] ws set read deadline connectionId:%d, error: %v", id, err))
	//	return nil
	//}
	//ws.SetPongHandler(func(appData string) error {
	//	err := ws.SetReadDeadline(time.Now().Add(time.Duration(pongTimeout)))
	//	if err != nil {
	//		logger.Error(fmt.Sprintf("[net] ws set pong read deadline connectionId:%d, error: %v", id, err))
	//		return err
	//	}
	//	return nil
	//})
	return s
}

// Start 启动 read/write pump 与 heartbeat
func (s *Session) Start() {
	s.wg.Add(3)
	go s.readPump()
	go s.writePump()
	go s.heartbeat()
}

// Close 关闭连接（可并发调用）
func (s *Session) Close() {
	s.cancel()
	err := s.conn.Close()
	if err != nil {
		logger.Error(fmt.Sprintf("[net] ws close connectionId:%d, error: %v", s.GetID(), err))
		return
	}
	s.wg.Wait()
}

// Send 安全发送（非阻塞，队列满时返回 ErrConnClosed 或错误）
func (s *Session) Send(msgId int32, message proto.Message) error {
	frame, err := s.codec.Marshal(msgId, message)
	if err != nil {
		return err
	}
	select {
	case <-s.ctx.Done():
		return errors.New(fmt.Sprintf("[net] connectionId:%d closed", s.GetID()))
	default:
	}
	select {
	case s.sendQueue <- frame:
		return nil
	case <-s.ctx.Done():
		return errors.New(fmt.Sprintf("[net] connectionId:%d closed", s.GetID()))
	}
}

// readPump: 持续读，解帧后交给 Router 处理
func (s *Session) readPump() {
	defer func() {
		s.cancel()
		s.wg.Done()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		typ, data, err := s.conn.ReadMessage()
		if err != nil {
			// log.Printf("read message error: %v", err)
			return
		}
		if typ != websocket.BinaryMessage && typ != websocket.TextMessage {
			continue
		}

		// 解析帧（简单格式：4字节msgID + payload）
		if len(data) < 4 {
			// invalid frame, ignore
			continue
		}
		msgID := binary.BigEndian.Uint32(data[:4])
		payload := data[4:]

		msg := s.router.GetMessage(int32(msgID))
		if msg == nil {
			logger.Error(fmt.Sprintf("[net] connectionId:%d unknown message type: %d", s.GetID(), msgID))
			continue
		}
		err = s.codec.Unmarshal(payload, msg)
		if err != nil {
			logger.Error(fmt.Sprintf("[net] connectionId:%d unmarshal error: %v", s.GetID(), err))
			continue
		}
		s.router.Dispatch(s, int32(msgID), msg)
	}
}

// writePump: 持续写，合并控制写超时等
func (s *Session) writePump() {
	defer func() {
		s.cancel()
		s.wg.Done()
	}()

	for {
		select {
		case <-s.ctx.Done():
			// flush remaining? 忽略
			return
		case data := <-s.sendQueue:
			err := s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				logger.Error(fmt.Sprintf("[net] ws set write deadline connectionId:%d, error: %v", s.GetID(), err))
				return
			}
			if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				// log.Printf("write message error: %v", err)
				return
			}
		}
	}
}

// heartbeat: 定期发送 Ping，检查 Pong 更新的 read deadline
func (s *Session) heartbeat() {
	defer func() {
		s.cancel()
		s.wg.Done()
	}()

	ticker := time.NewTicker(time.Duration(pingInterval))
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			//err := s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			//if err != nil {
			//	logger.Error(fmt.Sprintf("[net] ws set write deadline connectionId:%d, error: %v", s.GetID(), err))
			//	s.acceptor.OnConnectionTimeout(s)
			//	return
			//}
			//if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			//	return
			//}
			// 如果对端没有 pong，会在 read deadline 超时导致 readPump 退出
		}
	}
}

func (s *Session) GetID() int64 {
	return s.id
}

// Meta 操作
func (s *Session) SetMeta(k string, v interface{})      { s.meta.Store(k, v) }
func (s *Session) GetMeta(k string) (interface{}, bool) { return s.meta.Load(k) }
func (s *Session) DelMeta(k string)                     { s.meta.Delete(k) }
