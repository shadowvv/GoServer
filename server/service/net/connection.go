package net

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 默认心跳/超时
	pingInterval = 25 * time.Second
	pongTimeout  = 60 * time.Second

	// 发送队列大小
	sendQueueSize = 256
)

var ErrConnClosed = errors.New("connection closed")

// Conn 包装 websocket.Conn
type Conn struct {
	conn   *websocket.Conn
	router serviceInterface.RouterInterface

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	sendQueue chan []byte

	// meta 用于保存 session 信息
	meta sync.Map
}

func (c *Conn) OnDisconnect() {
	//TODO implement me
	panic("implement me")
}

// newConn
func newConn(ws *websocket.Conn, router serviceInterface.RouterInterface) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		conn:      ws,
		router:    router,
		ctx:       ctx,
		cancel:    cancel,
		sendQueue: make(chan []byte, sendQueueSize),
	}
	// pong handler updates deadline
	ws.SetReadDeadline(time.Now().Add(pongTimeout))
	ws.SetPongHandler(func(appData string) error {
		ws.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})
	return c
}

// Start 启动 read/write pump 与 heartbeat
func (c *Conn) Start() {
	c.wg.Add(3)
	go c.readPump()
	go c.writePump()
	go c.heartbeat()
}

// Close 关闭连接（可并发调用）
func (c *Conn) Close() {
	c.cancel()
	c.conn.Close()
	c.wg.Wait()
}

// Send 安全发送（非阻塞，队列满时返回 ErrConnClosed 或错误）
func (c *Conn) Send(frame []byte) error {
	select {
	case <-c.ctx.Done():
		return ErrConnClosed
	default:
	}
	select {
	case c.sendQueue <- frame:
		return nil
	case <-c.ctx.Done():
		return ErrConnClosed
	}
}

// readPump: 持续读，解帧后交给 Router 处理
func (c *Conn) readPump() {
	defer func() {
		c.cancel()
		c.wg.Done()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		typ, data, err := c.conn.ReadMessage()
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

		// dispatch (synchronous handler)
		c.router.Dispatch(msgID, c, payload)
	}
}

// writePump: 持续写，合并控制写超时等
func (c *Conn) writePump() {
	defer func() {
		c.cancel()
		c.wg.Done()
	}()

	for {
		select {
		case <-c.ctx.Done():
			// flush remaining? 忽略
			return
		case data := <-c.sendQueue:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				// log.Printf("write message error: %v", err)
				return
			}
		}
	}
}

// heartbeat: 定期发送 Ping，检查 Pong 更新的 read deadline
func (c *Conn) heartbeat() {
	defer func() {
		c.cancel()
		c.wg.Done()
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// 如果对端没有 pong，会在 read deadline 超时导致 readPump 退出
		}
	}
}

// Meta 操作
func (c *Conn) SetMeta(k string, v interface{})      { c.meta.Store(k, v) }
func (c *Conn) GetMeta(k string) (interface{}, bool) { return c.meta.Load(k) }
func (c *Conn) DelMeta(k string)                     { c.meta.Delete(k) }

// Utility：将 msgID + payload 封包
func Pack(msgID uint32, payload []byte) []byte {
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], msgID)
	copy(frame[4:], payload)
	return frame
}
