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
			fmt.Fprint(w, `[{"id": 1, "key_code": "MALIN-VIP-001", "is_used": false, "user_id": null}]`)
		})
		defer cleanup()

		key, err := repo.FindSerialKey("MALIN-VIP-001")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key.ID != 1 || key.KeyCode != "MALIN-VIP-001" || key.IsUsed {
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

		_, err := repo.FindSerialKey("MALIN-VIP-001")
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

		_, err := repo.FindSerialKey("MALIN-VIP-001")
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

		authID, err := repo.CreateSupabaseAuthUser("testuser01@garbagecollection.info", "MALIN-VIP-001", "testuser01")
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

		_, err := repo.CreateSupabaseAuthUser("testuser01@garbagecollection.info", "MALIN-VIP-001", "testuser01")
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
