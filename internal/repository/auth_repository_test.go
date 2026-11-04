package repository

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockSupabaseRepo(t *testing.T, handler http.HandlerFunc) (AuthRepository, func()) {
	t.Helper()
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rr := httptest.NewRecorder()
		handler(rr, req)
		return rr.Result(), nil
	})

	client, err := supabase.NewClient("https://mock.supabase.co", "test-api-key", nil)
	if err != nil {
		http.DefaultTransport = origTransport
		t.Fatalf("failed to create supabase client: %v", err)
	}
	repo := NewAuthRepository(client, zap.NewNop().Sugar())
	cleanup := func() {
		http.DefaultTransport = origTransport
	}
	return repo, cleanup
}

func TestFindSerialKey(t *testing.T) {
	t.Run("found serial key", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 1, "key_code": "BOOM-VIP-001", "is_used": false, "user_id": null}]`)
		})
		defer cleanup()

		key, err := repo.FindSerialKey("BOOM-VIP-001")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key.ID != 1 || key.KeyCode != "BOOM-VIP-001" || key.IsUsed {
			t.Errorf("unexpected key data: %+v", key)
		}
	})

	t.Run("serial key not found (empty result)", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		defer cleanup()

		_, err := repo.FindSerialKey("UNKNOWN")
		if err == nil {
			t.Fatal("expected error when key not found")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `invalid-json`)
		})
		defer cleanup()

		_, err := repo.FindSerialKey("BOOM-VIP-001")
		if err == nil {
			t.Fatal("expected error on invalid JSON")
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "db error"}`)
		})
		defer cleanup()

		_, err := repo.FindSerialKey("BOOM-VIP-001")
		if err == nil {
			t.Fatal("expected error on HTTP 500")
		}
	})
}

func TestFindUserByUsername(t *testing.T) {
	t.Run("found user", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 38, "username": "testuser01"}]`)
		})
		defer cleanup()

		user, err := repo.FindUserByUsername("testuser01")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.ID != 38 || user.Username != "testuser01" {
			t.Errorf("unexpected user: %+v", user)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		defer cleanup()

		_, err := repo.FindUserByUsername("missing")
		if err == nil {
			t.Fatal("expected error when user not found")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `invalid`)
		})
		defer cleanup()

		_, err := repo.FindUserByUsername("testuser01")
		if err == nil {
			t.Fatal("expected error on invalid JSON")
		}
	})

	t.Run("db error", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer cleanup()

		_, err := repo.FindUserByUsername("testuser01")
		if err == nil {
			t.Fatal("expected error on HTTP 500")
		}
	})
}

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 7, "username": "testuser01"}]`)
		})
		defer cleanup()

		user, err := repo.CreateUser("testuser01")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.ID != 7 || user.Username != "testuser01" {
			t.Errorf("unexpected user: %+v", user)
		}
	})

	t.Run("empty created user response", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		defer cleanup()

		_, err := repo.CreateUser("testuser01")
		if err == nil {
			t.Fatal("expected error on empty user response")
		}
	})

	t.Run("duplicate or db error", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"message": "duplicate key"}`)
		})
		defer cleanup()

		_, err := repo.CreateUser("testuser01")
		if err == nil {
			t.Fatal("expected error on duplicate username")
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		defer cleanup()

		if err := repo.DeleteUser(7); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "delete failed"}`)
		})
		defer cleanup()

		if err := repo.DeleteUser(7); err == nil {
			t.Fatal("expected error on delete failure")
		}
	})
}

func TestCreateSupabaseAuthUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/admin/users") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"id": "00000000-0000-0000-0000-000000000001", "email": "testuser01@garbagecollection.info"}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})
		defer cleanup()

		authID, err := repo.CreateSupabaseAuthUser("testuser01@garbagecollection.info", "BOOM-VIP-001", "testuser01")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if authID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("expected authID 00000000-0000-0000-0000-000000000001, got %q", authID)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"msg": "User already registered"}`)
		})
		defer cleanup()

		_, err := repo.CreateSupabaseAuthUser("testuser01@garbagecollection.info", "BOOM-VIP-001", "testuser01")
		if err == nil {
			t.Fatal("expected error when auth user creation fails")
		}
	})
}

func TestDeleteSupabaseAuthUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUser("00000000-0000-0000-0000-000000000001"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUser("invalid-uuid"); err == nil {
			t.Fatal("expected error on invalid UUID")
		}
	})

	t.Run("API failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"msg": "delete failed"}`)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUser("00000000-0000-0000-0000-000000000001"); err == nil {
			t.Fatal("expected error on API failure")
		}
	})
}

