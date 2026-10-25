package main

import (
	"sync"

	"spellcast/internal/config"
)

// configStore holds the active config. A failed reload keeps the old one.
type configStore struct {
	path string

	mu  sync.RWMutex
	cfg *config.Config
}

func newConfigStore(path string) *configStore {
	return &configStore{path: path}
}

// Get returns the active config (nil until the first successful Reload).
func (s *configStore) Get() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Reload reads the config file. On error the active config is unchanged.
func (s *configStore) Reload() (*config.Config, error) {
	cfg, err := config.Load(s.path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return cfg, nil
}
