package claude

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/zalando/go-keyring"
)

// The key lives in the OS credential store (Windows Credential Manager) under this name.
const (
	keyringService = "spellcast"
	keyringUser    = "anthropic-api-key"
	envKey         = "ANTHROPIC_API_KEY"
)

// Where a key came from.
const (
	SourceCredentialStore = "credential store"
	SourceEnvironment     = envKey
)

// ErrNoAPIKey means neither the credential store nor the environment has a key.
var ErrNoAPIKey = errors.New("no API key")

// ErrInvalidKey means the API rejected the key.
var ErrInvalidKey = errors.New("the API rejected the key")

// LoadKey returns the API key and where it came from. The credential store
// wins over the environment variable. The key is never logged.
func LoadKey() (key, source string, err error) {
	k, kerr := keyring.Get(keyringService, keyringUser)
	if kerr == nil && strings.TrimSpace(k) != "" {
		return strings.TrimSpace(k), SourceCredentialStore, nil
	}
	if k := strings.TrimSpace(os.Getenv(envKey)); k != "" {
		return k, SourceEnvironment, nil
	}
	if kerr != nil && !errors.Is(kerr, keyring.ErrNotFound) {
		return "", "", kerr // the store itself failed: tell the caller
	}
	return "", "", ErrNoAPIKey
}

// SaveKey stores the key in the credential store.
func SaveKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("the key is empty")
	}
	return keyring.Set(keyringService, keyringUser, key)
}

// DeleteKey removes the key from the credential store. A missing key is not an error.
func DeleteKey() error {
	if err := keyring.Delete(keyringService, keyringUser); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

// ValidateKey checks the key with a free request (list one model, no tokens
// are used). It returns ErrInvalidKey if the API rejects the key. Other errors
// (network, server) mean "could not check".
func ValidateKey(ctx context.Context, key string, opts ...option.RequestOption) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := New(key, opts...)
	_, err := client.api.Models.List(ctx, anthropic.ModelListParams{Limit: anthropic.Int(1)})
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
		return ErrInvalidKey
	}
	return err
}
