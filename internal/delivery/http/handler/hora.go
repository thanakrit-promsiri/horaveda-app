package handler

import (
	"encoding/json"
	"errors"
	"net/http"

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

	var username string
	if claims, ok := middleware.GetUserClaims(r.Context()); ok {
		if u, exists := claims["username"].(string); exists {
			username = u
		}
	}

	var req domain.HoraPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !isValidHoraPredictRequest(req) {
		h.Logger.Warnw("Invalid hora predict request payload", "error", err, "username", username, "payload", req)
		response.Error(w, http.StatusBadRequest, "กรุณาระบุข้อมูลวันเดือนปีเกิด เวลาเกิด และพิกัดละติจูด/ลองจิจูดให้ถูกต้องครบถ้วน")
		return
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
		var appErr *response.AppError
		if errors.As(err, &appErr) {
			h.Logger.Warnw("Hora prompt calculation failed with app error", "username", username, "code", appErr.Code, "message", appErr.Message)
			response.Error(w, appErr.Code, appErr.Message)
			return
		}

		h.Logger.Errorw("Hora prompt calculation failed with unexpected error", "username", username, "error", err)
		response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
		return
	}

	h.Logger.Infow("Hora prompt calculated successfully", "username", username, "status", result.Status)
	response.Success(w, http.StatusOK, "คำนวณข้อมูลพื้นดวงชะตาสำเร็จ", result)
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
