package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/metrics"
)

type httpClient struct {
	baseURL   string
	client    *http.Client
	onRate    func(rate RateLimitInfo)
	onRequest func(method, path string)
}

type RateLimitInfo struct {
	Resource  string
	Limit     int
	Remaining int
	Reset     time.Time
}

type apiError struct {
	Status int
	Msg    string
	Body   string
}

func (e *apiError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("github api status %d: %s", e.Status, e.Msg)
	}
	return fmt.Sprintf("github api status %d", e.Status)
}

func (c *httpClient) request(ctx context.Context, method, path, token string, body any) *http.Request {
	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err == nil {
			reader = &buf
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		req, _ = http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func (c *httpClient) do(ctx context.Context, req *http.Request, out any, installationID string) (*http.Response, error) {
	const maxAttempts = 4
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		metrics.GitHubAPIRequestsTotal.Inc()
		if c.onRequest != nil {
			c.onRequest(req.Method, req.URL.Path)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("github request: %w", err)
			metrics.GitHubAPIErrorsTotal.Inc()
			time.Sleep(backoff(attempt))
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		c.trackRate(resp)

		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			if attempt < maxAttempts {
				time.Sleep(backoff(attempt))
				continue
			}
		case resp.StatusCode == http.StatusTooManyRequests:
			retryAfter := parseRetryAfter(resp)
			time.Sleep(retryAfter)
			lastErr = &apiError{Status: resp.StatusCode, Msg: "rate limited"}
			if attempt < maxAttempts {
				continue
			}
		case resp.StatusCode >= 500:
			lastErr = &apiError{Status: resp.StatusCode, Msg: "server error", Body: string(raw)}
			metrics.GitHubAPIErrorsTotal.Inc()
			if attempt < maxAttempts {
				time.Sleep(backoff(attempt))
				continue
			}
		case resp.StatusCode >= 400:
			lastErr = &apiError{Status: resp.StatusCode, Msg: errorMessage(raw), Body: string(raw)}
			metrics.GitHubAPIErrorsTotal.Inc()
			return resp, lastErr
		}

		if out != nil && len(raw) > 0 {
			if err := json.Unmarshal(raw, out); err != nil {
				return resp, fmt.Errorf("decode github response: %w", err)
			}
		}
		if resp.StatusCode >= 400 {
			if lastErr != nil {
				return resp, lastErr
			}
		}
		return resp, nil
	}
	if lastErr == nil {
		lastErr = &apiError{Status: 0, Msg: "max attempts exceeded"}
	}
	return nil, lastErr
}

func (c *httpClient) trackRate(resp *http.Response) {
	if c.onRate == nil {
		return
	}
	limit, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
	remaining, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	reset, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Reset"))
	if limit == 0 || remaining == 0 {
		return
	}
	c.onRate(RateLimitInfo{
		Resource:  resp.Header.Get("X-RateLimit-Resource"),
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Unix(int64(reset), 0),
	})
}

func backoff(attempt int) time.Duration {
	return time.Duration(attempt) * time.Second
}

func parseRetryAfter(resp *http.Response) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if n, err := strconv.Atoi(ra); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return 5 * time.Second
}

func errorMessage(raw []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Message != "" {
		return e.Message
	}
	return strings.TrimSpace(string(raw))[:min(len(strings.TrimSpace(string(raw))), 120)]
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

var jsonHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}()

func newHTTPClient() *httpClient {
	return &httpClient{baseURL: "https://api.github.com", client: jsonHTTPClient}
}

type Client struct {
	httpClient *httpClient
	auth       *AuthManager
}

func NewClient(auth *AuthManager) *Client {
	return &Client{
		httpClient: newHTTPClient(),
		auth:       auth,
	}
}

func (c *Client) WithBaseURL(url string) *Client {
	c.httpClient.baseURL = url
	return c
}

func (c *Client) WithRateLimitCallback(cb func(RateLimitInfo)) *Client {
	c.httpClient.onRate = cb
	return c
}

func (c *Client) WithRequestCallback(cb func(method, path string)) *Client {
	c.httpClient.onRequest = cb
	return c
}

func (c *Client) Auth() *AuthManager { return c.auth }

func (c *Client) ListInstallations(ctx context.Context) ([]Installation, error) {
	jwt, err := c.auth.AppJWT()
	if err != nil {
		return nil, err
	}
	h := &httpClient{baseURL: c.httpClient.baseURL, client: c.httpClient.client}
	req := h.request(ctx, "GET", "/app/installations?per_page=100", jwt, nil)
	setAppHeaders(req)
	var out []Installation
	_, err = h.do(ctx, req, &out, "")
	return out, err
}

func setAppHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "github-telegram-control-center")
}