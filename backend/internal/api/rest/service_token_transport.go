package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/retran/nexus/backend/internal/auth"
)

const tokenRefreshSkew = 15 * time.Second

type serviceTokenTransport struct {
	expiresAt   time.Time
	base        http.RoundTripper
	tokenClient *auth.TokenClient
	subject     string
	token       string
	audience    []string
	ttl         time.Duration
	mu          sync.Mutex
}

func newServiceTokenTransport(
	base http.RoundTripper,
	tokenClient *auth.TokenClient,
	subject string,
	audience []string,
	ttl time.Duration,
) *serviceTokenTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	normalizedAudience := make([]string, 0, len(audience))
	for _, aud := range audience {
		if trimmed := strings.TrimSpace(aud); trimmed != "" {
			normalizedAudience = append(normalizedAudience, trimmed)
		}
	}

	return &serviceTokenTransport{
		base:        base,
		tokenClient: tokenClient,
		subject:     strings.TrimSpace(subject),
		audience:    normalizedAudience,
		ttl:         ttl,
	}
}

func (t *serviceTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.tokenClient == nil {
		return nil, errors.New("token client is not configured")
	}

	token, err := t.tokenForRequest(req.Context())
	if err != nil {
		return nil, fmt.Errorf("issue service token: %w", err)
	}

	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	cloned := req.Clone(ctx)
	cloned.Header = cloned.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+token)

	resp, err := t.base.RoundTrip(cloned)
	if err != nil {
		return nil, fmt.Errorf("perform graphql request: %w", err)
	}

	return resp, nil
}

func (t *serviceTokenTransport) tokenForRequest(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if t.token != "" && now.Add(tokenRefreshSkew).Before(t.expiresAt) {
		return t.token, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := t.tokenClient.IssueToken(ctx, &auth.IssueTokenRequest{
		Subject:  t.subject,
		Audience: append([]string{}, t.audience...),
		TTL:      t.ttl,
	})
	if err != nil {
		return "", fmt.Errorf("obtain signed token: %w", err)
	}

	t.token = resp.Token
	t.expiresAt = resp.ExpiresAt
	return t.token, nil
}
