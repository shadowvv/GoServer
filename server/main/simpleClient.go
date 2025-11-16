package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/drop/GoServer/server/logic/pb"
)

const (
	serverAddr   = "ws://127.0.0.1:8080/ws"
	pingInterval = 10 * time.Second
	pongWait     = 15 * time.Second
)

type WSClient struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
	ctx    context.Context
	once   sync.Once // 防止重复关闭
}

func NewWSClient(addr string) (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	client := &WSClient{conn: conn, ctx: ctx, cancel: cancel}

	// 初始化 read deadline
	conn.SetReadDeadline(time.Now().Add(pongWait))

	// 设置 pong handler
	conn.SetPongHandler(func(appData string) error {
		log.Println("[client] pong received")
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	return client, nil
}

func (c *WSClient) Start() {
	go c.readLoop()
	go c.heartbeat()
}

func (c *WSClient) Stop() {
	c.once.Do(func() {
		c.cancel()
		_ = c.conn.Close()
		log.Println("[client] connection closed")
	})
}

func (c *WSClient) readLoop() {
	defer c.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("[client] read error:", err)
			return
		}

		if msgType != websocket.BinaryMessage {
			continue
		}
		if len(data) < 4 {
			continue
		}

		msgID := binary.BigEndian.Uint32(data[:4])
		payload := data[4:]
		log.Printf("[client] recv msgID=%d len=%d", msgID, len(payload))

		switch msgID {
		case 2001: // 服务器返回LoginResp
			resp := &pb.LoginResp{}
			if err := proto.Unmarshal(payload, resp); err != nil {
				log.Println("[client] unmarshal error:", err)
				continue
			}
			log.Printf("[client] LoginResp: %+v", resp)
		default:
			log.Printf("[client] unknown msgID=%d", msgID)
		}
	}
}

func (c *WSClient) heartbeat() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	// 延迟第一次 ping，防止刚连上就发心跳
	time.Sleep(2 * time.Second)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("[client] ping error:", err)
				c.Stop()
				return
			}
			log.Println("[client] ping sent")
		}
	}
}

func (c *WSClient) Send(msgID uint32, pbMsg proto.Message) error {
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		return err
	}
	frame := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(frame[:4], msgID)
	copy(frame[4:], data)
	return c.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func main() {
	client, err := NewWSClient(serverAddr)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer client.Stop()

	client.Start()

	// 发送登录请求
	loginReq := &pb.LoginReq{
		Account: "test",
		Token:   "123456",
	}
	err = client.Send(1001, loginReq)
	if err != nil {
		log.Println("send error:", err)
	}

	// 阻塞等待 Ctrl+C 退出
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("client stopped")
}
