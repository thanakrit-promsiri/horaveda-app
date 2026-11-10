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
func (r *routerStubRepo) FindUserByUsername(username string) (*domain.User, error) {
	return &domain.User{ID: 38, Username: username}, nil
}
func (r *routerStubRepo) CreateUser(username string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: username}, nil
}
func (r *routerStubRepo) DeleteUser(userID int) error {
	return nil
}
func (r *routerStubRepo) CreateSupabaseAuthUser(email, password, username string) (string, error) {
	return "00000000-0000-0000-0000-000000000001", nil
}
func (r *routerStubRepo) DeleteSupabaseAuthUser(authUserID string) error {
	return nil
}
func (r *routerStubRepo) DeleteSupabaseAuthUserByEmail(email string) error {
	return nil
}
func (r *routerStubRepo) UpdateSerialKeyStatus(keyID int, userID int) error {
	return nil
}
func (r *routerStubRepo) RollbackSerialKeyStatus(keyID int) error {
	return nil
}
func (r *routerStubRepo) RevokeSerialKeysByUserID(userID int) ([]domain.SerialKey, error) {
	return []domain.SerialKey{{ID: 1, KeyCode: "BOOM-VIP-001"}}, nil
}
func (r *routerStubRepo) GetAllSerialKeys() ([]map[string]interface{}, error) {
	return nil, nil
}

func TestNewRouter(t *testing.T) {
	log := zap.NewNop().Sugar()
	secret := []byte("router-test-secret")
	cfg := &config.Config{
		Logger:      log,
		JWTSecret:   secret,
		AdminAPIKey: "test-admin-key",
	}
	svc := service.NewAuthService(&routerStubRepo{}, log, secret)
	authHandler := handler.NewAuthHandler(svc, log, cfg)
	authMiddleware := middleware.NewAuthMiddleware(svc, log)

	mux := NewRouter(authHandler, authMiddleware, log)

	t.Run("POST /api/v1/auth/register-serial", func(t *testing.T) {
		body := `{"username": "testuser01", "serial_key": "BOOM-VIP-001"}`
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

	t.Run("GET /api/v1/auth/serial-keys is public without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/serial-keys", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 without token, got %d", rr.Code)
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

	t.Run("POST /api/v1/auth/revoke-user with X-Admin-Key", func(t *testing.T) {
		body := `{"username": "testuser01"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", bytes.NewBufferString(body))
		req.Header.Set("X-Admin-Key", "test-admin-key")
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST /api/v1/hora/predict requires auth and returns prompt", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: routerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				rr := httptest.NewRecorder()
				rr.Header().Set("Content-Type", "application/json")
				rr.WriteHeader(http.StatusOK)
				_, _ = rr.Write([]byte(`{"status":"success","prompt":"[Vedic Sidereal(Lahiri)/WholeSign]"}`))
				return rr.Result(), nil
			}),
		}

		authHandler.HoraService = service.NewHoraService("http://localhost:8083/api/v1/hora/prompt", log, mockClient)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  1,
			"username": "testuser01",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)

		body := `{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/hora/predict", bytes.NewBufferString(body))
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST /api/v1/hora/transitpredict requires auth and returns 200", func(t *testing.T) {
		// 1. Without auth -> 401
		body := `{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018, "transit_year": 2026, "transit_month": 12}`
		reqNoAuth := httptest.NewRequest(http.MethodPost, "/api/v1/hora/transitpredict", bytes.NewBufferString(body))
		rrNoAuth := httptest.NewRecorder()
		mux.ServeHTTP(rrNoAuth, reqNoAuth)
		if rrNoAuth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth token, got %d", rrNoAuth.Code)
		}

		// 2. With auth -> 200
		mockClient := &http.Client{
			Transport: routerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				rr := httptest.NewRecorder()
				rr.Header().Set("Content-Type", "application/json")
				rr.WriteHeader(http.StatusOK)
				_, _ = rr.Write([]byte(`{"status":"success","prompt":"[Prompt Result]"}`))
				return rr.Result(), nil
			}),
		}

		authHandler.HoraService = service.NewHoraServiceWithTransit(
			"http://localhost:8083/api/v1/hora/prompt",
			"http://localhost:8083/api/v1/hora/transitdasha_prompt",
			log,
			mockClient,
		)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  1,
			"username": "testuser01",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)

		reqAuth := httptest.NewRequest(http.MethodPost, "/api/v1/hora/transitpredict", bytes.NewBufferString(body))
		reqAuth.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rrAuth := httptest.NewRecorder()

		mux.ServeHTTP(rrAuth, reqAuth)

		if rrAuth.Code != http.StatusOK {
			t.Errorf("expected 200 with auth token, got %d: %s", rrAuth.Code, rrAuth.Body.String())
		}
	})

	t.Run("POST /api/v1/hora/transitpredict_1 and _12 fix transit_year=2027 and month", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  1,
			"username": "testuser01",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)

		// ส่ง body โดยไม่ต้องมี transit_year หรือ transit_month
		simpleBody := `{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`

		// ทดสอบเดือน 1 (มกราคม 2570)
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/hora/transitpredict_1", bytes.NewBufferString(simpleBody))
		req1.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rr1 := httptest.NewRecorder()
		mux.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Fatalf("expected 200 for transitpredict_1, got %d: %s", rr1.Code, rr1.Body.String())
		}

		// ทดสอบเดือน 12 (ธันวาคม 2570)
		req12 := httptest.NewRequest(http.MethodPost, "/api/v1/hora/transitpredict_12", bytes.NewBufferString(simpleBody))
		req12.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenStr})
		rr12 := httptest.NewRecorder()
		mux.ServeHTTP(rr12, req12)
		if rr12.Code != http.StatusOK {
			t.Fatalf("expected 200 for transitpredict_12, got %d: %s", rr12.Code, rr12.Body.String())
		}
	})
}

type routerRoundTripFunc func(req *http.Request) (*http.Response, error)

func (f routerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
