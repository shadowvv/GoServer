package net

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Server struct {
	addr      string
	upgrader  websocket.Upgrader
	OnMessage func(conn *Conn, data []byte)
}

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: 安全策略可加白名单验证
			},
		},
	}
}

func (s *Server) Start() {
	http.HandleFunc("/ws", s.handleWS)
	log.Printf("WebSocket server started at %s\n", s.addr)
	if err := http.ListenAndServe(s.addr, nil); err != nil {
		log.Fatalf("WebSocket ListenAndServe failed: %v", err)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v\n", err)
		return
	}

	c := NewConn(conn)
	go c.ReadLoop(s.OnMessage)
}
