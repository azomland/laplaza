package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"plaza/agent"
	"plaza/models"
)

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Client struct {
	ID       string
	BancaID  string
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
}

type Agent struct {
	ID       string
	Name     string
	BancaID  string
	Backend  agent.Backend
	Send     chan []byte
	stop     chan struct{}
	h        *Handler
}

type BancaRoom struct {
	mu       sync.RWMutex
	Clients  map[string]*Client
	Agents   map[string]*Agent
	Messages []models.ChatMessage
	MaxMsg   int
}

var (
	bancas   = make(map[string]*BancaRoom)
	bancasMu sync.RWMutex
)

var (
	ipConnections   = make(map[string]int)
	ipConnectionsMu sync.Mutex
)

const (
	maxMessageLen  = 2000
	maxUsernameLen = 24
	maxWSConnsPerIP = 5
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	bancaID := r.PathValue("id")
	if !isValidID(bancaID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, ok := h.Plaza.GetBanca(bancaID); !ok {
		http.Error(w, "banca not found", http.StatusNotFound)
		return
	}

	// Check max connections per IP
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	ipConnectionsMu.Lock()
	if ipConnections[ip] >= maxWSConnsPerIP {
		ipConnectionsMu.Unlock()
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}
	ipConnections[ip]++
	ipConnectionsMu.Unlock()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			domain := h.Config.Domain
			return strings.Contains(origin, domain) ||
				strings.Contains(origin, "localhost") ||
				strings.Contains(origin, "127.0.0.1")
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ipConnectionsMu.Lock()
		ipConnections[ip]--
		ipConnectionsMu.Unlock()
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:      uuid.New().String()[:8],
		BancaID: bancaID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
	}

	bancasMu.Lock()
	room, ok := bancas[bancaID]
	if !ok {
		room = &BancaRoom{
			Clients: make(map[string]*Client),
			Agents:  make(map[string]*Agent),
			MaxMsg:  200,
		}
		bancas[bancaID] = room
	}
	room.Clients[client.ID] = client
	bancasMu.Unlock()

	h.Plaza.JoinBanca(bancaID, client.ID)

	go client.writePump()
	go client.readPump(h)

	client.Send <- encodeMessage("welcome", map[string]string{
		"client_id": client.ID,
		"banca_id":  bancaID,
	})

	if h.Store != nil {
		messages, err := h.Store.LoadMessages(bancaID, 50)
		if err == nil && len(messages) > 0 {
			client.Send <- encodeMessage("history", messages)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump(h *Handler) {
	defer func() {
		// Decrement IP connection count
		ip := c.Conn.RemoteAddr().String()
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
		ipConnectionsMu.Lock()
		if ipConnections[ip] > 0 {
			ipConnections[ip]--
		}
		ipConnectionsMu.Unlock()

		c.Conn.Close()
		h.Plaza.LeaveBanca(c.BancaID, c.ID)
		bancasMu.Lock()
		if room, ok := bancas[c.BancaID]; ok {
			delete(room.Clients, c.ID)
			if len(room.Clients) == 0 {
				for _, ag := range room.Agents {
					close(ag.stop)
				}
				delete(bancas, c.BancaID)
			}
		}
		bancasMu.Unlock()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "message":
			var payload struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}

			content := SanitizeMessage(payload.Content, maxMessageLen)
			if content == "" {
				continue
			}

			chatMsg := models.ChatMessage{
				Username:  c.Username,
				Content:   content,
				Timestamp: time.Now(),
			}

			if h.Store != nil {
				_ = h.Store.SaveMessage(c.BancaID, chatMsg)
			}

			bancasMu.RLock()
			room, ok := bancas[c.BancaID]
			bancasMu.RUnlock()
			if ok {
				room.mu.Lock()
				room.Messages = append(room.Messages, chatMsg)
				if len(room.Messages) > room.MaxMsg {
					room.Messages = room.Messages[len(room.Messages)-room.MaxMsg:]
				}
				room.mu.Unlock()

				broadcast := encodeMessage("message", chatMsg)
				room.mu.RLock()
				for _, cl := range room.Clients {
					select {
					case cl.Send <- broadcast:
					default:
					}
				}
				for _, ag := range room.Agents {
					select {
					case ag.Send <- broadcast:
					default:
					}
				}
				room.mu.RUnlock()
			}

		case "typing":
			bancasMu.RLock()
			room, ok := bancas[c.BancaID]
			bancasMu.RUnlock()
			if ok {
				broadcast := encodeMessage("typing", map[string]interface{}{
					"username":  c.Username,
					"client_id": c.ID,
				})
				room.mu.RLock()
				for _, cl := range room.Clients {
					if cl.ID == c.ID {
						continue
					}
					select {
					case cl.Send <- broadcast:
					default:
					}
				}
				room.mu.RUnlock()
			}

		case "set_username":
			var payload struct {
				Username string `json:"username"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}
			payload.Username = Sanitize(payload.Username, maxUsernameLen)
			if payload.Username == "" {
				payload.Username = "Anónimo"
			}
			c.Username = payload.Username
		}
	}
}

func encodeMessage(msgType string, data interface{}) []byte {
	payload, _ := json.Marshal(data)
	msg := WSMessage{Type: msgType, Payload: payload}
	b, _ := json.Marshal(msg)
	return b
}
