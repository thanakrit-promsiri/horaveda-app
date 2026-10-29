package service

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type mockAuthRepository struct {
	findSerialKeyFn           func(serialKey string) (*domain.SerialKey, error)
	createUserFn              func(username string) (*domain.User, error)
	deleteUserFn              func(userID int) error
	createSupabaseAuthUserFn  func(email, password, username string) (string, error)
	deleteSupabaseAuthUserFn  func(authUserID string) error
	updateSerialKeyStatusFn   func(keyID int, userID int) error
	rollbackSerialKeyStatusFn func(keyID int) error
	getAllSerialKeysFn        func() ([]map[string]interface{}, error)
}

func (m *mockAuthRepository) FindSerialKey(serialKey string) (*domain.SerialKey, error) {
	return m.findSerialKeyFn(serialKey)
}

func (m *mockAuthRepository) CreateUser(username string) (*domain.User, error) {
	return m.createUserFn(username)
}

func (m *mockAuthRepository) DeleteUser(userID int) error {
	if m.deleteUserFn != nil {
		return m.deleteUserFn(userID)
	}
	return nil
}

func (m *mockAuthRepository) CreateSupabaseAuthUser(email, password, username string) (string, error) {
	return m.createSupabaseAuthUserFn(email, password, username)
}

func (m *mockAuthRepository) DeleteSupabaseAuthUser(authUserID string) error {
	if m.deleteSupabaseAuthUserFn != nil {
		return m.deleteSupabaseAuthUserFn(authUserID)
	}
	return nil
}

func (m *mockAuthRepository) UpdateSerialKeyStatus(keyID int, userID int) error {
	return m.updateSerialKeyStatusFn(keyID, userID)
}

func (m *mockAuthRepository) RollbackSerialKeyStatus(keyID int) error {
	if m.rollbackSerialKeyStatusFn != nil {
		return m.rollbackSerialKeyStatusFn(keyID)
	}
	return nil
}

func (m *mockAuthRepository) GetAllSerialKeys() ([]map[string]interface{}, error) {
	return m.getAllSerialKeysFn()
}

func newTestLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

func TestRegisterWithSerial_Success(t *testing.T) {
	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return &domain.SerialKey{ID: 10, KeyCode: serialKey, IsUsed: false}, nil
		},
		createUserFn: func(username string) (*domain.User, error) {
			return &domain.User{ID: 7, Username: username}, nil
		},
		createSupabaseAuthUserFn: func(email, password, username string) (string, error) {
			if email != "testuser01@garbagecollection.info" {
				t.Errorf("unexpected email: %s", email)
			}
			return "00000000-0000-0000-0000-000000000001", nil
		},
		updateSerialKeyStatusFn: func(keyID int, userID int) error {
			if keyID != 10 || userID != 7 {
				t.Errorf("unexpected keyID=%d or userID=%d", keyID, userID)
			}
			return nil
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	token, err := svc.RegisterWithSerial("testuser01", "MALIN-VIP-001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT token")
	}

	claims, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	if claims["username"] != "testuser01" {
		t.Errorf("expected username testuser01, got %v", claims["username"])
	}
}

func TestRegisterWithSerial_KeyNotFound(t *testing.T) {
	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	_, err := svc.RegisterWithSerial("testuser01", "INVALID-KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*response.AppError)
	if !ok || appErr.Code != http.StatusNotFound {
		t.Errorf("expected 404 AppError, got %+v", err)
	}
}

func TestRegisterWithSerial_KeyAlreadyUsed(t *testing.T) {
	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: true}, nil
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	_, err := svc.RegisterWithSerial("testuser01", "USED-KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*response.AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 AppError, got %+v", err)
	}
}

func TestRegisterWithSerial_DuplicateUsername(t *testing.T) {
	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: false}, nil
		},
		createUserFn: func(username string) (*domain.User, error) {
			return nil, errors.New("duplicate key")
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	_, err := svc.RegisterWithSerial("existinguser", "VALID-KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*response.AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 AppError, got %+v", err)
	}
}

