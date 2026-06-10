package favorites

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinmugo/sls/internal/util"
)

// Entry represents metadata for a single host
type Entry struct {
	Favorite bool     `json:"favorite"`
	Count    int      `json:"count"`
	Tags     []string `json:"tags,omitempty"`
}

type Store struct {
	path string
	data map[string]Entry
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
	s := &Store{path: path, data: make(map[string]Entry)}
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
	// Atomic write: a torn meta.json (e.g. process killed mid-write) would fail
	// to parse and brick sls on the next startup, so write+rename instead.
	if err := util.AtomicWriteFile(s.path, b, 0o600); err != nil {
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

// Delete removes a host's metadata entry entirely (favorite flag, usage count,
// and tags). Used when a host or container is deleted so no orphan key lingers
// in meta.json.
func (s *Store) Delete(h string) error {
	if _, ok := s.data[h]; !ok {
		return nil
	}
	delete(s.data, h)
	return s.save()
}

// Rename moves a host's entire metadata entry (favorite flag, usage count, and
// tags) from oldKey to newKey, preserving history across a rename. If newKey
// already exists it is overwritten. No-op if oldKey is absent.
func (s *Store) Rename(oldKey, newKey string) error {
	e, ok := s.data[oldKey]
	if !ok || oldKey == newKey {
		return nil
	}
	s.data[newKey] = e
	delete(s.data, oldKey)
	return s.save()
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

// AddTag adds a tag to a host
func (s *Store) AddTag(h, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	e := s.data[h]
	// Check if tag already exists
	for _, t := range e.Tags {
		if t == tag {
			return fmt.Errorf("tag %q already exists for host %q", tag, h)
		}
	}
	e.Tags = append(e.Tags, tag)
	s.data[h] = e
	return s.save()
}

// RemoveTag removes a tag from a host
func (s *Store) RemoveTag(h, tag string) error {
	e, ok := s.data[h]
	if !ok {
		return nil // Host doesn't exist, nothing to do
	}

	found := false
	newTags := []string{}
	for _, t := range e.Tags {
		if t != tag {
			newTags = append(newTags, t)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("tag %q not found for host %q", tag, h)
	}

	e.Tags = newTags
	s.data[h] = e
	return s.save()
}

// GetTags returns all tags for a host
func (s *Store) GetTags(h string) []string {
	if e, ok := s.data[h]; ok {
		return e.Tags
	}
	return nil
}

// ListAllTags returns all unique tags across all hosts
func (s *Store) ListAllTags() []string {
	tagSet := make(map[string]struct{})
	for _, e := range s.data {
		for _, tag := range e.Tags {
			tagSet[tag] = struct{}{}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags
}

// HasTag checks if a host has a specific tag
func (s *Store) HasTag(h, tag string) bool {
	if e, ok := s.data[h]; ok {
		for _, t := range e.Tags {
			if t == tag {
				return true
			}
		}
	}
	return false
}

// Data returns the internal data map (for iteration)
func (s *Store) Data() map[string]Entry {
	return s.data
}
