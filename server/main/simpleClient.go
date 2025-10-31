package main

import (
	"encoding/binary"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 连接到 WebSocket 服务器
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// 启动一个 goroutine 来接收消息
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	req := &pb.LoginReq{
		Account: "test",
		Token:   "123456",
	}
	data, err := proto.Marshal(req)
	if err != nil {
		log.Println("marshal:", err)
		return
	}
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, uint32(1001)) // 大端序（按需切换）
	data = append(bytes, data...)
	// 发送一条测试消息
	err = conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("write:", err)
		return
	}

	// 等待一段时间以查看响应
	time.Sleep(5 * time.Second)
}
