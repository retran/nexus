package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	_, err := NewClient(nil)
	if err == nil {
		t.Fatalf("expected error when config is nil")
	}

	_, err = NewClient(&Config{})
	if err == nil {
		t.Fatalf("expected error when address is missing")
	}

	_, err = NewClient(&Config{Address: "http://127.0.0.1:8200"})
	if err == nil {
		t.Fatalf("expected error when role id is missing")
	}

	_, err = NewClient(&Config{
		Address: "http://127.0.0.1:8200",
		RoleID:  "role",
	})
	if err == nil {
		t.Fatalf("expected error when secret id is missing")
	}
}

func TestClientReadServiceSecrets(t *testing.T) {
	t.Parallel()

	const expectedToken = "test-token"

	var loginCalls, secretCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case http.MethodPut + " /v1/auth/approle/login":
			loginCalls++
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read login body: %v", err)
			}
			if !bytes.Contains(body, []byte(`"role_id":"role"`)) {
				t.Fatalf("expected role id in login request, got %s", string(body))
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"auth": map[string]any{
					"client_token":   expectedToken,
					"lease_duration": 3600,
				},
			})
		case http.MethodGet + " /v1/kv/data/services/gateway":
			secretCalls++
			if token := r.Header.Get("X-Vault-Token"); token != expectedToken {
				t.Fatalf("expected token %q, got %q", expectedToken, token)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{
				"data": map[string]any{
					"data": map[string]any{
						"postgres_host": "postgres",
						"postgres_port": "5432",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		Address:  server.URL,
		RoleID:   "role",
		SecretID: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	ctx := context.Background()
	secrets, err := client.ReadService(ctx, "gateway")
	if err != nil {
		t.Fatalf("unexpected error reading service secrets: %v", err)
	}

	if got, want := secrets["postgres_host"], "postgres"; got != want {
		t.Fatalf("postgres_host = %q, want %q", got, want)
	}
	if got, want := secrets["postgres_port"], "5432"; got != want {
		t.Fatalf("postgres_port = %q, want %q", got, want)
	}

	// Second call should reuse the cached token.
	if _, err := client.ReadService(ctx, "gateway"); err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}

	if loginCalls != 1 {
		t.Fatalf("expected one login call, got %d", loginCalls)
	}
	if secretCalls != 2 {
		t.Fatalf("expected two secret reads, got %d", secretCalls)
	}
}

func TestClientRenewsTokenWhenExpired(t *testing.T) {
	t.Parallel()

	fakeClock := newFakeClock(time.Unix(0, 0))

	var mu sync.Mutex
	loginCalls := 0
	currentToken := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.Method + " " + r.URL.Path {
		case http.MethodPut + " /v1/auth/approle/login":
			loginCalls++
			currentToken = "token-" + strconv.Itoa(loginCalls)

			writeJSON(t, w, http.StatusOK, map[string]any{
				"auth": map[string]any{
					"client_token":   currentToken,
					"lease_duration": 1,
				},
			})
		case http.MethodGet + " /v1/kv/data/services/worker":
			if token := r.Header.Get("X-Vault-Token"); token != currentToken {
				t.Fatalf("expected token %q, got %q", currentToken, token)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"data": map[string]any{
					"data": map[string]any{
						"foo": "bar",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		Address:          server.URL,
		RoleID:           "role",
		SecretID:         "secret",
		TokenRenewBuffer: 10 * time.Millisecond,
		Clock:            fakeClock.Now,
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	ctx := context.Background()

	if _, err := client.ReadService(ctx, "worker"); err != nil {
		t.Fatalf("unexpected error reading worker secret: %v", err)
	}
	if loginCalls != 1 {
		t.Fatalf("expected one login call after first read, got %d", loginCalls)
	}

	fakeClock.Advance(2 * time.Second)

	if _, err := client.ReadService(ctx, "worker"); err != nil {
		t.Fatalf("unexpected error reading worker secret after expiry: %v", err)
	}

	if loginCalls != 2 {
		t.Fatalf("expected token renewal, login calls = %d", loginCalls)
	}
}

func TestClientReadMissingSecret(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case http.MethodPut + " /v1/auth/approle/login":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"auth": map[string]any{
					"client_token":   "token",
					"lease_duration": 3600,
				},
			})
		case http.MethodGet + " /v1/kv/data/services/data":
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"errors": []string{"not found"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		Address:  server.URL,
		RoleID:   "role",
		SecretID: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = client.ReadService(context.Background(), "data")
	if err == nil {
		t.Fatalf("expected error when secret is missing")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

type fakeClock struct {
	now time.Time
	mu  sync.Mutex
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}