func TestDeleteSupabaseAuthUserByEmail(t *testing.T) {
	t.Run("found and deleted successfully", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet {
				fmt.Fprint(w, `{"users": [{"id": "00000000-0000-0000-0000-000000000038", "email": "testuser01@garbagecollection.info"}]}`)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUserByEmail("testuser01@garbagecollection.info"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("user not in auth list", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"users": []}`)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUserByEmail("missing@garbagecollection.info"); err != nil {
			t.Fatalf("expected nil error when auth user already missing, got %v", err)
		}
	})

	t.Run("list users failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUserByEmail("testuser01@garbagecollection.info"); err == nil {
			t.Fatal("expected error on list failure")
		}
	})

	t.Run("delete user failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"users": [{"id": "00000000-0000-0000-0000-000000000038", "email": "testuser01@garbagecollection.info"}]}`)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer cleanup()

		if err := repo.DeleteSupabaseAuthUserByEmail("testuser01@garbagecollection.info"); err == nil {
			t.Fatal("expected error when AdminDeleteUser fails")
		}
	})
}

func TestUpdateSerialKeyStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 1, "is_used": true, "user_id": 7}]`)
		})
		defer cleanup()

		if err := repo.UpdateSerialKeyStatus(1, 7); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "update failed"}`)
		})
		defer cleanup()

		if err := repo.UpdateSerialKeyStatus(1, 7); err == nil {
			t.Fatal("expected error on update failure")
		}
	})
}

func TestRollbackSerialKeyStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 1, "is_used": false, "user_id": null}]`)
		})
		defer cleanup()

		if err := repo.RollbackSerialKeyStatus(1); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "rollback failed"}`)
		})
		defer cleanup()

		if err := repo.RollbackSerialKeyStatus(1); err == nil {
			t.Fatal("expected error on rollback failure")
		}
	})
}

func TestRevokeSerialKeysByUserID(t *testing.T) {
	t.Run("found and revoked keys", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet {
				fmt.Fprint(w, `[{"id": 1, "key_code": "BOOM-VIP-001", "is_used": true, "user_id": 38}]`)
				return
			}
			fmt.Fprint(w, `[{"id": 1, "key_code": "BOOM-VIP-001", "is_used": false, "user_id": null}]`)
		})
		defer cleanup()

		keys, err := repo.RevokeSerialKeysByUserID(38)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(keys) != 1 || keys[0].KeyCode != "BOOM-VIP-001" {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("no keys bound to user", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		defer cleanup()

		keys, err := repo.RevokeSerialKeysByUserID(38)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("expected 0 keys, got %d", len(keys))
		}
	})

	t.Run("select error", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer cleanup()

		_, err := repo.RevokeSerialKeysByUserID(38)
		if err == nil {
			t.Fatal("expected error on select failure")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `invalid`)
		})
		defer cleanup()

		_, err := repo.RevokeSerialKeysByUserID(38)
		if err == nil {
			t.Fatal("expected error on invalid JSON")
		}
	})

	t.Run("update error", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `[{"id": 1, "key_code": "BOOM-VIP-001", "is_used": true, "user_id": 38}]`)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer cleanup()

		_, err := repo.RevokeSerialKeysByUserID(38)
		if err == nil {
			t.Fatal("expected error on update failure")
		}
	})
}

func TestGetAllSerialKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"id": 1, "key_code": "KEY-001"}, {"id": 2, "key_code": "KEY-002"}]`)
		})
		defer cleanup()

		keys, err := repo.GetAllSerialKeys()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `not-json`)
		})
		defer cleanup()

		_, err := repo.GetAllSerialKeys()
		if err == nil {
			t.Fatal("expected error on invalid JSON")
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo, cleanup := newMockSupabaseRepo(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "query failed"}`)
		})
		defer cleanup()

		_, err := repo.GetAllSerialKeys()
		if err == nil {
			t.Fatal("expected error on failure")
		}
	})
}
