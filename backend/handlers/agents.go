package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"plaza/agent"
	"plaza/models"
)

type agentRequest struct {
	Backend string `json:"backend"`
	Name    string `json:"name"`
}

func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	bancaID := r.PathValue("id")
	if !isValidID(bancaID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	bancasMu.RLock()
	room, ok := bancas[bancaID]
	bancasMu.RUnlock()

	if !ok {
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	type agentInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Backend string `json:"backend"`
	}
	list := make([]agentInfo, 0, len(room.Agents))
	for _, ag := range room.Agents {
		list = append(list, agentInfo{
			ID:      ag.ID,
			Name:    ag.Name,
			Backend: ag.Backend.Name(),
		})
	}
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) InviteAgent(w http.ResponseWriter, r *http.Request) {
	bancaID := r.PathValue("id")
	if !isValidID(bancaID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, ok := h.Plaza.GetBanca(bancaID); !ok {
		http.Error(w, "banca not found", http.StatusNotFound)
		return
	}

	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	name := Sanitize(req.Name, 24)
	if name == "" {
		name = "Acompañante"
	}

	var back agent.Backend
	switch req.Backend {
	case "echo":
		back = &agent.EchoBackend{}
	default:
		http.Error(w, "unknown backend: "+req.Backend, http.StatusBadRequest)
		return
	}

	ag := &Agent{
		ID:       uuid.New().String()[:8],
		Name:     name,
		BancaID:  bancaID,
		Backend:  back,
		Send:     make(chan []byte, 256),
		stop:     make(chan struct{}),
		h:        h,
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
	room.Agents[ag.ID] = ag
	bancasMu.Unlock()

	go ag.start()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"agent_id": ag.ID,
		"name":     ag.Name,
	})
}

func (h *Handler) RemoveAgent(w http.ResponseWriter, r *http.Request) {
	bancaID := r.PathValue("id")
	agentID := r.PathValue("agent_id")

	if !isValidID(bancaID) || !isValidID(agentID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	bancasMu.Lock()
	room, ok := bancas[bancaID]
	if !ok {
		bancasMu.Unlock()
		http.Error(w, "banca not found", http.StatusNotFound)
		return
	}

	ag, ok := room.Agents[agentID]
	if !ok {
		bancasMu.Unlock()
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	close(ag.stop)
	delete(room.Agents, agentID)
	bancasMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (a *Agent) start() {
	log.Printf("🤖 agent %s (%s) entró a banca %s", a.Name, a.Backend.Name(), a.BancaID)

	h := a.h

	h.broadcastSystem(a.BancaID, a.Name+" se sentó en la banca.")

	ctx := context.Background()

	for {
		select {
		case <-a.stop:
			h.broadcastSystem(a.BancaID, a.Name+" se levantó de la banca.")
			log.Printf("🤖 agent %s salió de banca %s", a.Name, a.BancaID)
			return

		case msgBytes, ok := <-a.Send:
			if !ok {
				return
			}

			var msg WSMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}
			if msg.Type != "message" {
				continue
			}

			var payload struct {
				Content  string `json:"content"`
				Username string `json:"username"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}
			if payload.Username == a.Name {
				continue
			}

			select {
			case <-a.stop:
				return
			case <-time.After(500 * time.Millisecond):
			}

			response, err := a.Backend.Generate(ctx, roomHistory(a.BancaID), payload.Content)
			if err != nil {
				log.Printf("🤖 agent %s error: %v", a.Name, err)
				continue
			}

			chatMsg := models.ChatMessage{
				Username:  a.Name,
				Content:   response,
				Timestamp: time.Now(),
			}

			if h.Store != nil {
				_ = h.Store.SaveMessage(a.BancaID, chatMsg)
			}

			bancasMu.RLock()
			room, exists := bancas[a.BancaID]
			bancasMu.RUnlock()

			if exists {
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
				for _, ag2 := range room.Agents {
					if ag2.ID == a.ID {
						continue
					}
					select {
					case ag2.Send <- broadcast:
					default:
					}
				}
				room.mu.RUnlock()
			}
		}
	}
}

func (h *Handler) broadcastSystem(bancaID string, text string) {
	bancasMu.RLock()
	room, ok := bancas[bancaID]
	bancasMu.RUnlock()
	if !ok {
		return
	}

	msg := encodeMessage("system", map[string]string{
		"text": text,
	})

	room.mu.RLock()
	defer room.mu.RUnlock()
	for _, cl := range room.Clients {
		select {
		case cl.Send <- msg:
		default:
		}
	}
}

func roomHistory(bancaID string) []models.ChatMessage {
	bancasMu.RLock()
	room, ok := bancas[bancaID]
	bancasMu.RUnlock()
	if !ok {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	cp := make([]models.ChatMessage, len(room.Messages))
	copy(cp, room.Messages)
	return cp
}


