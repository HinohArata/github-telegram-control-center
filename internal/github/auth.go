package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type AppConfig struct {
	AppID      int64
	PrivateKey []byte
	BaseURL    string
}

type AuthManager struct {
	cfg       AppConfig
	mu        sync.RWMutex
	tokenCache map[int64]cachedToken
}

type cachedToken struct {
	token   string
	expires time.Time
}

func NewAuthManager(cfg AppConfig) *AuthManager {
	return &AuthManager{cfg: cfg, tokenCache: map[int64]cachedToken{}}
}

func parseKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}

func (a *AuthManager) signJWT() (string, error) {
	key, err := parseKey(a.cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	now := time.Now()
	header := map[string]any{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": a.cfg.AppID,
	}
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	b64 := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}
	message := b64(h) + "." + b64(c)
	digest := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return message + "." + b64(sig), nil
}

func (a *AuthManager) AppJWT() (string, error) {
	return a.signJWT()
}

func (a *AuthManager) baseURL() string {
	if a.cfg.BaseURL != "" {
		return a.cfg.BaseURL
	}
	return "https://api.github.com"
}

type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (a *AuthManager) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	a.mu.RLock()
	if ct, ok := a.tokenCache[installationID]; ok && time.Now().Add(60*time.Second).Before(ct.expires) {
		a.mu.RUnlock()
		return ct.token, nil
	}
	a.mu.RUnlock()

	jwt, err := a.AppJWT()
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(map[string]any{})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL()+"/app/installations/"+itoa(installationID)+"/access_tokens", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "github-telegram-control-center")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("create installation token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create installation token: status %d", resp.StatusCode)
	}
	var tok InstallationToken
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", err
	}
	if tok.Token == "" {
		return "", fmt.Errorf("empty installation token from github")
	}

	a.mu.Lock()
	a.tokenCache[installationID] = cachedToken{token: tok.Token, expires: tok.ExpiresAt}
	a.mu.Unlock()

	return tok.Token, nil
}

func (a *AuthManager) InvalidateInstallationToken(installationID int64) {
	a.mu.Lock()
	delete(a.tokenCache, installationID)
	a.mu.Unlock()
}

func (a *AuthManager) ClearCache() {
	a.mu.Lock()
	a.tokenCache = map[int64]cachedToken{}
	a.mu.Unlock()
}