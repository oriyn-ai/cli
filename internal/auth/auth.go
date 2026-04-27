package auth

import (
	"bytes"
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
	supabaseURL            = "https://ddykhzwjzbgpomlmkeji.supabase.co"
	supabasePublishableKey = "sb_publishable_FZtcboPlsEdA9tFS0bOWdQ_YBFpphBv"
	keyringService         = "oriyn-cli"
	keyringUser            = "credentials"
)

var (
	ErrNotLoggedIn    = errors.New("not logged in — run `oriyn login`")
	ErrSessionExpired = errors.New("session expired — run `oriyn login` again")
)

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
	// Marshaling tokens for OS keyring storage — by design.
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

// GetValidAccessToken returns a valid access token, refreshing if needed.
// It implements the AuthProvider interface used by the API client.
func (s *Store) GetValidAccessToken(ctx context.Context) (string, error) {
	if token := os.Getenv("ORIYN_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := s.Load()
	if err != nil {
		return "", ErrNotLoggedIn
	}

	if creds.ExpiresAt-time.Now().Unix() > 60 {
		return creds.AccessToken, nil
	}

	refreshed, err := s.refreshToken(ctx, creds.RefreshToken)
	if err != nil {
		_ = s.Delete()
		return "", ErrSessionExpired
	}

	creds.AccessToken = refreshed.AccessToken
	creds.RefreshToken = refreshed.RefreshToken
	creds.ExpiresAt = refreshed.ExpiresAt
	if err := s.Save(creds); err != nil {
		return "", err
	}

	return creds.AccessToken, nil
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *Store) refreshToken(ctx context.Context, token string) (*Credentials, error) {
	// Marshaling refresh token into a request body sent to oriyn-api.
	//nolint:gosec // G117: intentional; destination is the API over TLS.
	body, err := json.Marshal(refreshRequest{RefreshToken: token})
	if err != nil {
		return nil, fmt.Errorf("failed to encode refresh request: %w", err)
	}

	url := supabaseURL + "/auth/v1/token?grant_type=refresh_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", supabasePublishableKey)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Supabase for token refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("session expired (%d) — run `oriyn login` again", resp.StatusCode)
	}

	var data refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse token refresh response: %w", err)
	}

	return &Credentials{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		ExpiresAt:    time.Now().Unix() + data.ExpiresIn,
	}, nil
}
