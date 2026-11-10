package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"

	"go.uber.org/zap"
)

const (
	DefaultHoraAPIURL        = "http://localhost:8083/api/v1/hora/prompt"
	DefaultHoraTransitAPIURL = "http://localhost:8083/api/v1/hora/transitdasha_prompt"
	FixedTimeZone            = 7.0
	FixedHouseSystem         = "W"
	defaultHTTPTimeout       = 15 * time.Second
)

type HoraService struct {
	HoraAPIURL        string
	HoraTransitAPIURL string
	HTTPClient        *http.Client
	Logger            *zap.SugaredLogger
}

func NewHoraService(horaAPIURL string, log *zap.SugaredLogger, httpClient *http.Client) *HoraService {
	return NewHoraServiceWithTransit(horaAPIURL, "", log, httpClient)
}

func NewHoraServiceWithTransit(horaAPIURL, horaTransitAPIURL string, log *zap.SugaredLogger, httpClient *http.Client) *HoraService {
	if horaAPIURL == "" {
		horaAPIURL = DefaultHoraAPIURL
	}
	if horaTransitAPIURL == "" {
		if strings.HasSuffix(horaAPIURL, "/prompt") {
			horaTransitAPIURL = strings.TrimSuffix(horaAPIURL, "/prompt") + "/transitdasha_prompt"
		} else {
			horaTransitAPIURL = DefaultHoraTransitAPIURL
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &HoraService{
		HoraAPIURL:        horaAPIURL,
		HoraTransitAPIURL: horaTransitAPIURL,
		HTTPClient:        httpClient,
		Logger:            log,
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

	s.Logger.Infow("Sending request to Hora Prompt API", "url", targetURL, "year", req.Year, "month", req.Month, "day", req.Day, "hour_thai", req.HourThai)
	return s.callHoraAPI(ctx, targetURL, "ระบบคำนวณดวงชะตา", bodyBytes)
}

// GetHoraTransitPrompt ส่งข้อมูลวันเดือนปีเกิด เวลาเกิด พิกัด และเดือนปีจรไปยัง Lahiri Transit Dasha Prompt API
func (s *HoraService) GetHoraTransitPrompt(ctx context.Context, req domain.HoraTransitPredictRequest) (*domain.HoraPromptResponse, error) {
	timeZone := req.TimeZone
	if timeZone == 0 {
		timeZone = FixedTimeZone
	}
	houseSystem := strings.TrimSpace(req.HouseSystem)
	if houseSystem == "" {
		houseSystem = FixedHouseSystem
	}

	promptReq := domain.HoraTransitPromptRequest{
		CalculateReq: domain.HoraTransitCalculateReq{
			Year:         req.Year,
			Month:        req.Month,
			Day:          req.Day,
			HourThai:     req.HourThai,
			TimeZone:     timeZone,
			Latitude:     req.Latitude,
			Longitude:    req.Longitude,
			HouseSystem:  houseSystem,
			TransitYear:  req.TransitYear,
			TransitMonth: req.TransitMonth,
		},
	}

	bodyBytes, err := json.Marshal(promptReq)
	if err != nil {
		s.Logger.Errorw("Failed to marshal HoraTransitPromptRequest", "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถเตรียมข้อมูลสำหรับคำนวณดาวจรได้", err)
	}

	targetURL := s.HoraTransitAPIURL
	if targetURL == "" {
		targetURL = DefaultHoraTransitAPIURL
	}

	s.Logger.Infow("Sending request to Hora Transit Dasha Prompt API",
		"url", targetURL,
		"year", req.Year,
		"month", req.Month,
		"day", req.Day,
		"transit_year", req.TransitYear,
		"transit_month", req.TransitMonth,
	)
	return s.callHoraAPI(ctx, targetURL, "ระบบคำนวณดาวจร", bodyBytes)
}

// callHoraAPI ทำหน้าที่ยิงคำขอ HTTP ไปยัง Lahiri Prompt API และแปลงผลลัพธ์เป็นมาตรฐาน
func (s *HoraService) callHoraAPI(ctx context.Context, targetURL, serviceName string, bodyBytes []byte) (*domain.HoraPromptResponse, error) {
	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		s.Logger.Errorw("Failed to create HTTP request for "+serviceName, "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถสร้างคำขอไปยัง"+serviceName+"ได้", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		s.Logger.Errorw("Failed to call "+serviceName, "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถเชื่อมต่อกับ"+serviceName+"ได้", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Errorw("Failed to read response from "+serviceName, "url", targetURL, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถอ่านผลลัพธ์จาก"+serviceName+"ได้", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		s.Logger.Errorw(serviceName+" returned non-2xx status", "status_code", resp.StatusCode, "body", string(respBytes))
		return nil, response.NewInternalServerError(serviceName+"แจ้งข้อผิดพลาดในการประมวลผล", fmt.Errorf("%s status %d: %s", serviceName, resp.StatusCode, string(respBytes)))
	}

	var promptResp domain.HoraPromptResponse
	if err := json.Unmarshal(respBytes, &promptResp); err != nil {
		s.Logger.Errorw("Failed to unmarshal response from "+serviceName, "error", err, "body", string(respBytes))
		return nil, response.NewInternalServerError("รูปแบบข้อมูลจาก"+serviceName+"ไม่ถูกต้อง", err)
	}

	s.Logger.Infow("Successfully received prompt from "+serviceName, "status", promptResp.Status)
	return &promptResp, nil
}

// GetBothPrompts ดึงข้อมูลทั้งดาวชะตากำเนิด (Natal Prompt) และดาวจร (Transit Prompt) พร้อมกันแบบ Concurrently
func (s *HoraService) GetBothPrompts(ctx context.Context, req domain.HoraTransitPredictRequest) (*domain.HoraPromptResponse, *domain.HoraPromptResponse, error) {
	var (
		natalResp   *domain.HoraPromptResponse
		transitResp *domain.HoraPromptResponse
		natalErr    error
		transitErr  error
		wg          sync.WaitGroup
	)

	natalReq := domain.HoraPredictRequest{
		Username:  req.Username,
		Year:      req.Year,
		Month:     req.Month,
		Day:       req.Day,
		HourThai:  req.HourThai,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		natalResp, natalErr = s.GetHoraPrompt(ctx, natalReq)
	}()

	go func() {
		defer wg.Done()
		transitResp, transitErr = s.GetHoraTransitPrompt(ctx, req)
	}()

	wg.Wait()

	if natalErr != nil {
		return nil, nil, natalErr
	}
	if transitErr != nil {
		return nil, nil, transitErr
	}

	return natalResp, transitResp, nil
}
