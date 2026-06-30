package models

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store interface {
	SaveMessage(bancaID string, msg ChatMessage) error
	LoadMessages(bancaID string, limit int) ([]ChatMessage, error)
}

type FileStore struct {
	mu  sync.Mutex
	dir string
}

func NewFileStore(dataDir string) *FileStore {
	dir := filepath.Join(dataDir, "bancas")
	os.MkdirAll(dir, 0755)
	return &FileStore{dir: dir}
}

func (s *FileStore) path(bancaID string) string {
	return filepath.Join(s.dir, bancaID+".jsonl")
}

func (s *FileStore) SaveMessage(bancaID string, msg ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path(bancaID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(msg)
}

func (s *FileStore) LoadMessages(bancaID string, limit int) ([]ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path(bancaID))
	if err != nil {
		if os.IsNotExist(err) {
			return []ChatMessage{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []ChatMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var msg ChatMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		all = append(all, msg)
	}

	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}
