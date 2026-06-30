package models

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Banca struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Users     int       `json:"users"`
	MaxUsers  int       `json:"max_users"`
	Active    bool      `json:"active"`
}

type ChatMessage struct {
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Plaza struct {
	mu      sync.RWMutex
	Bancas  map[string]*Banca
	Clients map[string]map[string]bool
}

func NewPlaza() *Plaza {
	return &Plaza{
		Bancas:  make(map[string]*Banca),
		Clients: make(map[string]map[string]bool),
	}
}

func randomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (p *Plaza) CreateBanca(title string, maxUsers int) *Banca {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := randomID()
	b := &Banca{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
		MaxUsers:  maxUsers,
		Active:    true,
	}
	p.Bancas[id] = b
	p.Clients[id] = make(map[string]bool)
	return b
}

func (p *Plaza) GetBancas() []*Banca {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var list []*Banca
	for _, b := range p.Bancas {
		if b.Active {
			list = append(list, b)
		}
	}
	return list
}

func (p *Plaza) GetBanca(id string) (*Banca, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	b, ok := p.Bancas[id]
	return b, ok
}

func (p *Plaza) JoinBanca(bancaID string, clientID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	b, ok := p.Bancas[bancaID]
	if !ok || !b.Active {
		return false
	}
	if len(p.Clients[bancaID]) >= b.MaxUsers {
		return false
	}
	p.Clients[bancaID][clientID] = true
	b.Users = len(p.Clients[bancaID])
	return true
}

func (p *Plaza) LeaveBanca(bancaID string, clientID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.Clients[bancaID]; ok {
		delete(p.Clients[bancaID], clientID)
		if b, ok := p.Bancas[bancaID]; ok {
			b.Users = len(p.Clients[bancaID])
			if b.Users == 0 {
				b.Active = false
				delete(p.Bancas, bancaID)
				delete(p.Clients, bancaID)
			}
		}
	}
}
