package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "oriyn-cli"
	keyringUser    = "credentials"
)

var (
	ErrNotLoggedIn    = errors.New("not logged in — run `oriyn login`")
	ErrSessionExpired = errors.New("session expired — run `oriyn login` again")
)

// Credentials stores a Clerk-issued JWT minted via the `cli` JWT template.
// There is no refresh path — Clerk session tokens for the CLI are short-lived
// (24h by default per the template config) and the user re-runs `oriyn login`
// when they expire. This is intentional: it keeps the CLI free of any
// long-term refresh secret.
type Credentials struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

// Keyring abstracts OS keychain access for testability.
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type defaultKeyring struct{}

func (defaultKeyring) Get(service, user string) (string, error) { return keyring.Get(service, user) }
func (defaultKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}
func (defaultKeyring) Delete(service, user string) error { return keyring.Delete(service, user) }

type Store struct {
	keyring Keyring
	http    *http.Client
}

func NewStore() *Store {
	return &Store{keyring: defaultKeyring{}, http: &http.Client{Timeout: 15 * time.Second}}
}

func NewStoreWithKeyring(kr Keyring) *Store {
	return &Store{keyring: kr, http: &http.Client{Timeout: 15 * time.Second}}
}

func (s *Store) Load() (*Credentials, error) {
	data, err := s.keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, fmt.Errorf("not logged in — run `oriyn login`: %w", err)
	}
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse stored credentials — run `oriyn login`: %w", err)
	}
	return &creds, nil
}

func (s *Store) Save(creds *Credentials) error {
	//nolint:gosec // G117: intentional secret marshaling; destination is the OS keyring.
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}
	if err := s.keyring.Set(keyringService, keyringUser, string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in OS keychain: %w", err)
	}
	return nil
}

func (s *Store) Delete() error {
	_ = s.keyring.Delete(keyringService, keyringUser)
	return nil
}

// GetValidAccessToken returns the stored token if not expired. Implements
// the AuthProvider interface used by the API client. There is no refresh —
// when the token expires the user must re-run `oriyn login`.
func (s *Store) GetValidAccessToken(_ context.Context) (string, error) {
	if token := os.Getenv("ORIYN_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := s.Load()
	if err != nil {
		return "", ErrNotLoggedIn
	}

	// 60s safety margin — if the token expires within a minute, force re-login
	// rather than risking a request that will be rejected mid-flight.
	if creds.ExpiresAt-time.Now().Unix() <= 60 {
		_ = s.Delete()
		return "", ErrSessionExpired
	}

	return creds.AccessToken, nil
}
