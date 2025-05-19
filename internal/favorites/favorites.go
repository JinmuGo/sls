package favorites

import (
	"encoding/json"
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

func DefaultStore() *Store {
	home, _ := os.UserHomeDir()
	metaPath := filepath.Join(home, ".config", "sls", "meta.json")
	return NewStore(metaPath)
}

func NewStore(path string) *Store {
	s := &Store{path: path, data: make(map[string]entry)}
	s.load()
	return s
}

func (s *Store) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &s.data)
}

func (s *Store) save() error {
	b, _ := json.MarshalIndent(s.data, "", "  ")
	os.MkdirAll(filepath.Dir(s.path), 0o755)
	return os.WriteFile(s.path, b, 0o644)
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

func (s *Store) Increment(h string) {
	e := s.data[h]
	e.Count++
	s.data[h] = e
	_ = s.save()
}

func (s *Store) Count(h string) int {
	if e, ok := s.data[h]; ok {
		return e.Count
	}
	return 0
}
