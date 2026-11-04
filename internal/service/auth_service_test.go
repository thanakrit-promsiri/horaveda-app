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
	findSerialKeyFn                 func(serialKey string) (*domain.SerialKey, error)
	findUserByUsernameFn            func(username string) (*domain.User, error)
	createUserFn                    func(username string) (*domain.User, error)
	deleteUserFn                    func(userID int) error
	createSupabaseAuthUserFn        func(email, password, username string) (string, error)
	deleteSupabaseAuthUserFn        func(authUserID string) error
	deleteSupabaseAuthUserByEmailFn func(email string) error
	updateSerialKeyStatusFn         func(keyID int, userID int) error
	rollbackSerialKeyStatusFn       func(keyID int) error
	revokeSerialKeysByUserIDFn      func(userID int) ([]domain.SerialKey, error)
	getAllSerialKeysFn              func() ([]map[string]interface{}, error)
}

func (m *mockAuthRepository) FindSerialKey(serialKey string) (*domain.SerialKey, error) {
	return m.findSerialKeyFn(serialKey)
}

func (m *mockAuthRepository) FindUserByUsername(username string) (*domain.User, error) {
	if m.findUserByUsernameFn != nil {
		return m.findUserByUsernameFn(username)
	}
	return nil, nil
}

func (m *mockAuthRepository) CreateUser(username string) (*domain.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(username)
	}
	return nil, nil
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

func (m *mockAuthRepository) DeleteSupabaseAuthUserByEmail(email string) error {
	if m.deleteSupabaseAuthUserByEmailFn != nil {
		return m.deleteSupabaseAuthUserByEmailFn(email)
	}
	return nil
}

func (m *mockAuthRepository) UpdateSerialKeyStatus(keyID int, userID int) error {
	if m.updateSerialKeyStatusFn != nil {
		return m.updateSerialKeyStatusFn(keyID, userID)
	}
	return nil
}

func (m *mockAuthRepository) RollbackSerialKeyStatus(keyID int) error {
	if m.rollbackSerialKeyStatusFn != nil {
		return m.rollbackSerialKeyStatusFn(keyID)
	}
	return nil
}

func (m *mockAuthRepository) RevokeSerialKeysByUserID(userID int) ([]domain.SerialKey, error) {
	if m.revokeSerialKeysByUserIDFn != nil {
		return m.revokeSerialKeysByUserIDFn(userID)
	}
	return nil, nil
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
	token, err := svc.RegisterWithSerial("testuser01", "BOOM-VIP-001")
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

func TestRevokeUserAndSerial(t *testing.T) {
	t.Run("success revoking BOOM-VIP-001 and deleting testuser01", func(t *testing.T) {
		var deletedUserID int
		var deletedEmail string
		userID38 := 38

		repo := &mockAuthRepository{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return &domain.User{ID: 38, Username: "testuser01"}, nil
			},
			revokeSerialKeysByUserIDFn: func(userID int) ([]domain.SerialKey, error) {
				return []domain.SerialKey{
					{ID: 1, KeyCode: "BOOM-VIP-001", IsUsed: true, UserID: &userID38},
				}, nil
			},
			deleteUserFn: func(userID int) error {
				deletedUserID = userID
				return nil
			},
			deleteSupabaseAuthUserByEmailFn: func(email string) error {
				deletedEmail = email
				return nil
			},
		}

		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		res, err := svc.RevokeUserAndSerial("testuser01")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if deletedUserID != 38 {
			t.Errorf("expected deletedUserID 38, got %d", deletedUserID)
		}
		if deletedEmail != "testuser01@garbagecollection.info" {
			t.Errorf("expected deletedEmail testuser01@garbagecollection.info, got %q", deletedEmail)
		}
		if len(res.RevokedKeys) != 1 || res.RevokedKeys[0] != "BOOM-VIP-001" {
			t.Errorf("unexpected RevokedKeys: %+v", res.RevokedKeys)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		repo := &mockAuthRepository{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return nil, errors.New("not found")
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		_, err := svc.RevokeUserAndSerial("unknown")
		if err == nil {
			t.Fatal("expected error when user not found")
		}
	})

	t.Run("revoke serial keys failure", func(t *testing.T) {
		repo := &mockAuthRepository{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return &domain.User{ID: 38, Username: "testuser01"}, nil
			},
			revokeSerialKeysByUserIDFn: func(userID int) ([]domain.SerialKey, error) {
				return nil, errors.New("revoke failed")
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		_, err := svc.RevokeUserAndSerial("testuser01")
		if err == nil {
			t.Fatal("expected error when revoke serial keys fails")
		}
	})

	t.Run("delete user failure triggers serial key restoration", func(t *testing.T) {
		var restoredKeyID int
		repo := &mockAuthRepository{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return &domain.User{ID: 38, Username: "testuser01"}, nil
			},
			revokeSerialKeysByUserIDFn: func(userID int) ([]domain.SerialKey, error) {
				return []domain.SerialKey{{ID: 1, KeyCode: "BOOM-VIP-001"}}, nil
			},
			deleteUserFn: func(userID int) error {
				return errors.New("delete user failed")
			},
			updateSerialKeyStatusFn: func(keyID int, userID int) error {
				restoredKeyID = keyID
				return nil
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		_, err := svc.RevokeUserAndSerial("testuser01")
		if err == nil {
			t.Fatal("expected error when delete user fails")
		}
		if restoredKeyID != 1 {
			t.Errorf("expected key 1 to be restored, got %d", restoredKeyID)
		}
	})

	t.Run("delete supabase auth user failure triggers rollback", func(t *testing.T) {
		var recreatedUsername string
		repo := &mockAuthRepository{
			findUserByUsernameFn: func(username string) (*domain.User, error) {
				return &domain.User{ID: 38, Username: "testuser01"}, nil
			},
			revokeSerialKeysByUserIDFn: func(userID int) ([]domain.SerialKey, error) {
				return []domain.SerialKey{{ID: 1, KeyCode: "BOOM-VIP-001"}}, nil
			},
			deleteUserFn: func(userID int) error {
				return nil
			},
			deleteSupabaseAuthUserByEmailFn: func(email string) error {
				return errors.New("supabase delete failed")
			},
			createUserFn: func(username string) (*domain.User, error) {
				recreatedUsername = username
				return &domain.User{ID: 39, Username: username}, nil
			},
		}
		svc := NewAuthService(repo, newTestLogger(), []byte("secret"))
		_, err := svc.RevokeUserAndSerial("testuser01")
		if err == nil {
			t.Fatal("expected error when supabase auth deletion fails")
		}
		if recreatedUsername != "testuser01" {
			t.Errorf("expected user testuser01 to be recreated on rollback, got %q", recreatedUsername)
		}
	})
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
