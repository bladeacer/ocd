package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var CacheDir = ".obsidian_cache"

var (
	ErrCacheMiss  = errors.New("cache miss")
	ErrCacheStale = errors.New("cache stale")
)

type Entry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

type Store struct {
	dir string
	ttl time.Duration
}

func New(ttl time.Duration) *Store {
	s := &Store{
		dir: CacheDir,
		ttl: ttl,
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		panic(err)
	}
	return s
}

func (s *Store) path(name string) string {
	return filepath.Join(s.dir, name+".json")
}

func (s *Store) Get(name string, v interface{}) error {
	p := s.path(name)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrCacheMiss
		}
		return err
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	if s.ttl > 0 && time.Since(entry.Timestamp) > s.ttl {
		return ErrCacheStale
	}
	return json.Unmarshal(entry.Data, v)
}

func (s *Store) Set(name string, v interface{}) error {
	p := s.path(name)
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	entry := Entry{
		Data:      json.RawMessage(data),
		Timestamp: time.Now(),
	}
	out, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(p, out, 0644)
}

func (s *Store) Delete(name string) error {
	p := s.path(name)
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) Clear() error {
	d, err := os.Open(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		os.Remove(filepath.Join(s.dir, name))
	}
	return nil
}
