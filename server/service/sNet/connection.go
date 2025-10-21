package sNet

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

const (
	// 默认心跳/超时
	pingInterval = 25 * time.Second
	pongTimeout  = 60 * time.Second

	// 发送队列大小
	sendQueueSize = 256
)

// Conn 包装 websocket.Conn
type Conn struct {
	id   int64
	conn *websocket.Conn
	meta sync.Map

	codec  serviceInterface.CodecInterface
	router serviceInterface.RouterInterface

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	sendQueue chan []byte
}

// newConn
func newConn(ws *websocket.Conn, router serviceInterface.RouterInterface, codec serviceInterface.CodecInterface, id int64) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		id:        id,
		conn:      ws,
		codec:     codec,
		router:    router,
		ctx:       ctx,
		cancel:    cancel,
		sendQueue: make(chan []byte, sendQueueSize),
	}
	// pong handler updates deadline
	err := ws.SetReadDeadline(time.Now().Add(pongTimeout))
	if err != nil {
		logger.Error(fmt.Sprintf("[net] ws set read deadline connectionId:%d, error: %v", id, err))
		return nil
	}
	ws.SetPongHandler(func(appData string) error {
		err := ws.SetReadDeadline(time.Now().Add(pongTimeout))
		if err != nil {
			logger.Error(fmt.Sprintf("[net] ws set pong read deadline connectionId:%d, error: %v", id, err))
			return err
		}
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
	err := c.conn.Close()
	if err != nil {
		logger.Error(fmt.Sprintf("[net] ws close connectionId:%d, error: %v", c.GetID(), err))
		return
	}
	c.wg.Wait()
}

// Send 安全发送（非阻塞，队列满时返回 ErrConnClosed 或错误）
func (c *Conn) Send(message proto.Message) error {
	frame, err := c.codec.Marshal(message)
	if err != nil {
		return err
	}
	select {
	case <-c.ctx.Done():
		return errors.New(fmt.Sprintf("[net] connectionId:%d closed", c.GetID()))
	default:
	}
	select {
	case c.sendQueue <- frame:
		return nil
	case <-c.ctx.Done():
		return errors.New(fmt.Sprintf("[net] connectionId:%d closed", c.GetID()))
	}
}

func (c *Conn) HandleConnectionClosed() {
	//TODO implement me
	panic("implement me")
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

		msg := c.router.GetMessage(msgID)
		if msg == nil {
			logger.Error(fmt.Sprintf("[net] connectionId:%d unknown message type: %d", c.GetID(), msgID))
			continue
		}
		err = c.codec.Unmarshal(payload, msg)
		if err != nil {
			logger.Error(fmt.Sprintf("[net] connectionId:%d unmarshal error: %v", c.GetID(), err))
			continue
		}
		c.router.Dispatch(msgID, msg)
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
			err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				logger.Error(fmt.Sprintf("[net] ws set write deadline connectionId:%d, error: %v", c.GetID(), err))
				return
			}
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
			err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				logger.Error(fmt.Sprintf("[net] ws set write deadline connectionId:%d, error: %v", c.GetID(), err))
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// 如果对端没有 pong，会在 read deadline 超时导致 readPump 退出
		}
	}
}

func (c *Conn) GetID() int64 {
	return c.id
}

// Meta 操作
func (c *Conn) SetMeta(k string, v interface{})      { c.meta.Store(k, v) }
func (c *Conn) GetMeta(k string) (interface{}, bool) { return c.meta.Load(k) }
func (c *Conn) DelMeta(k string)                     { c.meta.Delete(k) }
