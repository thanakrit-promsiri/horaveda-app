package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"horaveda-app/internal/config"
	"horaveda-app/internal/domain"
	"horaveda-app/internal/service"

	"go.uber.org/zap"
)

type stubAuthRepo struct {
	findSerialKeyFn                 func(serialKey string) (*domain.SerialKey, error)
	findUserByUsernameFn            func(username string) (*domain.User, error)
	createUserFn                    func(username string) (*domain.User, error)
	createSupabaseAuthUserFn        func(email, password, username string) (string, error)
	updateSerialKeyStatusFn         func(keyID int, userID int) error
	revokeSerialKeysByUserIDFn      func(userID int) ([]domain.SerialKey, error)
	deleteSupabaseAuthUserByEmailFn func(email string) error
	getAllSerialKeysFn              func() ([]map[string]interface{}, error)
}

func (s *stubAuthRepo) FindSerialKey(serialKey string) (*domain.SerialKey, error) {
	return s.findSerialKeyFn(serialKey)
}
func (s *stubAuthRepo) FindUserByUsername(username string) (*domain.User, error) {
	if s.findUserByUsernameFn != nil {
		return s.findUserByUsernameFn(username)
	}
	return &domain.User{ID: 38, Username: username}, nil
}
func (s *stubAuthRepo) CreateUser(username string) (*domain.User, error) {
	return s.createUserFn(username)
}
func (s *stubAuthRepo) DeleteUser(userID int) error {
	return nil
}
func (s *stubAuthRepo) CreateSupabaseAuthUser(email, password, username string) (string, error) {
	return s.createSupabaseAuthUserFn(email, password, username)
}
func (s *stubAuthRepo) DeleteSupabaseAuthUser(authUserID string) error {
	return nil
}
func (s *stubAuthRepo) DeleteSupabaseAuthUserByEmail(email string) error {
	if s.deleteSupabaseAuthUserByEmailFn != nil {
		return s.deleteSupabaseAuthUserByEmailFn(email)
	}
	return nil
}
func (s *stubAuthRepo) UpdateSerialKeyStatus(keyID int, userID int) error {
	return s.updateSerialKeyStatusFn(keyID, userID)
}
func (s *stubAuthRepo) RollbackSerialKeyStatus(keyID int) error {
	return nil
}
func (s *stubAuthRepo) RevokeSerialKeysByUserID(userID int) ([]domain.SerialKey, error) {
	if s.revokeSerialKeysByUserIDFn != nil {
		return s.revokeSerialKeysByUserIDFn(userID)
	}
	return []domain.SerialKey{{ID: 1, KeyCode: "BOOM-VIP-001"}}, nil
}
func (s *stubAuthRepo) GetAllSerialKeys() ([]map[string]interface{}, error) {
	return s.getAllSerialKeysFn()
}

func setupHandler(repo *stubAuthRepo) *AuthHandler {
	log := zap.NewNop().Sugar()
	cfg := &config.Config{
		Logger:      log,
		Port:        "8080",
		JWTSecret:   []byte("handler-secret"),
		CORSOrigin:  "*",
		AdminAPIKey: "admin-key",
	}
	svc := service.NewAuthService(repo, log, cfg.JWTSecret)
	return NewAuthHandler(svc, log, cfg)
}

