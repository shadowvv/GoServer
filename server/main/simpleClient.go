package main

import (
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

	// 发送一条测试消息
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello, Server!"))
	if err != nil {
		log.Println("write:", err)
		return
	}

	// 等待一段时间以查看响应
	time.Sleep(5 * time.Second)
}
