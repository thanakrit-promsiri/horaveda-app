package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/internal/domain"
	"horaveda-app/internal/service"
	"horaveda-app/pkg/response"
)

// HandleHoraPredict รับข้อมูลวันเดือนปีเกิด เวลาเกิด และพิกัดสถานที่เกิด เพื่อคำนวณและทำนายพื้นดวงชะตา
func (h *AuthHandler) HandleHoraPredict(w http.ResponseWriter, r *http.Request) {
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
	username := claimsUsername

	var req domain.HoraPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !isValidHoraPredictRequest(req) {
		h.Logger.Warnw("Invalid hora predict request payload", "error", err, "username", username, "payload", req)
		response.Error(w, http.StatusBadRequest, "กรุณาระบุข้อมูลวันเดือนปีเกิด เวลาเกิด และพิกัดละติจูด/ลองจิจูดให้ถูกต้องครบถ้วน")
		return
	}

	// ตรวจสอบ username ที่ใช้อ้างอิง Workspace บน AnythingLLM
	req.Username = strings.TrimSpace(req.Username)
	if req.Username != "" {
		if !usernameRegex.MatchString(req.Username) {
			h.Logger.Warnw("Invalid username in hora predict request", "username", req.Username)
			response.Error(w, http.StatusBadRequest, "username ต้องเป็นอักษรโรมันและตัวเลขเท่านั้น")
			return
		}
		// ป้องกันการเข้าถึง Workspace ของผู้อื่น: username ต้องตรงกับบัญชีที่เข้าสู่ระบบ
		if claimsUsername != "" && !strings.EqualFold(req.Username, claimsUsername) {
			h.Logger.Warnw("Hora predict username does not match authenticated user", "request_username", req.Username, "token_username", claimsUsername)
			response.Error(w, http.StatusForbidden, "username ไม่ตรงกับบัญชีที่เข้าสู่ระบบ")
			return
		}
		username = req.Username
	}

	h.Logger.Infow("Processing horoscope natal chart prediction", "username", username, "year", req.Year, "month", req.Month, "day", req.Day, "hour_thai", req.HourThai)

	horaSvc := h.HoraService
	if horaSvc == nil {
		horaAPIURL := ""
		if h.Cfg != nil {
			horaAPIURL = h.Cfg.HoraAPIURL
		}
		horaSvc = service.NewHoraService(horaAPIURL, h.Logger, nil)
	}

	result, err := horaSvc.GetHoraPrompt(r.Context(), req)
	if err != nil {
		h.writeHoraError(w, "Hora prompt calculation", username, err)
		return
	}

	h.Logger.Infow("Hora prompt calculated successfully", "username", username, "status", result.Status)

	resp := domain.HoraPredictResponse{
		Status: result.Status,
		Prompt: result.Prompt,
	}

	// ส่งพิกัดดวงดาวเข้า AnythingLLM (1 User : 1 Workspace : 1 Thread) เพื่อรับคำทำนายพื้นดวง
	if !h.AnythingLLMService.IsEnabled() {
		h.Logger.Warnw("AnythingLLM is not configured, returning chart prompt only", "username", username)
		response.Success(w, http.StatusOK, "คำนวณข้อมูลพื้นดวงชะตาสำเร็จ", resp)
		return
	}
	if username == "" {
		h.Logger.Warnw("Missing username for AnythingLLM prediction, returning chart prompt only")
		resp.Warning = "ไม่สามารถเชื่อมต่อ AI ทำนายได้เนื่องจากไม่ได้ระบุ username"
		response.Success(w, http.StatusOK, "คำนวณข้อมูลพื้นดวงชะตาสำเร็จ (ไม่ระบุ username สำหรับ AI)", resp)
		return
	}

	prediction, err := h.AnythingLLMService.PredictNatalChart(r.Context(), username, result.Prompt)
	if err != nil {
		// จัดการ Graceful Fallback: เมื่อ AnythingLLM ขัดข้องหรือไม่สามารถเชื่อมต่อได้ (เช่น connection refused หรือ timeout)
		// ไม่ส่ง Error 502 กลับหา Client เพราะ:
		// 1) Cloudflare จะดักจับ HTTP 502 แล้วแทนที่ด้วยหน้าจอ Error 502 ของ Cloudflare ทันที ทำให้ Frontend พัง
		// 2) การคำนวณพิกัดดวงดาว (Prompt) จาก Lahiri สำเร็จแล้ว ผู้ใช้ควรได้รับข้อมูลผูกดวงตามปกติ
		anythingURL := ""
		if h.Cfg != nil {
			anythingURL = h.Cfg.AnythingLLMAPIURL
		}
		guide := "ตรวจสอบว่า AnythingLLM Desktop/Docker เปิดทำงานอยู่หรือไม่ และพอร์ตเปิดรับการเชื่อมต่อจริงหรือไม่"
		if strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "abort") {
			guide = "เชื่อมต่อ AnythingLLM ได้สำเร็จแล้ว แต่โมเดล LLM ภายใน AnythingLLM ตอบกลับข้อผิดพลาด (503 Service Unavailable) ให้ตรวจสอบการตั้งค่า AI Provider ในหน้า Settings > LLM Preference ของ AnythingLLM"
		}
		h.Logger.Warnw("AnythingLLM prediction unavailable, falling back to prompt only",
			"username", username,
			"anythingllm_url", anythingURL,
			"fallback_action", "HTTP 200 OK with prompt and warning message",
			"investigation_guide", guide,
			"error", err,
		)
		resp.Warning = "ไม่สามารถเชื่อมต่อบริการทำนาย AI ได้ในขณะนี้ ระบบได้ส่งคืนข้อมูลพื้นดวงชะตาสำหรับการผูกดวงแล้ว"
		response.Success(w, http.StatusOK, "คำนวณข้อมูลพื้นดวงชะตาสำเร็จ (ระบบทำนาย AI ไม่พร้อมใช้งานชั่วคราว)", resp)
		return
	}

	resp.Prediction = prediction.TextResponse
	resp.WorkspaceSlug = prediction.WorkspaceSlug
	resp.ThreadSlug = prediction.ThreadSlug

	h.Logger.Infow("Hora prediction completed successfully", "username", username, "workspace_slug", resp.WorkspaceSlug, "thread_slug", resp.ThreadSlug)
	response.Success(w, http.StatusOK, "พยากรณ์พื้นดวงชะตาสำเร็จ", resp)
}

// writeHoraError แปลง Error จาก Service เป็น JSON Response มาตรฐาน
// โดยป้องกันไม่ให้ส่ง HTTP 502 ไปยัง Reverse Proxy / Cloudflare ซึ่งจะทำให้ Cloudflare ตัดหน้าแสดง Error Page
func (h *AuthHandler) writeHoraError(w http.ResponseWriter, step, username string, err error) {
	var appErr *response.AppError
	if errors.As(err, &appErr) {
		h.Logger.Warnw(step+" failed with app error", "username", username, "code", appErr.Code, "message", appErr.Message, "error", appErr.Err)
		code := appErr.Code
		if code == http.StatusBadGateway {
			// แปลง 502 เป็น 500 เพื่อป้องกัน Cloudflare Interception
			code = http.StatusInternalServerError
		}
		response.Error(w, code, appErr.Message)
		return
	}

	h.Logger.Errorw(step+" failed with unexpected error", "username", username, "error", err)
	response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
}

func isValidHoraPredictRequest(req domain.HoraPredictRequest) bool {
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
	return true
}
