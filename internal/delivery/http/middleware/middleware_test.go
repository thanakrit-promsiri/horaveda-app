package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"horaveda-app/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func TestCORSMiddleware(t *testing.T) {
	allowedOrigin := "https://hora.garbagecollection.info"
	cors := CORSMiddleware(allowedOrigin)

	t.Run("normal GET request passes through with CORS headers", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		rr := httptest.NewRecorder()

		cors(next).ServeHTTP(rr, req)

		if !called {
			t.Error("expected next handler to be called")
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
			t.Errorf("expected Access-Control-Allow-Origin %q, got %q", allowedOrigin, got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials true, got %q", got)
		}
	})

	t.Run("OPTIONS preflight request returns 200 without calling next", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register-serial", nil)
		rr := httptest.NewRecorder()

		cors(next).ServeHTTP(rr, req)

		if called {
			t.Error("expected next handler NOT to be called on OPTIONS")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

func TestGetUserClaims(t *testing.T) {
	t.Run("from UserClaimsKey with jwt.MapClaims", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserClaimsKey, jwt.MapClaims{"username": "alice"})
		claims, ok := GetUserClaims(ctx)
		if !ok || claims["username"] != "alice" {
			t.Errorf("expected alice, got %v (ok=%v)", claims, ok)
		}
	})

	t.Run("from string key with map[string]interface{}", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "userClaims", map[string]interface{}{"username": "bob"})
		claims, ok := GetUserClaims(ctx)
		if !ok || claims["username"] != "bob" {
			t.Errorf("expected bob, got %v (ok=%v)", claims, ok)
		}
	})

	t.Run("missing claims", func(t *testing.T) {
		_, ok := GetUserClaims(context.Background())
		if ok {
			t.Error("expected ok=false for empty context")
		}
	})
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	secret := []byte("middleware-test-secret")
	log := zap.NewNop().Sugar()
	authSvc := service.NewAuthService(nil, log, secret)
	mw := NewAuthMiddleware(authSvc, log)

	t.Run("missing auth_token cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		rr := httptest.NewRecorder()

		handler := mw.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not reach protected handler")
		})
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("invalid auth_token cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: "invalid-token"})
		rr := httptest.NewRecorder()

		handler := mw.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not reach protected handler")
		})
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("valid auth_token cookie attaches claims and calls next", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  1,
			"username": "testuser01",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rr := httptest.NewRecorder()

		var capturedUsername string
		handler := mw.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok {
				t.Fatal("expected userClaims in context")
			}
			capturedUsername, _ = claims["username"].(string)
			w.WriteHeader(http.StatusOK)
		})
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if capturedUsername != "testuser01" {
			t.Errorf("expected username testuser01, got %q", capturedUsername)
		}
	})
}

func TestAuthMiddleware_RequireAdminKey(t *testing.T) {
	log := zap.NewNop().Sugar()
	mw := NewAuthMiddleware(nil, log)
	adminKey := "super-secret-admin-key"

	t.Run("unconfigured adminKey returns 500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", nil)
		rr := httptest.NewRecorder()

		mw.RequireAdminKey("", func(w http.ResponseWriter, r *http.Request) {})(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("missing header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", nil)
		rr := httptest.NewRecorder()

		mw.RequireAdminKey(adminKey, func(w http.ResponseWriter, r *http.Request) {})(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("wrong key returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", nil)
		req.Header.Set("X-Admin-Key", "wrong-key")
		rr := httptest.NewRecorder()

		mw.RequireAdminKey(adminKey, func(w http.ResponseWriter, r *http.Request) {})(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rr.Code)
		}
	})

	t.Run("valid X-Admin-Key succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", nil)
		req.Header.Set("X-Admin-Key", adminKey)
		rr := httptest.NewRecorder()

		called := false
		mw.RequireAdminKey(adminKey, func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})(rr, req)

		if !called || rr.Code != http.StatusOK {
			t.Errorf("expected handler called and 200 OK, got called=%v code=%d", called, rr.Code)
		}
	})

	t.Run("valid Authorization Bearer succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", nil)
		req.Header.Set("Authorization", "Bearer "+adminKey)
		rr := httptest.NewRecorder()

		called := false
		mw.RequireAdminKey(adminKey, func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})(rr, req)

		if !called || rr.Code != http.StatusOK {
			t.Errorf("expected handler called and 200 OK, got called=%v code=%d", called, rr.Code)
		}
	})
}
