package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients map[*Client]bool
	*sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool), Mutex: &sync.Mutex{}}
}

func (h *Hub) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Unable to upgrade websocket connection")
		return
	}

	defer conn.Close()

	client := NewClient(conn, h)
	h.AddClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func (h *Hub) AddClient(client *Client) {
	h.Mutex.Lock()
	h.clients[client] = true
	defer h.Mutex.Unlock()
}

func (h *Hub) RemoveClient(client *Client) {
	h.Mutex.Lock()
	client.conn.Close()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
	}
	defer h.Mutex.Unlock()
}
