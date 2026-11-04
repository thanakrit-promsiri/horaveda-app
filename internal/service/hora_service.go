package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"

	"go.uber.org/zap"
)

const (
	DefaultHoraAPIURL  = "http://localhost:8083/api/v1/hora/prompt"
	FixedTimeZone      = 7.0
	FixedHouseSystem   = "W"
	defaultHTTPTimeout = 15 * time.Second
)

type HoraService struct {
	HoraAPIURL string
	HTTPClient *http.Client
	Logger     *zap.SugaredLogger
}

func NewHoraService(horaAPIURL string, log *zap.SugaredLogger, httpClient *http.Client) *HoraService {
	if horaAPIURL == "" {
		horaAPIURL = DefaultHoraAPIURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &HoraService{
		HoraAPIURL: horaAPIURL,
		HTTPClient: httpClient,
		Logger:     log,
	}
}

// GetHoraPrompt ส่งข้อมูลวันเดือนปีเกิด เวลาเกิด และพิกัดไปยังบริการคำนวณดวงชะตา (Lahiri Prompt API)
func (s *HoraService) GetHoraPrompt(ctx context.Context, req domain.HoraPredictRequest) (*domain.HoraPromptResponse, error) {
	promptReq := domain.HoraPromptRequest{
		CalculateReq: domain.HoraCalculateReq{
			Year:        req.Year,
			Month:       req.Month,
			Day:         req.Day,
			HourThai:    req.HourThai,
			TimeZone:    FixedTimeZone,
			Latitude:    req.Latitude,
			Longitude:   req.Longitude,
			HouseSystem: FixedHouseSystem,
		},
	}

	bodyBytes, err := json.Marshal(promptReq)
	if err != nil {
		s.Logger.Errorw("Failed to marshal HoraPromptRequest", "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถเตรียมข้อมูลสำหรับคำนวณดวงชะตาได้", err)
	}

	targetURL := s.HoraAPIURL
	if targetURL == "" {
		targetURL = DefaultHoraAPIURL
	}

	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	s.Logger.Infow("Sending request to Hora Prompt API", "url", targetURL, "year", req.Year, "month", req.Month, "day", req.Day, "hour_thai", req.HourThai)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		s.Logger.Errorw("Failed to create HTTP request for Hora Prompt API", "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถสร้างคำขอไปยังระบบคำนวณดวงชะตาได้", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		s.Logger.Errorw("Failed to call Hora Prompt API", "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถเชื่อมต่อกับระบบคำนวณดวงชะตาได้", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Errorw("Failed to read response from Hora Prompt API", "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถอ่านผลลัพธ์จากระบบคำนวณดวงชะตาได้", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		s.Logger.Errorw("Hora Prompt API returned non-2xx status", "status_code", resp.StatusCode, "body", string(respBytes))
		return nil, response.NewInternalServerError("ระบบคำนวณดวงชะตาแจ้งข้อผิดพลาดในการประมวลผล", fmt.Errorf("hora api status %d: %s", resp.StatusCode, string(respBytes)))
	}

	var promptResp domain.HoraPromptResponse
	if err := json.Unmarshal(respBytes, &promptResp); err != nil {
		s.Logger.Errorw("Failed to unmarshal HoraPromptResponse", "error", err, "body", string(respBytes))
		return nil, response.NewInternalServerError("รูปแบบข้อมูลจากระบบคำนวณดวงชะตาไม่ถูกต้อง", err)
	}

	s.Logger.Infow("Successfully received prompt from Hora Prompt API", "status", promptResp.Status)
	return &promptResp, nil
}
