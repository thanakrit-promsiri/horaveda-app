package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func (h *AuthHandler) HandleSerialRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req domain.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || req.SerialKey == "" || !usernameRegex.MatchString(req.Username) {
		h.Logger.Warnw("Invalid registration request payload", "error", err, "username", req.Username)
		response.Error(w, http.StatusBadRequest, "Username ต้องเป็นอักษรโรมันและตัวเลขเท่านั้น และกรอกข้อมูลให้ครบถ้วน")
		return
	}

	h.Logger.Infow("Processing serial registration", "username", req.Username)

	token, err := h.AuthService.RegisterWithSerial(req.Username, req.SerialKey)
	if err != nil {
		// ตรวจสอบว่าเป็น AppError ที่กำหนดมาจาก Service หรือไม่
		var appErr *response.AppError
		if errors.As(err, &appErr) {
			h.Logger.Warnw("Registration failed with app error", "username", req.Username, "code", appErr.Code, "message", appErr.Message)
			response.Error(w, appErr.Code, appErr.Message)
			return
		}

		// Fallback กรณีเกิด Error อื่นๆ ที่ไม่ได้คาดคิด
		h.Logger.Errorw("Registration failed with unexpected error", "username", req.Username, "error", err)
		response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
		return
	}

	secureCookie := false
	if h.Cfg != nil {
		secureCookie = h.Cfg.CookieSecure
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	http.SetCookie(w, &cookie)

	h.Logger.Infow("User registered successfully", "username", req.Username)
	response.Success(w, http.StatusOK, "ลงทะเบียนสำเร็จ", nil)
}
