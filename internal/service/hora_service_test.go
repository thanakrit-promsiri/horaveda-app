package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockHTTPClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rr := httptest.NewRecorder()
			handler(rr, req)
			return rr.Result(), nil
		}),
	}
}

func TestHoraService_GetHoraPrompt(t *testing.T) {
	samplePrompt := "[Vedic Sidereal(Lahiri)/WholeSign]\nASC:Aq 03°39'23.46\"|D9:Sc|Nak:Dhanishta(4,Ma)"

	t.Run("success sends fixed time_zone=7.0 and house_system=W and returns prompt", func(t *testing.T) {
		var capturedReq domain.HoraPromptRequest

		mockClient := newMockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&capturedReq); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(domain.HoraPromptResponse{
				Status: "success",
				Prompt: samplePrompt,
			})
		})

		svc := NewHoraService("http://localhost:8083/api/v1/hora/prompt", newTestLogger(), mockClient)
		res, err := svc.GetHoraPrompt(context.Background(), domain.HoraPredictRequest{
			Year:      1988,
			Month:     12,
			Day:       23,
			HourThai:  10.35,
			Latitude:  13.7563,
			Longitude: 100.5018,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if capturedReq.CalculateReq.Year != 1988 ||
			capturedReq.CalculateReq.Month != 12 ||
			capturedReq.CalculateReq.Day != 23 ||
			capturedReq.CalculateReq.HourThai != 10.35 ||
			capturedReq.CalculateReq.TimeZone != 7.0 ||
			capturedReq.CalculateReq.Latitude != 13.7563 ||
			capturedReq.CalculateReq.Longitude != 100.5018 ||
			capturedReq.CalculateReq.HouseSystem != "W" {
			t.Errorf("unexpected calculateReq sent: %+v", capturedReq.CalculateReq)
		}

		if res.Status != "success" || res.Prompt != samplePrompt {
			t.Errorf("unexpected response: %+v", res)
		}
	})

	t.Run("non-2xx status from downstream returns 500 AppError", func(t *testing.T) {
		mockClient := newMockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
		})

		svc := NewHoraService("http://localhost:8083/api/v1/hora/prompt", newTestLogger(), mockClient)
		_, err := svc.GetHoraPrompt(context.Background(), domain.HoraPredictRequest{
			Year: 1988, Month: 12, Day: 23, HourThai: 10.35, Latitude: 13.7563, Longitude: 100.5018,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*response.AppError)
		if !ok || appErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 AppError, got %+v", err)
		}
	})

	t.Run("invalid JSON from downstream returns 500 AppError", func(t *testing.T) {
		mockClient := newMockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid-json`))
		})

		svc := NewHoraService("http://localhost:8083/api/v1/hora/prompt", newTestLogger(), mockClient)
		_, err := svc.GetHoraPrompt(context.Background(), domain.HoraPredictRequest{
			Year: 1988, Month: 12, Day: 23, HourThai: 10.35, Latitude: 13.7563, Longitude: 100.5018,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("network connection error returns 500 AppError", func(t *testing.T) {
		errClient := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			}),
		}
		svc := NewHoraService("http://localhost:8083/api/v1/hora/prompt", newTestLogger(), errClient)
		_, err := svc.GetHoraPrompt(context.Background(), domain.HoraPredictRequest{
			Year: 1988, Month: 12, Day: 23, HourThai: 10.35, Latitude: 13.7563, Longitude: 100.5018,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("defaults when URL and HTTPClient are empty", func(t *testing.T) {
		svc := NewHoraService("", newTestLogger(), nil)
		if svc.HoraAPIURL != DefaultHoraAPIURL {
			t.Errorf("expected default URL %q, got %q", DefaultHoraAPIURL, svc.HoraAPIURL)
		}
		if svc.HTTPClient == nil {
			t.Error("expected non-nil default HTTPClient")
		}
	})
}
