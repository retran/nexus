package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/retran/nexus/backend/internal/secrets"
)

func TestTokenClientIssueAndVerify(t *testing.T) {
	t.Parallel()

	server := newVaultTransitServer(t)
	defer server.Close()

	privateKey, publicPEM := generateRSAKey(t)
	server.addKeyVersion(1, privateKey, publicPEM)

	secretsClient := newTestSecretsClient(t, server.URL())

	tokenClient, err := NewTokenClient(&TokenClientConfig{
		SecretsClient:    secretsClient,
		SigningKeyName:   server.keyName,
		TransitMountPath: "transit",
		Issuer:           "nexus-core",
		VersionCacheTTL:  time.Minute,
	})
	if err != nil {
		t.Fatalf("NewTokenClient error: %v", err)
	}

	ctx := context.Background()
	tokenResp, err := tokenClient.IssueToken(ctx, &IssueTokenRequest{
		Subject:  "service:webhooks",
		Audience: []string{"nexus-api"},
		TTL:      5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}
	if tokenResp.KeyVersion != 1 {
		t.Fatalf("expected key version 1, got %d", tokenResp.KeyVersion)
	}
	if tokenResp.Token == "" {
		t.Fatalf("token is empty")
	}

	if signs := server.signCount(); signs != 1 {
		t.Fatalf("expected 1 sign request during issue, got %d", signs)
	}

	vaultClient := newVaultClientForVerifier(t, server.URL(), server.token)
	verifier, err := NewJWTVerifier(&JWTVerifierConfig{
		VaultClient: vaultClient,
		KeyName:     server.keyName,
	})
	if err != nil {
		t.Fatalf("NewJWTVerifier error: %v", err)
	}

	claims := &jwt.RegisteredClaims{}
	parsed, err := verifier.Verify(ctx, tokenResp.Token, claims)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !parsed.Valid {
		t.Fatalf("expected token to be valid")
	}
	if claims.Subject != "service:webhooks" {
		t.Fatalf("unexpected subject %q", claims.Subject)
	}
	found := false
	for _, aud := range claims.Audience {
		if aud == "nexus-api" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected audience nexus-api")
	}
	if claims.Issuer != "nexus-core" {
		t.Fatalf("unexpected issuer %q", claims.Issuer)
	}
}

func TestTokenClientRefreshesKeyVersion(t *testing.T) {
	t.Parallel()

	server := newVaultTransitServer(t)
	defer server.Close()

	privV1, pubV1 := generateRSAKey(t)
	privV2, pubV2 := generateRSAKey(t)
	server.addKeyVersion(1, privV1, pubV1)

	fakeClock := newFakeClock(time.Unix(0, 0))

	secretsClient := newTestSecretsClient(t, server.URL())
	tokenClient, err := NewTokenClient(&TokenClientConfig{
		SecretsClient:    secretsClient,
		SigningKeyName:   server.keyName,
		TransitMountPath: "transit",
		VersionCacheTTL:  30 * time.Second,
		Clock:            fakeClock.Now,
	})
	if err != nil {
		t.Fatalf("NewTokenClient error: %v", err)
	}

	ctx := context.Background()

	first, err := tokenClient.IssueToken(ctx, &IssueTokenRequest{
		Subject:  "service:gateway",
		Audience: []string{"gateway"},
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("first IssueToken error: %v", err)
	}
	if first.KeyVersion != 1 {
		t.Fatalf("expected first token key version 1, got %d", first.KeyVersion)
	}

	initialReads := server.keyReads()

	// Advance time past the cache TTL to force metadata refresh.
	fakeClock.Advance(45 * time.Second)
	server.addKeyVersion(2, privV2, pubV2)
	server.setCurrentVersion(2)

	second, err := tokenClient.IssueToken(ctx, &IssueTokenRequest{
		Subject:  "service:gateway",
		Audience: []string{"gateway"},
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("second IssueToken error: %v", err)
	}
	if second.KeyVersion != 2 {
		t.Fatalf("expected second token key version 2, got %d", second.KeyVersion)
	}

	if server.keyReads() != initialReads+1 {
		t.Fatalf("expected one additional key metadata fetch after rotation; got %d total", server.keyReads())
	}

	segments := strings.Split(second.Token, ".")
	if len(segments) != 3 {
		t.Fatalf("invalid token structure: %q", second.Token)
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		t.Fatalf("decode token header: %v", err)
	}
	if !strings.Contains(string(headerBytes), "\"kid\":\"service-jwt-key-v2\"") {
		t.Fatalf("expected token header to include kid for v2, got %s", string(headerBytes))
	}
}

func TestTokenClientRejectsReservedClaims(t *testing.T) {
	t.Parallel()

	server := newVaultTransitServer(t)
	defer server.Close()

	priv, pub := generateRSAKey(t)
	server.addKeyVersion(1, priv, pub)

	secretsClient := newTestSecretsClient(t, server.URL())
	tokenClient, err := NewTokenClient(&TokenClientConfig{
		SecretsClient:  secretsClient,
		SigningKeyName: server.keyName,
	})
	if err != nil {
		t.Fatalf("NewTokenClient error: %v", err)
	}

	_, err = tokenClient.IssueToken(context.Background(), &IssueTokenRequest{
		Subject:  "user-123",
		Audience: []string{"api"},
		TTL:      time.Minute,
		AdditionalClaims: map[string]any{
			"sub": "override",
		},
	})
	if err == nil {
		t.Fatalf("expected error when overriding reserved claim")
	}
}

// --- Test helpers ---

//nolint:govet // fieldalignment: test helper prioritises readability over padding savings.
type vaultTransitServer struct {
	mu sync.Mutex

	t      *testing.T
	server *httptest.Server

	privateKeys map[int]*rsa.PrivateKey
	publicPEMs  map[int]string

	keyName  string
	roleID   string
	secretID string
	token    string

	currentVersion   int
	keyReadCount     int
	signRequestCount int
	loginCount       int
}

func newVaultTransitServer(t *testing.T) *vaultTransitServer {
	t.Helper()

	s := &vaultTransitServer{
		t:              t,
		keyName:        "service-jwt-key",
		roleID:         "role-id",
		secretID:       "secret-id",
		token:          "vault-development-token",
		privateKeys:    make(map[int]*rsa.PrivateKey),
		publicPEMs:     make(map[int]string),
		currentVersion: 1,
	}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/approle/login" && r.Method == http.MethodPut:
			s.handleLogin(w, r)
		case r.URL.Path == "/v1/transit/keys/"+s.keyName && r.Method == http.MethodGet:
			s.handleKeyMetadata(w, r)
		case r.URL.Path == "/v1/transit/sign/"+s.keyName:
			s.handleSign(w, r)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	return s
}

func (s *vaultTransitServer) Close() {
	s.server.Close()
}

func (s *vaultTransitServer) URL() string {
	return s.server.URL
}

func (s *vaultTransitServer) addKeyVersion(version int, priv *rsa.PrivateKey, publicPEM string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.privateKeys[version] = priv
	s.publicPEMs[version] = publicPEM
	if version > s.currentVersion {
		s.currentVersion = version
	}
}

func (s *vaultTransitServer) setCurrentVersion(version int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentVersion = version
}

func (s *vaultTransitServer) keyReads() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keyReadCount
}

func (s *vaultTransitServer) signCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.signRequestCount
}

func (s *vaultTransitServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loginCount++

	var body struct {
		RoleID   string `json:"role_id"`
		SecretID string `json:"secret_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.t.Fatalf("decode login body: %v", err)
	}
	if body.RoleID != s.roleID || body.SecretID != s.secretID {
		http.Error(w, "invalid credentials", http.StatusForbidden)
		return
	}

	writeJSON(s.t, w, map[string]any{
		"auth": map[string]any{
			"client_token": s.token,
		},
	})
}

func (s *vaultTransitServer) handleKeyMetadata(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyReadCount++

	keys := make(map[string]any, len(s.publicPEMs))
	for version, pem := range s.publicPEMs {
		keys[strconv.Itoa(version)] = map[string]any{
			"public_key": pem,
		}
	}

	writeJSON(s.t, w, map[string]any{
		"data": map[string]any{
			"latest_version": s.currentVersion,
			"keys":           keys,
		},
	})
}

func (s *vaultTransitServer) handleSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		s.t.Fatalf("unexpected method for sign: %s", r.Method)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.signRequestCount++

	var body struct {
		Input      string `json:"input"`
		KeyVersion int    `json:"key_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.t.Fatalf("decode sign body: %v", err)
	}

	version := body.KeyVersion
	if version == 0 {
		version = s.currentVersion
	}

	key, ok := s.privateKeys[version]
	if !ok {
		http.Error(w, "unknown key version", http.StatusBadRequest)
		return
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(body.Input)
	if err != nil {
		s.t.Fatalf("decode sign input: %v", err)
	}
	hash := sha256.Sum256(payloadBytes)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		s.t.Fatalf("sign payload: %v", err)
	}

	writeJSON(s.t, w, map[string]any{
		"data": map[string]any{
			"signature":   fmt.Sprintf("vault:v%d:%s", version, base64.StdEncoding.EncodeToString(sig)),
			"key_version": version,
		},
	})
}

func newTestSecretsClient(t *testing.T, addr string) *secrets.Client {
	t.Helper()

	client, err := secrets.NewClient(&secrets.Config{
		Address:  addr,
		RoleID:   "role-id",
		SecretID: "secret-id",
	})
	if err != nil {
		t.Fatalf("secrets.NewClient error: %v", err)
	}
	return client
}

func newVaultClientForVerifier(t *testing.T, addr, token string) *vaultapi.Client {
	t.Helper()

	cfg := vaultapi.DefaultConfig()
	cfg.Address = addr
	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetToken(token)
	return client
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