func TestHandleSerialRegister(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register-serial", nil)
		rr := httptest.NewRecorder()

		h.HandleSerialRegister(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register-serial", bytes.NewBufferString(`{invalid`))
		rr := httptest.NewRecorder()

		h.HandleSerialRegister(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("non-alphanumeric username", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		body := `{"username": "ผู้ใช้_01!", "serial_key": "BOOM-VIP-001"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register-serial", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		h.HandleSerialRegister(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("service returns AppError (404 not found)", func(t *testing.T) {
		repo := &stubAuthRepo{
			findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
				return nil, errors.New("not found")
			},
		}
		h := setupHandler(repo)
		body := `{"username": "testuser01", "serial_key": "MISSING-KEY"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register-serial", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		h.HandleSerialRegister(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("successful registration sets auth_token cookie", func(t *testing.T) {
		repo := &stubAuthRepo{
			findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
				return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: false}, nil
			},
			createUserFn: func(username string) (*domain.User, error) {
				return &domain.User{ID: 7, Username: username}, nil
			},
			createSupabaseAuthUserFn: func(email, password, username string) (string, error) {
				return "00000000-0000-0000-0000-000000000001", nil
			},
			updateSerialKeyStatusFn: func(keyID int, userID int) error {
				return nil
			},
		}
		h := setupHandler(repo)
		body := `{"username": "testuser01", "serial_key": "BOOM-VIP-001"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register-serial", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		h.HandleSerialRegister(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var foundCookie *http.Cookie
		for _, c := range rr.Result().Cookies() {
			if c.Name == "auth_token" {
				foundCookie = c
				break
			}
		}
		if foundCookie == nil || foundCookie.Value == "" {
			t.Fatal("expected non-empty auth_token cookie")
		}
		if !foundCookie.HttpOnly {
			t.Error("expected HttpOnly cookie")
		}
	})
}

func TestHandleRevokeUser(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/revoke-user", nil)
		rr := httptest.NewRecorder()

		h.HandleRevokeUser(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", bytes.NewBufferString(`{"username": ""}`))
		rr := httptest.NewRecorder()

		h.HandleRevokeUser(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("user not found returns 404", func(t *testing.T) {
		repo := &stubAuthRepo{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return nil, errors.New("not found")
			},
		}
		h := setupHandler(repo)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", bytes.NewBufferString(`{"username": "testuser01"}`))
		rr := httptest.NewRecorder()

		h.HandleRevokeUser(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/revoke-user", bytes.NewBufferString(`{"username": "testuser01"}`))
		rr := httptest.NewRecorder()

		h.HandleRevokeUser(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestHandleVerifyAuth(t *testing.T) {
	h := setupHandler(&stubAuthRepo{})

	t.Run("with userClaims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		ctx := context.WithValue(req.Context(), "userClaims", map[string]interface{}{
			"username": "testuser01",
		})
		rr := httptest.NewRecorder()

		h.HandleVerifyAuth(rr, req.WithContext(ctx))

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("without userClaims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
		rr := httptest.NewRecorder()

		h.HandleVerifyAuth(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestHandleLogout(t *testing.T) {
	h := setupHandler(&stubAuthRepo{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.HandleLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var foundCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "auth_token" {
			foundCookie = c
			break
		}
	}
	if foundCookie == nil {
		t.Fatal("expected auth_token cookie to be set for clearing")
	}
	if foundCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", foundCookie.MaxAge)
	}
}

func TestHandleDebugGetAllKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &stubAuthRepo{
			getAllSerialKeysFn: func() ([]map[string]interface{}, error) {
				return []map[string]interface{}{{"id": 1, "key_code": "KEY-01"}}, nil
			},
		}
		h := setupHandler(repo)
		req := httptest.NewRequest(http.MethodGet, "/debug/keys", nil)
		rr := httptest.NewRecorder()

		h.HandleDebugGetAllKeys(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		repo := &stubAuthRepo{
			getAllSerialKeysFn: func() ([]map[string]interface{}, error) {
				return nil, errors.New("db failure")
			},
		}
		h := setupHandler(repo)
		req := httptest.NewRequest(http.MethodGet, "/debug/keys", nil)
		rr := httptest.NewRecorder()

		h.HandleDebugGetAllKeys(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHandleHoraPredict(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hora/predict", nil)
		rr := httptest.NewRecorder()

		h.HandleHoraPredict(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/hora/predict", bytes.NewBufferString(`{invalid`))
		rr := httptest.NewRecorder()

		h.HandleHoraPredict(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("invalid field ranges", func(t *testing.T) {
		h := setupHandler(&stubAuthRepo{})
		cases := []string{
			`{"year": 0, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`,
			`{"year": 1988, "month": 13, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`,
			`{"year": 1988, "month": 12, "day": 32, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`,
			`{"year": 1988, "month": 12, "day": 23, "hour_thai": 24.5, "latitude": 13.7563, "longitude": 100.5018}`,
			`{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 95.0, "longitude": 100.5018}`,
			`{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 190.0}`,
		}
		for _, body := range cases {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/hora/predict", bytes.NewBufferString(body))
			rr := httptest.NewRecorder()
			h.HandleHoraPredict(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for body %s, got %d", body, rr.Code)
			}
		}
	})

	t.Run("success calls Hora Prompt API and returns prompt", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				rr := httptest.NewRecorder()
				rr.Header().Set("Content-Type", "application/json")
				rr.WriteHeader(http.StatusOK)
				_, _ = rr.Write([]byte(`{"status":"success","prompt":"[Vedic Sidereal(Lahiri)/WholeSign]"}`))
				return rr.Result(), nil
			}),
		}

		h := setupHandler(&stubAuthRepo{})
		h.HoraService = service.NewHoraService("http://localhost:8083/api/v1/hora/prompt", h.Logger, mockClient)

		body := `{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/hora/predict", bytes.NewBufferString(body))
		ctx := context.WithValue(req.Context(), "userClaims", map[string]interface{}{"username": "testuser01"})
		rr := httptest.NewRecorder()

		h.HandleHoraPredict(rr, req.WithContext(ctx))

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("downstream error returns 500", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				rr := httptest.NewRecorder()
				rr.WriteHeader(http.StatusInternalServerError)
				return rr.Result(), nil
			}),
		}

		h := setupHandler(&stubAuthRepo{})
		h.HoraService = service.NewHoraService("http://localhost:8083/api/v1/hora/prompt", h.Logger, mockClient)

		body := `{"year": 1988, "month": 12, "day": 23, "hour_thai": 10.35, "latitude": 13.7563, "longitude": 100.5018}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/hora/predict", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		h.HandleHoraPredict(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})
}
