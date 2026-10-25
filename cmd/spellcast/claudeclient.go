package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"spellcast/internal/claude"
)

// clientHolder owns the Claude client. The key is read when the client is
// created (at start, "Reload config", "Set API key…"), not for every request.
type clientHolder struct {
	mu     sync.RWMutex
	client *claude.Client // nil without a key
	source string         // where the key came from
	err    error          // set if the credential store itself failed
}

// Reload reads the key and creates a new client. It returns the key source
// ("" if there is no key).
func (h *clientHolder) Reload() string {
	key, source, err := claude.LoadKey()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client, h.source, h.err = nil, "", nil
	switch {
	case err == nil:
		h.client, h.source = claude.New(key), source
		slog.Info("API key loaded", "source", source) // never the key itself
	case errors.Is(err, claude.ErrNoAPIKey):
		slog.Warn("no API key")
	default:
		h.err = err
		slog.Error("cannot read the API key", "err", err)
	}
	return h.source
}

// Source returns where the current key comes from ("" if none).
func (h *clientHolder) Source() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.source
}

// Client implements core.ClientFunc.
func (h *clientHolder) Client() (claude.Streamer, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.client != nil {
		return h.client, nil
	}
	if h.err != nil {
		return nil, h.err
	}
	return nil, claude.ErrNoAPIKey
}

// SaveKey implements ui.SaveKeyFunc: check the key, store it, use it at once.
// A key the API rejects is not stored. If the check itself fails (network),
// the key is stored anyway and the message says so.
func (h *clientHolder) SaveKey(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "No key received. Paste the key and try again.", errors.New("empty key")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	verr := claude.ValidateKey(ctx, key)
	if errors.Is(verr, claude.ErrInvalidKey) {
		return "The API rejected this key. Nothing was saved.", verr
	}
	if err := claude.SaveKey(key); err != nil {
		return "Could not store the key: " + err.Error(), err
	}
	h.Reload()
	if verr != nil {
		slog.Warn("API key stored but not verified", "err", verr)
		return "Saved. The key could not be checked right now (network?), but it will be used.", nil
	}
	return "Saved. The key works.", nil
}
