// Package auth holds the CLI's credential storage and token-refresh logic.
//
// The CLI authenticates via OAuth 2.0 + PKCE (see internal/oauth) and
// persists the resulting access + refresh tokens in the OS keychain. The
// API client asks for a valid access token; if the cached one is within
// 60s of expiring we transparently refresh against Clerk's /oauth/token.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/oriyn-ai/cli/internal/oauth"
)

const (
	keyringService = "oriyn-cli"
	keyringUser    = "credentials"
)

var (
	ErrNotLoggedIn    = errors.New("not logged in — run `oriyn login`")
	ErrSessionExpired = errors.New("session expired — run `oriyn login` again")
)

// Credentials hold the OAuth 2.0 token pair returned by Clerk's token
// endpoint. RefreshToken is used to mint a fresh AccessToken when the
// current one expires; both rotate together because Clerk rotates refresh
// tokens on every refresh.
type Credentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
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

// Store coordinates credential persistence and OAuth refresh. Concurrent
// API calls may all observe an expiring token at once; the mutex ensures a
// single refresh per expiry window.
type Store struct {
	keyring Keyring
	http    *http.Client
	oauth   oauth.Config

	mu sync.Mutex
}

// NewStore returns a Store with default OS keychain access and the given
// OAuth config. The OAuth config is required even for non-login commands
// because the refresh path needs it.
func NewStore(cfg oauth.Config) *Store {
	return &Store{
		keyring: defaultKeyring{},
		http:    &http.Client{Timeout: 30 * time.Second},
		oauth:   cfg,
	}
}

// NewStoreWithKeyring is the test seam for swapping the keyring backend.
func NewStoreWithKeyring(kr Keyring, cfg oauth.Config) *Store {
	return &Store{
		keyring: kr,
		http:    &http.Client{Timeout: 30 * time.Second},
		oauth:   cfg,
	}
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

// GetValidAccessToken returns a non-expired access token, refreshing
// against Clerk if needed. ORIYN_ACCESS_TOKEN remains an escape hatch for
// CI / agent environments — callers are expected to set it to a token the
// API will accept; no refresh happens for env-supplied tokens.
func (s *Store) GetValidAccessToken(ctx context.Context) (string, error) {
	if token := os.Getenv("ORIYN_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	creds, err := s.Load()
	if err != nil {
		return "", ErrNotLoggedIn
	}

	// 60s safety margin — refresh proactively rather than risk a request
	// rejected mid-flight.
	if creds.ExpiresAt-time.Now().Unix() > 60 {
		return creds.AccessToken, nil
	}

	// Try refresh. If refresh itself fails (no refresh token, refresh
	// rejected, network error against the token endpoint), drop the
	// credentials and force re-login — anything else risks looping
	// forever on a permanently-broken token.
	if creds.RefreshToken == "" {
		_ = s.Delete()
		return "", ErrSessionExpired
	}

	tr, err := s.oauth.RefreshToken(ctx, s.http, creds.RefreshToken)
	if err != nil {
		_ = s.Delete()
		return "", ErrSessionExpired
	}

	refreshed := &Credentials{
		AccessToken:  tr.AccessToken,
		RefreshToken: orFallback(tr.RefreshToken, creds.RefreshToken),
		ExpiresAt:    time.Now().Unix() + tr.ExpiresIn,
	}
	if err := s.Save(refreshed); err != nil {
		// Saving failed but the token is good for this request; surface
		// it without persisting and let the next call retry.
		return refreshed.AccessToken, nil
	}
	return refreshed.AccessToken, nil
}

func orFallback(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
