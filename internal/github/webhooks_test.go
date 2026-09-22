package github

import (
	"net/http"
	"strings"
	"testing"
)

func TestHMACSHA256(t *testing.T) {
	got := HMACSHA256("secret", []byte(`{"hello":"world"}`))
	want := "2677ad3e7c090b2fa2c0fb13020d66d5420879b8316eb356a2d60fb9073bc778"
	if got != want {
		// GitHub documents this vector; a mismatch means the HMAC is wrong.
		t.Fatalf("HMACSHA256 = %q; want documented vector", got)
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "super-secret"
	body := []byte(`{"action":"opened"}`)
	sig := "sha256=" + HMACSHA256(secret, body)

	tests := []struct {
		name    string
		secret  string
		header  string
		body    []byte
		wantErr bool
	}{
		{"valid", secret, sig, body, false},
		{"wrong secret", "other", sig, body, true},
		{"tampered body", secret, sig, []byte(`{"action":"closed"}`), true},
		{"missing header", secret, "", body, true},
		{"empty header", secret, " ", body, true},
		{"wrong prefix", secret, "sha1=" + strings.TrimPrefix(sig, "sha256="), body, true},
		{"trailing whitespace ok", secret, sig + " ", body, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyWebhookSignature(tt.secret, tt.body, tt.header)
			if tt.wantErr && err == nil {
				t.Fatal("expected signature verification error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseWebhookRequest(t *testing.T) {
	secret := "s"
	body := []byte(`{"action":"opened","number":1}`)
	r := httptestRequest(body, map[string]string{
		"X-GitHub-Event":      "pull_request",
		"X-GitHub-Delivery":   "abc-123",
		"X-Hub-Signature-256": "sha256=" + HMACSHA256(secret, body),
	})

	got, err := ParseWebhookRequest(r, secret, body)
	if err != nil {
		t.Fatalf("ParseWebhookRequest: %v", err)
	}
	if got.DeliveryID != "abc-123" {
		t.Errorf("DeliveryID = %q", got.DeliveryID)
	}
	if got.EventType != "pull_request" {
		t.Errorf("EventType = %q", got.EventType)
	}
	if string(got.Body) != string(body) {
		t.Errorf("Body mismatch")
	}
}

func TestParseWebhookRequestRejectsBadSignature(t *testing.T) {
	r := httptestRequest([]byte(`{}`), map[string]string{
		"X-GitHub-Event":      "push",
		"X-Hub-Signature-256": "sha256=deadbeef",
	})
	if _, err := ParseWebhookRequest(r, "s", []byte(`{}`)); err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestParseWebhookRequestRejectsMissingSignature(t *testing.T) {
	r := httptestRequest([]byte(`{}`), map[string]string{
		"X-GitHub-Event": "push",
	})
	if _, err := ParseWebhookRequest(r, "s", []byte(`{}`)); err == nil {
		t.Fatal("expected error for missing signature")
	}
}

func TestParseWebhookRequestMalformedJSON(t *testing.T) {
	// Signature is computed over the raw body, so malformed JSON with a valid
	// signature must still be accepted here; the handler validates structure.
	body := []byte(`{not json`)
	r := httptestRequest(body, map[string]string{
		"X-GitHub-Event":      "push",
		"X-Hub-Signature-256": "sha256=" + HMACSHA256("s", body),
	})
	got, err := ParseWebhookRequest(r, "s", body)
	if err != nil {
		t.Fatalf("ParseWebhookRequest: %v", err)
	}
	if got.EventType != "push" {
		t.Errorf("EventType = %q", got.EventType)
	}
}

func httptestRequest(body []byte, headers map[string]string) *http.Request {
	r, err := http.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(string(body)))
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}
