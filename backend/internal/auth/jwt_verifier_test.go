package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	vaultapi "github.com/hashicorp/vault/api"
)

const transitKeyPath = "/v1/transit/keys/service-jwt-key"

func TestJWTVerifierValidTokenUsesCache(t *testing.T) {
	t.Parallel()

	privateKey, publicKeyPEM := generateRSAKey(t)

	var mu sync.Mutex
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method != http.MethodGet || r.URL.Path != transitKeyPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		requests++

		writeJSON(t, w, map[string]any{
			"data": map[string]any{
				"keys": map[string]any{
					"1": map[string]any{
						"public_key": publicKeyPEM,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newVaultClient(t, server.URL)

	verifier, err := NewJWTVerifier(&JWTVerifierConfig{
		VaultClient: client,
		KeyName:     "service-jwt-key",
		CacheTTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTVerifier error: %v", err)
	}

	tokenString := makeSignedToken(t, privateKey, map[string]any{
		"sub": "user-123",
		"aud": "gateway",
	}, map[string]any{
		"kid": "service-jwt-key-v1",
	})

	ctx := context.Background()
	claims := &jwt.RegisteredClaims{}

	token, err := verifier.Verify(ctx, tokenString, claims)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !token.Valid {
		t.Fatalf("expected token to be valid")
	}
	if claims.Subject != "user-123" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}

	// Second verification should use cache (no extra HTTP requests).
	if _, err := verifier.Verify(ctx, tokenString, &jwt.RegisteredClaims{}); err != nil {
		t.Fatalf("second Verify returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("expected 1 key fetch, got %d", requests)
	}
}

func TestJWTVerifierRefreshesOnUnknownKid(t *testing.T) {
	t.Parallel()

	privateKeyV1, publicKeyPEMV1 := generateRSAKey(t)
	privateKeyV2, publicKeyPEMV2 := generateRSAKey(t)

	var mu sync.Mutex
	requestCount := 0
	currentVersion := "1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method != http.MethodGet || r.URL.Path != transitKeyPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		requestCount++

		data := map[string]any{
			"1": map[string]any{"public_key": publicKeyPEMV1},
		}
		if currentVersion == "2" {
			data["2"] = map[string]any{"public_key": publicKeyPEMV2}
		}

		writeJSON(t, w, map[string]any{
			"data": map[string]any{
				"keys": data,
			},
		})
	}))
	defer server.Close()

	client := newVaultClient(t, server.URL)

	verifier, err := NewJWTVerifier(&JWTVerifierConfig{
		VaultClient: client,
		KeyName:     "service-jwt-key",
		CacheTTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTVerifier error: %v", err)
	}

	ctx := context.Background()

	tokenV1 := makeSignedToken(t, privateKeyV1, map[string]any{"sub": "user"}, map[string]any{"kid": "service-jwt-key-v1"})
	if _, err := verifier.Verify(ctx, tokenV1, &jwt.RegisteredClaims{}); err != nil {
		t.Fatalf("verify v1 failed: %v", err)
	}

	mu.Lock()
	if requestCount != 1 {
		t.Fatalf("expected 1 fetch after first verification, got %d", requestCount)
	}
	currentVersion = "2"
	mu.Unlock()

	tokenV2 := makeSignedToken(t, privateKeyV2, map[string]any{"sub": "user"}, map[string]any{"kid": "service-jwt-key-v2"})

	if _, err := verifier.Verify(ctx, tokenV2, &jwt.RegisteredClaims{}); err != nil {
		t.Fatalf("verify v2 failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCount != 2 {
		t.Fatalf("expected second fetch after rotation, got %d", requestCount)
	}
}

func TestJWTVerifierRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	_, publicKeyPEM := generateRSAKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != transitKeyPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"data": map[string]any{
				"keys": map[string]any{
					"1": map[string]any{
						"public_key": publicKeyPEM,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newVaultClient(t, server.URL)

	verifier, err := NewJWTVerifier(&JWTVerifierConfig{
		VaultClient: client,
		KeyName:     "service-jwt-key",
		CacheTTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTVerifier error: %v", err)
	}

	// Token signed with a different key.
	otherKey, _ := generateRSAKey(t)
	tokenString := makeSignedToken(t, otherKey, map[string]any{"sub": "evil"}, map[string]any{"kid": "service-jwt-key-v1"})

	_, err = verifier.Verify(context.Background(), tokenString, &jwt.RegisteredClaims{})
	if err == nil {
		t.Fatalf("expected verification to fail for invalid signature")
	}
}

func newVaultClient(t *testing.T, addr string) *vaultapi.Client {
	t.Helper()

	cfg := vaultapi.DefaultConfig()
	cfg.Address = addr

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		t.Fatalf("create vault client: %v", err)
	}
	client.SetToken("dummy-token")

	return client
}

func makeSignedToken(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any, header map[string]any) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	for k, v := range header {
		token.Header[k] = v
	}

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func generateRSAKey(t *testing.T) (priv *rsa.PrivateKey, pemString string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})

	priv = key
	pemString = string(pemBytes)
	return
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}
