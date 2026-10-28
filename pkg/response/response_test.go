package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		underlying := errors.New("database connection failed")
		appErr := NewInternalServerError("เกิดข้อผิดพลาด", underlying)

		if appErr.Error() != "database connection failed" {
			t.Errorf("expected %q, got %q", "database connection failed", appErr.Error())
		}
		if appErr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, appErr.Code)
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		appErr := NewBadRequestError("ข้อมูลไม่ถูกต้อง", nil)

		if appErr.Error() != "ข้อมูลไม่ถูกต้อง" {
			t.Errorf("expected %q, got %q", "ข้อมูลไม่ถูกต้อง", appErr.Error())
		}
		if appErr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, appErr.Code)
		}
	})
}

func TestErrorConstructors(t *testing.T) {
	err := errors.New("root error")

	badReq := NewBadRequestError("bad request", err)
	if badReq.Code != http.StatusBadRequest || badReq.Message != "bad request" || badReq.Err != err {
		t.Errorf("NewBadRequestError mismatch: %+v", badReq)
	}

	unauth := NewUnauthorizedError("unauthorized", err)
	if unauth.Code != http.StatusUnauthorized || unauth.Message != "unauthorized" || unauth.Err != err {
		t.Errorf("NewUnauthorizedError mismatch: %+v", unauth)
	}

	notFound := NewNotFoundError("not found", err)
	if notFound.Code != http.StatusNotFound || notFound.Message != "not found" || notFound.Err != err {
		t.Errorf("NewNotFoundError mismatch: %+v", notFound)
	}

	internal := NewInternalServerError("internal error", err)
	if internal.Code != http.StatusInternalServerError || internal.Message != "internal error" || internal.Err != err {
		t.Errorf("NewInternalServerError mismatch: %+v", internal)
	}
}

func TestSuccessResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"username": "testuser"}

	Success(rr, http.StatusOK, "สำเร็จ", data)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected Success=true, got false")
	}
	if resp.Message != "สำเร็จ" {
		t.Errorf("expected Message=%q, got %q", "สำเร็จ", resp.Message)
	}
}

func TestErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()

	Error(rr, http.StatusUnauthorized, "ไม่ได้รับอนุญาต")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Success {
		t.Errorf("expected Success=false, got true")
	}
	if resp.Message != "ไม่ได้รับอนุญาต" {
		t.Errorf("expected Message=%q, got %q", "ไม่ได้รับอนุญาต", resp.Message)
	}
}
