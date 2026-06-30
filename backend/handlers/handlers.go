package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"plaza/config"
	"plaza/models"
)

type Handler struct {
	Plaza  *models.Plaza
	Config config.PlazaConfig
	Store  models.Store
}

func New(plaza *models.Plaza, cfg config.PlazaConfig, store models.Store) *Handler {
	return &Handler{Plaza: plaza, Config: cfg, Store: store}
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":              h.Config.Title,
		"domain":             h.Config.Domain,
		"allow_anonymous":    h.Config.AllowAnonymous,
		"max_users_per_banca": h.Config.MaxUsersPerBench,
	})
}

func (h *Handler) ListBancas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	bancas := h.Plaza.GetBancas()
	if bancas == nil {
		bancas = []*models.Banca{}
	}
	json.NewEncoder(w).Encode(bancas)
}

func (h *Handler) CreateBanca(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	title := Sanitize(req.Title, 100)
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}

	b := h.Plaza.CreateBanca(title, h.Config.MaxUsersPerBench)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h *Handler) GetBanca(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidID(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	b, ok := h.Plaza.GetBanca(id)
	if !ok {
		http.Error(w, "banca not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// ── Agent discovery ──

func (h *Handler) WellKnownPlaza(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	wsProto := "ws"
	if r.TLS != nil {
		wsProto = "wss"
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := h.Config.Domain
	if host == "localhost" {
		host = r.Host
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        h.Config.Title,
		"protocol":    "plaza",
		"version":     "1",
		"description": "Una plaza pública para conversar. Sin algoritmos. Personas reales.",
		"websocket":   fmt.Sprintf("%s://%s/ws/", wsProto, host),
		"api":         fmt.Sprintf("%s://%s/api/", scheme, host),
		"features":    []string{"bancas", "agents"},
		"links": map[string]string{
			"config":       fmt.Sprintf("%s://%s/api/config", scheme, host),
			"bancas":       fmt.Sprintf("%s://%s/api/bancas", scheme, host),
			"robots":       fmt.Sprintf("%s://%s/robots.txt", scheme, host),
			"self":         fmt.Sprintf("%s://%s/.well-known/plaza", scheme, host),
		},
	})
}

func (h *Handler) RobotsTXT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	host := h.Config.Domain
	if host == "localhost" {
		host = r.Host
	}

	fmt.Fprintf(w, `# Plaza — descubrimiento para agentes
# Más info: https://personnn.com/plaza

User-agent: *
Allow: /api/config
Allow: /api/bancas
Allow: /.well-known/
Allow: /plaza
Allow: /banca

Sitemap: http://%s/.well-known/plaza
`, host)
}

// ── Sanitization ──

func Sanitize(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var clean strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			clean.WriteRune(r)
		}
	}
	s = clean.String()
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func SanitizeMessage(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func isValidID(id string) bool {
	if len(id) != 8 {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'f') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
