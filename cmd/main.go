package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

type Server struct {
	conn map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{
		conn: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	mutex := new(sync.Mutex)

	fmt.Printf("New incoming connection from client: %s\n", ws.RemoteAddr())

	mutex.Lock()
	s.conn[ws] = true
	mutex.Unlock()

	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 1024)

	for {
		n, err := ws.Read(buf)

		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		msg := buf[:n]

		s.broadCast(msg)
	}
}

func (s *Server) broadCast(b []byte) {
	for ws := range s.conn {
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(b); err != nil {
				fmt.Println(err)
			}
		}(ws)
	}
}

func main() {
	server := NewServer()

	http.Handle("/ws", websocket.Handler(server.handleWS))
	fmt.Println("Server is running on 3000...")
	http.ListenAndServe(":3000", nil)
}
