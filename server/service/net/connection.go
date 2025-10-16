package net

import (
	"fmt"
	"github.com/drop/GoServer/server/service/log"
	"github.com/gorilla/websocket"
	"sync"
)

type Conn struct {
	ws     *websocket.Conn
	sendMu sync.Mutex
}

func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{ws: ws}
}

func (c *Conn) ReadLoop(onMessage func(conn *Conn, data []byte)) {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			log.Error(fmt.Sprintf("read error: %v", err), 0, 0, 0)
			return
		}
		onMessage(c, data)
	}
}

func (c *Conn) Send(data []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.ws.WriteMessage(websocket.BinaryMessage, data)
}

func (c *Conn) Close() {
	c.ws.Close()
}