func TestRegisterWithSerial_SupabaseAuthFailure_RollsBackUser(t *testing.T) {
	var rolledBackUserID int
	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: false}, nil
		},
		createUserFn: func(username string) (*domain.User, error) {
			return &domain.User{ID: 5, Username: username}, nil
		},
		deleteUserFn: func(userID int) error {
			rolledBackUserID = userID
			return errors.New("simulated rollback log test")
		},
		createSupabaseAuthUserFn: func(email, password, username string) (string, error) {
			return "", errors.New("supabase auth error")
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	_, err := svc.RegisterWithSerial("testuser", "VALID-KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if rolledBackUserID != 5 {
		t.Errorf("expected rollback DeleteUser(5) to be called, got %d", rolledBackUserID)
	}
	appErr, ok := err.(*response.AppError)
	if !ok || appErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 AppError, got %+v", err)
	}
}

func TestRegisterWithSerial_UpdateSerialKeyStatusFailure_RollsBackBothUsers(t *testing.T) {
	var rolledBackUserID int
	var rolledBackAuthUserID string

	repo := &mockAuthRepository{
		findSerialKeyFn: func(serialKey string) (*domain.SerialKey, error) {
			return &domain.SerialKey{ID: 1, KeyCode: serialKey, IsUsed: false}, nil
		},
		createUserFn: func(username string) (*domain.User, error) {
			return &domain.User{ID: 5, Username: username}, nil
		},
		deleteUserFn: func(userID int) error {
			rolledBackUserID = userID
			return nil
		},
		createSupabaseAuthUserFn: func(email, password, username string) (string, error) {
			return "00000000-0000-0000-0000-000000000099", nil
		},
		deleteSupabaseAuthUserFn: func(authUserID string) error {
			rolledBackAuthUserID = authUserID
			return errors.New("simulated auth rollback log test")
		},
		updateSerialKeyStatusFn: func(keyID int, userID int) error {
			return errors.New("update status failed")
		},
	}

	svc := NewAuthService(repo, newTestLogger(), []byte("test-secret"))
	_, err := svc.RegisterWithSerial("testuser", "VALID-KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if rolledBackUserID != 5 {
		t.Errorf("expected DeleteUser(5) to be called, got %d", rolledBackUserID)
	}
	if rolledBackAuthUserID != "00000000-0000-0000-0000-000000000099" {
		t.Errorf("expected DeleteSupabaseAuthUser to be called, got %q", rolledBackAuthUserID)
	}
	appErr, ok := err.(*response.AppError)
	if !ok || appErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 AppError, got %+v", err)
	}
}

func TestGetAllKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockAuthRepository{
			getAllSerialKeysFn: func() ([]map[string]interface{}, error) {
				return []map[string]interface{}{{"id": 1, "key_code": "KEY-01"}}, nil
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		keys, err := svc.GetAllKeys()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 {
			t.Errorf("expected 1 key, got %d", len(keys))
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo := &mockAuthRepository{
			getAllSerialKeysFn: func() ([]map[string]interface{}, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		_, err := svc.GetAllKeys()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*response.AppError)
		if !ok || appErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 AppError, got %+v", err)
		}
	})
}

func TestVerifyToken(t *testing.T) {
	secret := []byte("super-secret")
	svc := NewAuthService(&mockAuthRepository{}, newTestLogger(), secret)

	t.Run("invalid malformed token", func(t *testing.T) {
		_, err := svc.VerifyToken("not.a.jwt")
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": "test",
			"exp":      time.Now().Add(-time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(secret)
		_, err := svc.VerifyToken(tokenStr)
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})

	t.Run("wrong signing method (none)", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
			"username": "test",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		_, err := svc.VerifyToken(tokenStr)
		if err == nil {
			t.Fatal("expected error for unexpected signing method")
		}
	})
}
