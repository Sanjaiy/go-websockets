package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	webSocketUpgrader = websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
		CheckOrigin: checkOrigin,
	}
)

type Manager struct {
	clients ClientList
	mtx sync.RWMutex
	handlers map[string]EventHandler
	tokens RetentionMap
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients: make(ClientList),
		handlers: make(map[string]EventHandler),
		tokens: NewRetentionMap(ctx, 5 * time.Second),
	}

	m.SetupHandlers()

	return m
}

func (m *Manager) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.tokens.VerifyToken(token) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("New Connection")

	// Upgrage regular http connection into websocket
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)
	m.AddClient(client)

	go client.ReadMessages()
	go client.WriteMessages()
}

func (m *Manager) AddClient(client *Client) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.clients[client] = true
}

func (m *Manager) RemoveClient(client *Client) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, ok := m.clients[client]; ok {
		client.conn.Close()
		delete(m.clients, client)
	}
}

func (m *Manager) SetupHandlers() {
	m.handlers[EventSendMessage] = SendMessage
}

func SendMessage(event Event, c *Client) error {
	fmt.Println(event)
	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("no handler for event type: %s", event.Type)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}

func (m *Manager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`	
	}
	
	var req userLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "sanjaiy" && req.Password == "SK@123" {
		type reponse struct {
			Token string `json:"token"`
		}

		token := m.tokens.NewToken()

		resp := reponse{
			Token: token.Key,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error parsing token", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
}