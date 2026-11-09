package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		if svc.HoraTransitAPIURL != DefaultHoraTransitAPIURL {
			t.Errorf("expected default Transit URL %q, got %q", DefaultHoraTransitAPIURL, svc.HoraTransitAPIURL)
		}
		if svc.HTTPClient == nil {
			t.Error("expected non-nil default HTTPClient")
		}
	})
}

func TestHoraService_GetHoraTransitPrompt(t *testing.T) {
	transitPromptSample := "[Vedic Transit(Lahiri)/WholeSign | Date:2026-12-23]\nTransit: Ju:Ge 05°"

	t.Run("success sends transit_year and transit_month and returns prompt", func(t *testing.T) {
		var capturedReq domain.HoraTransitPromptRequest

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
				Prompt: transitPromptSample,
			})
		})

		svc := NewHoraServiceWithTransit(
			"http://localhost:8083/api/v1/hora/prompt",
			"http://localhost:8083/api/v1/hora/transitdasha_prompt",
			newTestLogger(),
			mockClient,
		)

		res, err := svc.GetHoraTransitPrompt(context.Background(), domain.HoraTransitPredictRequest{
			Year:         1988,
			Month:        12,
			Day:          23,
			HourThai:     10.35,
			Latitude:     13.7563,
			Longitude:    100.5018,
			TransitYear:  2026,
			TransitMonth: 12,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if capturedReq.CalculateReq.TransitYear != 2026 || capturedReq.CalculateReq.TransitMonth != 12 {
			t.Errorf("unexpected transit date: year=%d, month=%d", capturedReq.CalculateReq.TransitYear, capturedReq.CalculateReq.TransitMonth)
		}
		if capturedReq.CalculateReq.TimeZone != 7.0 || capturedReq.CalculateReq.HouseSystem != "W" {
			t.Errorf("expected default timezone 7.0 and house_system W, got tz=%f, house=%s", capturedReq.CalculateReq.TimeZone, capturedReq.CalculateReq.HouseSystem)
		}
		if res.Prompt != transitPromptSample {
			t.Errorf("unexpected prompt: %s", res.Prompt)
		}
	})
}

func TestHoraService_GetBothPrompts(t *testing.T) {
	natalPrompt := "[Natal Prompt]"
	transitPrompt := "[Transit Prompt]"

	mockClient := newMockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "transitdasha_prompt") {
			_ = json.NewEncoder(w).Encode(domain.HoraPromptResponse{
				Status: "success",
				Prompt: transitPrompt,
			})
		} else {
			_ = json.NewEncoder(w).Encode(domain.HoraPromptResponse{
				Status: "success",
				Prompt: natalPrompt,
			})
		}
	})

	svc := NewHoraServiceWithTransit(
		"http://localhost:8083/api/v1/hora/prompt",
		"http://localhost:8083/api/v1/hora/transitdasha_prompt",
		newTestLogger(),
		mockClient,
	)

	natalRes, transitRes, err := svc.GetBothPrompts(context.Background(), domain.HoraTransitPredictRequest{
		Year:         1988,
		Month:        12,
		Day:          23,
		HourThai:     10.35,
		Latitude:     13.7563,
		Longitude:    100.5018,
		TransitYear:  2026,
		TransitMonth: 12,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if natalRes.Prompt != natalPrompt {
		t.Errorf("expected natal prompt %q, got %q", natalPrompt, natalRes.Prompt)
	}
	if transitRes.Prompt != transitPrompt {
		t.Errorf("expected transit prompt %q, got %q", transitPrompt, transitRes.Prompt)
	}
}
