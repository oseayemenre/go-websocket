package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"sync"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = pongWait * 9 / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Hub struct {
	clients map[*Client]bool
	mutex   *sync.Mutex
}

type Client struct {
	conn   *websocket.Conn
	hub    *Hub
	egress chan []byte
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool), mutex: &sync.Mutex{}}
}

func NewClient(ws *websocket.Conn, hub *Hub) *Client {
	return &Client{conn: ws, hub: hub, egress: make(chan []byte)}
}

func (h *Hub) addClient(c *Client) {
	h.mutex.Lock()
	h.clients[c] = true
	defer h.mutex.Unlock()
}

func (h *Hub) removeClient(c *Client) {
	h.mutex.Lock()

	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
	}

	defer h.mutex.Unlock()
}

func (c *Client) readMessage() {
	defer func() {
		log.Println("Hit read clean up")
		c.hub.removeClient(c)
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(pongMsg string) error {
		log.Println("pong")
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("connection closed: %s", err.Error())
			}
			break
		}

		log.Println(string(message))

		for client := range c.hub.clients {
			client.egress <- message
		}
	}
}

func (c *Client) writeMessage() {

	defer func() {
		log.Println("Hit write clean up")
		c.hub.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("connection closed: %s", err.Error())
				}
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				continue
			}

			log.Printf("Message sent")

		case <-ticker.C:
			log.Println("ping")
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				log.Printf("error: %s", err.Error())
				return
			}
		}
	}
}

func (h *Hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Unable to upgrade websocket connection")
		return
	}

	client := NewClient(conn, h)

	h.addClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	hub := NewHub()

	r.HandleFunc("/ws", hub.wsHandler)

	log.Println("Server is listening on PORT 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Unable to start server: %s", err.Error())
	}
}
