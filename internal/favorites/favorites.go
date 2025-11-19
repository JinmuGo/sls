package favorites

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type entry struct {
	Favorite bool `json:"favorite"`
	Count    int  `json:"count"`
}

type Store struct {
	path string
	data map[string]entry
}

func DefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}
	metaPath := filepath.Join(home, ".config", "sls", "meta.json")
	return NewStore(metaPath)
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: make(map[string]entry)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		// File not existing is not an error for first time usage
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read meta file: %w", err)
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return fmt.Errorf("parse meta file %s: %w (file may be corrupted)", s.path, err)
	}
	return nil
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta data: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		return fmt.Errorf("write meta file: %w", err)
	}
	return nil
}

func (s *Store) Add(h string) error {
	e := s.data[h]
	e.Favorite = true
	s.data[h] = e
	return s.save()
}

func (s *Store) Remove(h string) error {
	if e, ok := s.data[h]; ok {
		e.Favorite = false
		s.data[h] = e
		return s.save()
	}
	return nil
}

func (s *Store) List() []string {
	var out []string
	for k, v := range s.data {
		if v.Favorite {
			out = append(out, k)
		}
	}
	return out
}

func (s *Store) IsFavorite(h string) bool {
	if e, ok := s.data[h]; ok {
		return e.Favorite
	}
	return false
}

func (s *Store) Increment(h string) error {
	e := s.data[h]
	e.Count++
	s.data[h] = e
	return s.save()
}

func (s *Store) Count(h string) int {
	if e, ok := s.data[h]; ok {
		return e.Count
	}
	return 0
}
