package models

import (
	"testing"
)

func TestNewPlaza(t *testing.T) {
	p := NewPlaza()
	if p == nil {
		t.Fatal("expected non-nil plaza")
	}
	if len(p.Bancas) != 0 {
		t.Errorf("expected empty bancas, got %d", len(p.Bancas))
	}
}

func TestCreateAndGetBanca(t *testing.T) {
	p := NewPlaza()
	b := p.CreateBanca("test banca", 33)
	if b.Title != "test banca" {
		t.Errorf("expected 'test banca', got %q", b.Title)
	}
	if b.MaxUsers != 33 {
		t.Errorf("expected 33, got %d", b.MaxUsers)
	}
	if !b.Active {
		t.Error("expected new banca to be active")
	}
	if b.Users != 0 {
		t.Errorf("expected 0 users, got %d", b.Users)
	}
	if b.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(b.ID) != 8 {
		t.Errorf("expected 8-char ID, got %d", len(b.ID))
	}
	if b.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	got, ok := p.GetBanca(b.ID)
	if !ok {
		t.Fatal("expected to find banca by ID")
	}
	if got.Title != b.Title {
		t.Errorf("expected %q, got %q", b.Title, got.Title)
	}
}

func TestListBancas(t *testing.T) {
	p := NewPlaza()
	p.CreateBanca("banca 1", 10)
	p.CreateBanca("banca 2", 20)

	list := p.GetBancas()
	if len(list) != 2 {
		t.Errorf("expected 2 bancas, got %d", len(list))
	}
}

func TestListBancasEmpty(t *testing.T) {
	p := NewPlaza()
	list := p.GetBancas()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestJoinBanca(t *testing.T) {
	p := NewPlaza()
	b := p.CreateBanca("test", 5)

	ok := p.JoinBanca(b.ID, "client-1")
	if !ok {
		t.Fatal("expected join to succeed")
	}

	b2, _ := p.GetBanca(b.ID)
	if b2.Users != 1 {
		t.Errorf("expected 1 user, got %d", b2.Users)
	}
}

func TestJoinBancaFull(t *testing.T) {
	p := NewPlaza()
	b := p.CreateBanca("test", 2)

	p.JoinBanca(b.ID, "c1")
	p.JoinBanca(b.ID, "c2")
	ok := p.JoinBanca(b.ID, "c3")
	if ok {
		t.Error("expected join to fail when banca is full")
	}
}

func TestJoinNonExistentBanca(t *testing.T) {
	p := NewPlaza()
	ok := p.JoinBanca("nonexistent", "c1")
	if ok {
		t.Error("expected join to fail for non-existent banca")
	}
}

func TestLeaveBanca(t *testing.T) {
	p := NewPlaza()
	b := p.CreateBanca("test", 10)

	p.JoinBanca(b.ID, "c1")
	p.JoinBanca(b.ID, "c2")
	p.LeaveBanca(b.ID, "c1")

	b2, _ := p.GetBanca(b.ID)
	if b2.Users != 1 {
		t.Errorf("expected 1 user after leave, got %d", b2.Users)
	}
	if !b2.Active {
		t.Error("expected banca to stay active with users")
	}
}

func TestLeaveBancaLastPerson(t *testing.T) {
	p := NewPlaza()
	b := p.CreateBanca("test", 10)

	p.JoinBanca(b.ID, "c1")
	p.LeaveBanca(b.ID, "c1")

	_, ok := p.GetBanca(b.ID)
	if ok {
		t.Error("expected banca to be removed after last person leaves")
	}
}

func TestMultipleBancas(t *testing.T) {
	p := NewPlaza()
	b1 := p.CreateBanca("b1", 5)
	b2 := p.CreateBanca("b2", 5)

	p.JoinBanca(b1.ID, "c1")
	p.JoinBanca(b1.ID, "c2")
	p.JoinBanca(b2.ID, "c3")

	list := p.GetBancas()
	if len(list) != 2 {
		t.Errorf("expected 2 bancas, got %d", len(list))
	}

	b1got, _ := p.GetBanca(b1.ID)
	if b1got.Users != 2 {
		t.Errorf("expected b1 to have 2 users, got %d", b1got.Users)
	}

	b2got, _ := p.GetBanca(b2.ID)
	if b2got.Users != 1 {
		t.Errorf("expected b2 to have 1 user, got %d", b2got.Users)
	}
}

func TestLeaveNonExistent(t *testing.T) {
	p := NewPlaza()
	p.CreateBanca("test", 10)
	p.LeaveBanca("nonexistent", "c1")
	p.LeaveBanca("test", "c1")
}

func TestRandomIDLength(t *testing.T) {
	for i := 0; i < 10; i++ {
		p := NewPlaza()
		b := p.CreateBanca("test", 10)
		if len(b.ID) != 8 {
			t.Fatalf("expected 8-char ID, got %d: %q", len(b.ID), b.ID)
		}
	}
}

func TestRandomIDUnique(t *testing.T) {
	p := NewPlaza()
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		b := p.CreateBanca("test", 10)
		if ids[b.ID] {
			t.Fatalf("duplicate ID: %s", b.ID)
		}
		ids[b.ID] = true
	}
}
