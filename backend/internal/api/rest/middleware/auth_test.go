package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

type stubVerifier struct {
	err    error
	claims oathkeeperClaims
}

func (s *stubVerifier) Verify(_ context.Context, _ string, claims jwt.Claims, _ ...jwt.ParserOption) (*jwt.Token, error) {
	if s.err != nil {
		return nil, s.err
	}

	target, ok := claims.(*oathkeeperClaims)
	if !ok {
		return nil, errors.New("unexpected claims type")
	}

	*target = s.claims
	return &jwt.Token{}, nil
}

func TestAuthMiddlewareRequireAuthSuccess(t *testing.T) {
	identityID := "a3e9bb0b-8f6e-4e08-8b51-3327bf16ed3d"
	email := "user@example.com"
	role := "member"

	mw, err := NewAuthMiddleware(Config{
		Verifier: &stubVerifier{
			claims: oathkeeperClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:   "http://auth.nexus.local",
					Subject:  "oathkeeper",
					Audience: jwt.ClaimStrings{"gateway"},
				},
				Session: struct {
					ID       string `json:"id"`
					Identity struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					} "json:\"identity\""
				}{
					ID: "sid-12345",
					Identity: struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					}{
						ID: identityID,
						Traits: struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						}{
							Email: email,
							Role:  role,
							Name: struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							}{
								First: "Test",
								Last:  "User",
							},
						},
					},
				},
			},
		},
		Issuer:   "http://auth.nexus.local",
		Subject:  "oathkeeper",
		Audience: []string{"gateway"},
	})
	if err != nil {
		t.Fatalf("NewAuthMiddleware error: %v", err)
	}

	var captured *AuthInfo
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = AuthInfoFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-User", identityID)
	req.Header.Set("X-User-Email", email)
	req.Header.Set("X-User-Role", role)

	resp := httptest.NewRecorder()
	mw.RequireAuth(next).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if captured == nil {
		t.Fatal("expected auth info in context")
	}
	if captured.UserID.String() != identityID {
		t.Fatalf("expected user id %s, got %s", identityID, captured.UserID)
	}
	if captured.Email != email {
		t.Fatalf("expected email %s, got %s", email, captured.Email)
	}
	if captured.Role != role {
		t.Fatalf("expected role %s, got %s", role, captured.Role)
	}
	if captured.SessionID != "sid-12345" {
		t.Fatalf("expected session id sid-12345, got %s", captured.SessionID)
	}
	if captured.FullName != "Test User" {
		t.Fatalf("expected full name Test User, got %q", captured.FullName)
	}
}

func TestAuthMiddlewareRequireAuthAudienceMismatch(t *testing.T) {
	mw, err := NewAuthMiddleware(Config{
		Verifier: &stubVerifier{
			claims: oathkeeperClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:   "http://auth.nexus.local",
					Subject:  "oathkeeper",
					Audience: jwt.ClaimStrings{"other-service"},
				},
				Session: struct {
					ID       string "json:\"id\""
					Identity struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					} "json:\"identity\""
				}{
					ID: "sid-1",
					Identity: struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					}{
						ID: "c5e1fa56-2c3a-4db3-a165-147a4895c8f3",
						Traits: struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						}{
							Email: "user@example.com",
							Role:  "member",
							Name: struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							}{},
						},
					},
				},
			},
		},
		Issuer:   "http://auth.nexus.local",
		Subject:  "oathkeeper",
		Audience: []string{"gateway"},
	})
	if err != nil {
		t.Fatalf("NewAuthMiddleware error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-User", "c5e1fa56-2c3a-4db3-a165-147a4895c8f3")
	req.Header.Set("X-User-Email", "user@example.com")
	req.Header.Set("X-User-Role", "member")

	resp := httptest.NewRecorder()
	mw.RequireAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareRequireAuthHeaderMismatch(t *testing.T) {
	identityID := "e0cbcf6d-5d08-45f3-a842-26023bcbf2db"

	mw, err := NewAuthMiddleware(Config{
		Verifier: &stubVerifier{
			claims: oathkeeperClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:   "http://auth.nexus.local",
					Subject:  "oathkeeper",
					Audience: jwt.ClaimStrings{"gateway"},
				},
				Session: struct {
					ID       string "json:\"id\""
					Identity struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					} "json:\"identity\""
				}{
					ID: "sid-2",
					Identity: struct {
						ID     string "json:\"id\""
						Traits struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						} "json:\"traits\""
					}{
						ID: identityID,
						Traits: struct {
							Email string "json:\"email\""
							Role  string "json:\"role\""
							Name  struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							} "json:\"name\""
						}{
							Email: "user@example.com",
							Role:  "member",
							Name: struct {
								First string "json:\"first\""
								Last  string "json:\"last\""
							}{},
						},
					},
				},
			},
		},
		Issuer:   "http://auth.nexus.local",
		Subject:  "oathkeeper",
		Audience: []string{"gateway"},
	})
	if err != nil {
		t.Fatalf("NewAuthMiddleware error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-User", "different-id")
	req.Header.Set("X-User-Email", "user@example.com")
	req.Header.Set("X-User-Role", "member")

	resp := httptest.NewRecorder()
	mw.RequireAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareOptionalAuthNoToken(t *testing.T) {
	mw, err := NewAuthMiddleware(Config{
		Verifier: &stubVerifier{},
	})
	if err != nil {
		t.Fatalf("NewAuthMiddleware error: %v", err)
	}

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/open", http.NoBody)
	resp := httptest.NewRecorder()

	mw.OptionalAuth(next).ServeHTTP(resp, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}
