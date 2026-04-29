// Package oauth implements the OAuth 2.0 Authorization Code flow with PKCE
// for native/CLI clients (RFC 6749 + RFC 7636 + RFC 8252).
//
// The CLI is registered as a public OAuth client at Clerk; PKCE replaces
// the client secret. The verifier never leaves the CLI process; only the
// SHA-256 challenge is sent to the authorize endpoint. The authorization
// code is exchanged for an access token + refresh token over a server-to-
// server POST that includes the verifier — proving the same client that
// initiated the flow is completing it.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds the static OAuth client configuration. ClientID and the
// endpoint URLs come from the Clerk OAuth application registration; the
// CLI does not have (or need) a client secret because PKCE proves
// possession instead.
type Config struct {
	ClientID     string
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

// PKCEPair is a code_verifier + its derived code_challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
}

// GeneratePKCE produces a random 32-byte verifier and its S256 challenge.
// The verifier must be at least 43 chars (RFC 7636 §4.1); base64url of 32
// raw bytes gives 43 chars without padding.
func GeneratePKCE() (PKCEPair, error) {
	verifier, err := randomURLString(32)
	if err != nil {
		return PKCEPair{}, fmt.Errorf("generating verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return PKCEPair{Verifier: verifier, Challenge: challenge}, nil
}

// GenerateState returns a random opaque value for CSRF binding of the
// authorize → callback round-trip.
func GenerateState() (string, error) {
	s, err := randomURLString(24)
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return s, nil
}

func randomURLString(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// BuildAuthorizeURL builds the URL the user opens in their browser. The
// state and code_challenge bind this attempt to the local callback server.
func (c Config) BuildAuthorizeURL(state, codeChallenge, redirectURI string) string {
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(c.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	return c.AuthorizeURL + "?" + q.Encode()
}

// TokenResponse is the standard OAuth 2.0 token endpoint shape.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// ExchangeCode trades a one-time code for tokens. The code_verifier proves
// this client is the one that initiated the authorize step.
func (c Config) ExchangeCode(ctx context.Context, httpClient *http.Client, code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("client_id", c.ClientID)
	body.Set("code_verifier", codeVerifier)
	body.Set("redirect_uri", redirectURI)
	return c.postToken(ctx, httpClient, body)
}

// RefreshToken trades a refresh token for a fresh access token. Clerk
// rotates the refresh token, so callers must persist the new one.
func (c Config) RefreshToken(ctx context.Context, httpClient *http.Client, refreshToken string) (*TokenResponse, error) {
	body := url.Values{}
	body.Set("grant_type", "refresh_token")
	body.Set("client_id", c.ClientID)
	body.Set("refresh_token", refreshToken)
	return c.postToken(ctx, httpClient, body)
}

func (c Config) postToken(ctx context.Context, httpClient *http.Client, body url.Values) (*TokenResponse, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	return &tr, nil
}

// UserInfo is the subset of OIDC /userinfo claims the CLI consumes for the
// "Logged in as ..." line.
type UserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// FetchUserInfo calls the OIDC /userinfo endpoint with the access token.
// Used at login time to greet the user with their email; failure is
// non-fatal (we still consider the login successful if the token endpoint
// returned a token).
func (c Config) FetchUserInfo(ctx context.Context, httpClient *http.Client, accessToken string) (*UserInfo, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned %d", resp.StatusCode)
	}

	var ui UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&ui); err != nil {
		return nil, fmt.Errorf("decoding userinfo: %w", err)
	}
	return &ui, nil
}
