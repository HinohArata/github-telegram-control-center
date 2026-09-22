package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

func VerifyWebhookSignature(secret string, body []byte, header string) error {
	if header == "" {
		return fmt.Errorf("missing X-Hub-Signature-256")
	}
	expected := "sha256=" + HMACSHA256(secret, body)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(header))) {
		return fmt.Errorf("invalid webhook signature")
	}
	return nil
}

func HMACSHA256(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type WebhookRequest struct {
	DeliveryID string
	EventType  string
	Body       []byte
}

func ParseWebhookRequest(r *http.Request, secret string, body []byte) (*WebhookRequest, error) {
	event := r.Header.Get("X-GitHub-Event")
	delivery := r.Header.Get("X-GitHub-Delivery")
	if err := VerifyWebhookSignature(secret, body, r.Header.Get("X-Hub-Signature-256")); err != nil {
		return nil, err
	}
	return &WebhookRequest{DeliveryID: delivery, EventType: event, Body: body}, nil
}
