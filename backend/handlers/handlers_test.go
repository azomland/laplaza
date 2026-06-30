package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plaza/config"
	"plaza/models"
)

func newTestHandler() *Handler {
	plaza := models.NewPlaza()
	cfg := config.Default()
	return New(plaza, cfg, nil)
}

func TestGetConfig(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()

	h.GetConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if body["title"] != "Mi Plaza" {
		t.Errorf("expected 'Mi Plaza', got %v", body["title"])
	}
	if body["max_users_per_banca"] != float64(33) {
		t.Errorf("expected 33, got %v", body["max_users_per_banca"])
	}
}

func TestListBancasEmpty(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/bancas", nil)
	w := httptest.NewRecorder()

	h.ListBancas(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var bancas []*models.Banca
	json.NewDecoder(resp.Body).Decode(&bancas)

	if len(bancas) != 0 {
		t.Errorf("expected empty list, got %d", len(bancas))
	}
}

func TestCreateBanca(t *testing.T) {
	h := newTestHandler()
	body := `{"title":"Nueva Banca"}`
	req := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateBanca(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var banca models.Banca
	json.NewDecoder(resp.Body).Decode(&banca)

	if banca.Title != "Nueva Banca" {
		t.Errorf("expected 'Nueva Banca', got %q", banca.Title)
	}
	if banca.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(banca.ID) != 8 {
		t.Errorf("expected 8-char ID, got %d", len(banca.ID))
	}
}

func TestCreateBancaEmptyTitle(t *testing.T) {
	h := newTestHandler()
	body := `{"title":""}`
	req := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateBanca(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateAndList(t *testing.T) {
	h := newTestHandler()

	body := `{"title":"Banca 1"}`
	req := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateBanca(w, req)

	req2 := httptest.NewRequest("GET", "/api/bancas", nil)
	w2 := httptest.NewRecorder()
	h.ListBancas(w2, req2)

	var bancas []*models.Banca
	json.NewDecoder(w2.Result().Body).Decode(&bancas)

	if len(bancas) != 1 {
		t.Fatalf("expected 1 banca, got %d", len(bancas))
	}
	if bancas[0].Title != "Banca 1" {
		t.Errorf("expected 'Banca 1', got %q", bancas[0].Title)
	}
}

func TestGetBanca(t *testing.T) {
	h := newTestHandler()

	creq := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(`{"title":"find me"}`))
	creq.Header.Set("Content-Type", "application/json")
	cw := httptest.NewRecorder()
	h.CreateBanca(cw, creq)

	var created models.Banca
	json.NewDecoder(cw.Result().Body).Decode(&created)

	greq := httptest.NewRequest("GET", "/api/bancas/"+created.ID, nil)
	greq.SetPathValue("id", created.ID)
	gw := httptest.NewRecorder()
	h.GetBanca(gw, greq)

	if gw.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", gw.Result().StatusCode)
	}

	var got models.Banca
	json.NewDecoder(gw.Result().Body).Decode(&got)
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
}

func TestGetBancaNotFound(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/bancas/ffffffff", nil)
	req.SetPathValue("id", "ffffffff")
	w := httptest.NewRecorder()

	h.GetBanca(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Result().StatusCode)
	}
}

func TestCreateBancaInvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateBanca(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestCreateBancaWrongMethod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/bancas", nil)
	w := httptest.NewRecorder()

	h.CreateBanca(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Result().StatusCode)
	}
}

// ── Sanitization tests ──

func TestSanitizeStripsHTML(t *testing.T) {
	got := Sanitize("<script>alert('xss')</script>Hola", 100)
	if got != "alert('xss')Hola" {
		t.Errorf("expected 'alert(xss)Hola', got %q", got)
	}
}

func TestSanitizeMaxLen(t *testing.T) {
	got := Sanitize("abcdefghij", 5)
	if got != "abcde" {
		t.Errorf("expected 'abcde', got %q", got)
	}
}

func TestSanitizeTrimsSpaces(t *testing.T) {
	got := Sanitize("  hola  ", 100)
	if got != "hola" {
		t.Errorf("expected 'hola', got %q", got)
	}
}

func TestSanitizeEmpty(t *testing.T) {
	got := Sanitize("", 100)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSanitizeNestedTags(t *testing.T) {
	got := Sanitize("<div><span>texto</span></div>", 100)
	if got != "texto" {
		t.Errorf("expected 'texto', got %q", got)
	}
}

func TestSanitizeMessageEscapesHTML(t *testing.T) {
	got := SanitizeMessage("<script>alert(1)</script>", 2000)
	if got != "&lt;script&gt;alert(1)&lt;/script&gt;" {
		t.Errorf("expected escaped, got %q", got)
	}
}

func TestSanitizeMessageMaxLen(t *testing.T) {
	got := SanitizeMessage("abcdefghij", 5)
	if got != "abcde" {
		t.Errorf("expected 'abcde', got %q", got)
	}
}

func TestSanitizeMessageEmpty(t *testing.T) {
	got := SanitizeMessage("", 100)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSanitizeMessageOnlySpaces(t *testing.T) {
	got := SanitizeMessage("   ", 100)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestIsValidID(t *testing.T) {
	if !isValidID("abc12345") {
		t.Error("expected 'abc12345' to be valid")
	}
	if isValidID("abc") {
		t.Error("expected 'abc' to be invalid (too short)")
	}
	if isValidID("abcdefghij") {
		t.Error("expected 'abcdefghij' to be invalid (too long)")
	}
	if isValidID("abc def!") {
		t.Error("expected 'abc def!' to be invalid (special chars)")
	}
	if isValidID("xxxxyyyy") {
		t.Error("expected 'xxxxyyyy' to be invalid (hex only)")
	}
}

func TestCreateBancaSanitized(t *testing.T) {
	h := newTestHandler()
	body := `{"title":"<b>XSS</b> Banca"}`
	req := httptest.NewRequest("POST", "/api/bancas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateBanca(w, req)

	var banca models.Banca
	json.NewDecoder(w.Result().Body).Decode(&banca)
	if banca.Title != "XSS Banca" {
		t.Errorf("expected sanitized title, got %q", banca.Title)
	}
}

func TestGetBancaInvalidID(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/bancas/invalid", nil)
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	h.GetBanca(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ID, got %d", w.Result().StatusCode)
	}
}
