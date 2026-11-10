package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/internal/domain"
	"horaveda-app/internal/service"
	"horaveda-app/pkg/response"
)

// FixedBE2570TransitYear ปี ค.ศ. สำหรับปี พ.ศ. 2570
const FixedBE2570TransitYear = 2027

// HandleHoraTransitPredict รับข้อมูลวันเดือนปีเกิด เวลาเกิด พิกัด และเดือนปีจร เพื่อคำนวณและพยากรณ์ดวงชะตาดาวจรประจำเดือน
func (h *AuthHandler) HandleHoraTransitPredict(w http.ResponseWriter, r *http.Request) {
	h.processTransitPredict(w, r, 0, 0)
}

// HandleHoraTransitPredictMonth สร้าง HTTP Handler สำหรับการพยากรณ์ดาวจรเฉพาะเดือนของปี พ.ศ. 2570 (ค.ศ. 2027)
// โดยล็อก house_system="W", transit_year=2027 และ transit_month ตามเดือนที่กำหนด (1-12)
func (h *AuthHandler) HandleHoraTransitPredictMonth(month int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.processTransitPredict(w, r, FixedBE2570TransitYear, month)
	}
}

func (h *AuthHandler) processTransitPredict(w http.ResponseWriter, r *http.Request, fixedYear, fixedMonth int) {
	if r.Method != http.MethodPost {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var claimsUsername string
	if claims, ok := middleware.GetUserClaims(r.Context()); ok {
		if u, exists := claims["username"].(string); exists {
			claimsUsername = u
		}
	}
	if claimsUsername == "" {
		h.Logger.Warnw("Missing or unauthenticated user in context for hora transit predict")
		response.Error(w, http.StatusUnauthorized, "ไม่พบข้อมูลยืนยันตัวตน กรุณาเข้าสู่ระบบ")
		return
	}
	username := claimsUsername

	var req domain.HoraTransitPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Warnw("Invalid hora transit predict request payload", "error", err, "username", username)
		response.Error(w, http.StatusBadRequest, "กรุณาระบุข้อมูลวันเดือนปีเกิด เวลาเกิด และพิกัดสถานที่เกิดให้ถูกต้องครบถ้วน")
		return
	}

	// หากมีการกำหนดปีและเดือนจรแบบคงที่ (เช่น API 12 เส้นครอบ ปี 2570) ให้แทนที่ค่าทันที
	if fixedYear > 0 {
		req.TransitYear = fixedYear
	}
	if fixedMonth > 0 {
		req.TransitMonth = fixedMonth
	}
	req.HouseSystem = service.FixedHouseSystem

	if req.TimeZone == 0 {
		req.TimeZone = service.FixedTimeZone
	}

	if !isValidHoraTransitPredictRequest(req) {
		h.Logger.Warnw("Invalid hora transit predict request values", "username", username, "payload", req)
		response.Error(w, http.StatusBadRequest, "กรุณาระบุข้อมูลวันเดือนปีเกิด เวลาเกิด พิกัดสถานที่เกิด และเดือนปีจรให้ถูกต้องครบถ้วน")
		return
	}

	// ตรวจสอบ username ที่ส่งมาใน JSON (ถ้ามี)
	req.Username = strings.TrimSpace(req.Username)
	if req.Username != "" {
		if !usernameRegex.MatchString(req.Username) {
			h.Logger.Warnw("Invalid username in hora transit predict request", "username", req.Username)
			response.Error(w, http.StatusBadRequest, "username ต้องเป็นอักษรโรมันและตัวเลขเท่านั้น")
			return
		}
		// ป้องกันการเข้าถึง Workspace ของผู้อื่น: username ต้องตรงกับบัญชีที่เข้าสู่ระบบ
		if !strings.EqualFold(req.Username, claimsUsername) {
			h.Logger.Warnw("Hora transit predict username does not match authenticated user", "request_username", req.Username, "token_username", claimsUsername)
			response.Error(w, http.StatusForbidden, "username ไม่ตรงกับบัญชีที่เข้าสู่ระบบ")
			return
		}
	}
	req.Username = username

	h.Logger.Infow("Processing horoscope monthly transit prediction",
		"username", username,
		"year", req.Year,
		"month", req.Month,
		"day", req.Day,
		"transit_year", req.TransitYear,
		"transit_month", req.TransitMonth,
	)

	horaSvc := h.HoraService
	if horaSvc == nil {
		horaAPIURL := ""
		horaTransitAPIURL := ""
		if h.Cfg != nil {
			horaAPIURL = h.Cfg.HoraAPIURL
			horaTransitAPIURL = h.Cfg.HoraTransitAPIURL
		}
		horaSvc = service.NewHoraServiceWithTransit(horaAPIURL, horaTransitAPIURL, h.Logger, nil)
	}

	// ดึงข้อมูลดาวชะตากำเนิดและดาวจรพร้อมกันแบบ Concurrently
	natalResult, transitResult, err := horaSvc.GetBothPrompts(r.Context(), req)
	if err != nil {
		h.writeHoraError(w, "Hora natal/transit calculation", username, err)
		return
	}

	h.Logger.Infow("Hora prompts calculated successfully",
		"username", username,
		"natal_status", natalResult.Status,
		"transit_status", transitResult.Status,
	)

	transitMonthThai := service.FormatThaiMonthYear(req.TransitYear, req.TransitMonth)
	assembledPrompt := service.BuildVedicTransitPredictionPrompt(natalResult.Prompt, transitResult.Prompt, req.TransitYear, req.TransitMonth)

	resp := domain.HoraTransitPredictResponse{
		Status:           transitResult.Status,
		NatalPrompt:      natalResult.Prompt,
		TransitPrompt:    transitResult.Prompt,
		TransitMonth:     req.TransitMonth,
		TransitYear:      req.TransitYear,
		TransitMonthThai: transitMonthThai,
	}

	// ตรวจสอบการเชื่อมต่อ AnythingLLM
	if h.AnythingLLMService == nil || !h.AnythingLLMService.IsEnabled() {
		h.Logger.Warnw("AnythingLLM is not configured, returning transit prompt only", "username", username)
		response.Success(w, http.StatusOK, "คำนวณข้อมูลดาวชะตากำเนิดและดาวจรสำเร็จ", resp)
		return
	}

	// ส่งพิกัดดาวจรเข้า AnythingLLM ใน Thread ({workspace}-transit)
	prediction, err := h.AnythingLLMService.PredictTransitChart(r.Context(), username, assembledPrompt)
	if err != nil {
		h.logLLMFallback("transit", username, err)
		resp.Warning = "ไม่สามารถเชื่อมต่อบริการทำนาย AI ได้ในขณะนี้ ระบบได้ส่งคืนข้อมูลดาวชะตากำเนิดและดาวจรสำหรับการผูกดวงแล้ว"
		response.Success(w, http.StatusOK, "คำนวณข้อมูลดาวจรสำเร็จ (ระบบทำนาย AI ไม่พร้อมใช้งานชั่วคราว)", resp)
		return
	}

	resp.Prediction = prediction.TextResponse
	resp.WorkspaceSlug = prediction.WorkspaceSlug
	resp.ThreadSlug = prediction.ThreadSlug

	h.Logger.Infow("Hora monthly transit prediction completed successfully",
		"username", username,
		"workspace_slug", resp.WorkspaceSlug,
		"thread_slug", resp.ThreadSlug,
	)
	response.Success(w, http.StatusOK, "พยากรณ์ดวงชะตาดาวจรประจำเดือนสำเร็จ", resp)
}

func isValidHoraTransitPredictRequest(req domain.HoraTransitPredictRequest) bool {
	if req.Year <= 0 {
		return false
	}
	if req.Month < 1 || req.Month > 12 {
		return false
	}
	if req.Day < 1 || req.Day > 31 {
		return false
	}
	if req.HourThai < 0 || req.HourThai >= 24 {
		return false
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return false
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return false
	}
	if req.TransitYear <= 0 {
		return false
	}
	if req.TransitMonth < 1 || req.TransitMonth > 12 {
		return false
	}
	return true
}
