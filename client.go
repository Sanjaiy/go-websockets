package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second
	pingInterval = ( pongWait * 9 ) / 10
)

type ClientList map[*Client]bool

type Client struct {
	conn *websocket.Conn
	manager *Manager

	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		conn: conn,
		manager: manager,
		egress: make(chan Event),
	}
}

func (c *Client) ReadMessages() {
	defer func ()  {
		c.manager.RemoveClient(c)	
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	// Jumbo frames to avoid clients sending large amount of data
	c.conn.SetReadLimit(512)

	c.conn.SetPongHandler(c.pongHandler)

	for {
		_, payload, err :=  c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Error reading message: ", err)
			}
			break
		}
		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Println("Error parsing message: ", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("Error handling message: ", err)
		}
	}
}

func (c *Client) WriteMessages() {
	defer func() {
		c.manager.RemoveClient(c)	
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <- c.egress:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("Connection closed: ", err)
					return
				}
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("Failed to send message: ", err)
			}
		case <- ticker.C:
			log.Println("Ping")
			
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("ping error: ", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("Pong")
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}
