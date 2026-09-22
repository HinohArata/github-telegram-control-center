package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"
)

func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// pemKey builds a PEM private key in the given PKCS format, matching what
// GitHub produces for app keys.
func pemKey(t *testing.T, key *rsa.PrivateKey, pkcs8 bool) []byte {
	t.Helper()
	var der []byte
	var blockType string
	if pkcs8 {
		der = x509PKCS8(t, key)
		blockType = "PRIVATE KEY"
	} else {
		der = x509.MarshalPKCS1PrivateKey(key)
		blockType = "RSA PRIVATE KEY"
	}
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}

func x509PKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	return der
}

func TestAppJWTSignatureAndClaims(t *testing.T) {
	key := generateKey(t)
	auth := NewAuthManager(AppConfig{AppID: 4242, PrivateKey: pemKey(t, key, false)})

	jwt, err := auth.AppJWT()
	if err != nil {
		t.Fatalf("AppJWT: %v", err)
	}

	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT must have 3 parts, got %d", len(parts))
	}

	header := decodeJWTJSON(t, parts[0])
	if alg, _ := header["alg"].(string); alg != "RS256" {
		t.Errorf("header alg = %q; want RS256", alg)
	}
	if typ, _ := header["typ"].(string); typ != "JWT" {
		t.Errorf("header typ = %q; want JWT", typ)
	}

	claims := decodeJWTJSON(t, parts[1])
	iss, _ := claims["iss"].(float64)
	if int64(iss) != 4242 {
		t.Errorf("iss = %v; want 4242", iss)
	}
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)
	now := float64(time.Now().Unix())
	if iat > now {
		t.Errorf("iat %v must not be in the future", iat)
	}
	if exp <= now {
		t.Errorf("exp %v must be in the future", exp)
	}
	if got := exp - iat; got < 8*60 || got > 10*60 {
		t.Errorf("token lifetime = %vs; want ~9-10 minutes", got)
	}

	// Signature must verify against the public key.
	signed := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Errorf("signature verification failed: %v", err)
	}
}

func TestAppJWTUsesConfiguredAppID(t *testing.T) {
	key := generateKey(t)
	auth := NewAuthManager(AppConfig{AppID: 7, PrivateKey: pemKey(t, key, false)})
	jwt, err := auth.AppJWT()
	if err != nil {
		t.Fatalf("AppJWT: %v", err)
	}
	claims := decodeJWTJSON(t, strings.Split(jwt, ".")[1])
	if iss, _ := claims["iss"].(float64); int64(iss) != 7 {
		t.Errorf("iss = %v; want 7", iss)
	}
}

func TestParseKeyAcceptsPKCS1AndPKCS8(t *testing.T) {
	key := generateKey(t)
	if _, err := parseKey(pemKey(t, key, false)); err != nil {
		t.Errorf("parseKey(PKCS#1): %v", err)
	}
	if _, err := parseKey(pemKey(t, key, true)); err != nil {
		t.Errorf("parseKey(PKCS#8): %v", err)
	}
}

func TestParseKeyRejectsGarbage(t *testing.T) {
	if _, err := parseKey([]byte("not a key")); err == nil {
		t.Error("parseKey must reject garbage input")
	}
	if _, err := parseKey(nil); err == nil {
		t.Error("parseKey must reject nil input")
	}
}

func TestTokenCacheHelpersAreSafeBeforeUse(t *testing.T) {
	key := generateKey(t)
	auth := NewAuthManager(AppConfig{AppID: 1, PrivateKey: pemKey(t, key, false)})
	auth.ClearCache()
	auth.InvalidateInstallationToken(123)
	auth.ClearCache()
}

func decodeJWTJSON(t *testing.T, b64 string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return out
}
