package main

import (
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	conn   *websocket.Conn
	hub    *Hub
	egress chan []byte
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{conn: conn, hub: hub, egress: make(chan []byte)}
}

func (c *Client) readMessage() {
	defer c.hub.RemoveClient(c)

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %s", err.Error())
			}
			break
		}

		for client := range c.hub.clients {
			client.egress <- message
		}

		log.Println(message)
	}
}

func (c *Client) writeMessage() {
	defer c.hub.RemoveClient(c)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("close connection: %s", err.Error())
				}
				break
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("failed to send message: %s", err.Error())
			}

			log.Println("Message sent")
		}
	}
}