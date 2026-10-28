package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"horaveda-app/internal/config"
	"horaveda-app/internal/delivery/http/handler"
	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/internal/domain"
	"horaveda-app/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type routerStubRepo struct{}

func (r *routerStubRepo) FindSerialKey(serialKey string) (*domain.SerialKey, error) {
	return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: false}, nil
}
func (r *routerStubRepo) CreateUser(username string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: username}, nil
}
func (r *routerStubRepo) CreateSupabaseAuthUser(email, password, username string) error {
	return nil
}
func (r *routerStubRepo) UpdateSerialKeyStatus(keyID int, userID int) error {
	return nil
}
func (r *routerStubRepo) GetAllSerialKeys() ([]map[string]interface{}, error) {
	return nil, nil
}

func TestNewRouter(t *testing.T) {
	log := zap.NewNop().Sugar()
	secret := []byte("router-test-secret")
	cfg := &config.Config{
		Logger:    log,
		JWTSecret: secret,
	}
	svc := service.NewAuthService(&routerStubRepo{}, log, secret)
	authHandler := handler.NewAuthHandler(svc, log, cfg)
	authMiddleware := middleware.NewAuthMiddleware(svc, log)

	mux := NewRouter(authHandler, authMiddleware, log)

	t.Run("POST /api/v1/auth/register-serial", func(t *testing.T) {
		body := `{"username": "testuser01", "serial_key": "MALIN-VIP-001"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register-serial", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST /api/v1/auth/logout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("GET /api/v1/auth/verify with valid cookie", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  1,
			"username": "testuser01",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}
