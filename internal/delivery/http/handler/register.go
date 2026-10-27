package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleSerialRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://hora.garbagecollection.info")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req domain.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || req.SerialKey == "" {
		h.Logger.Warnw("Invalid registration request payload", "error", err, "username", req.Username)
		response.Error(w, http.StatusBadRequest, "กรุณากรอกข้อมูลให้ครบถ้วน")
		return
	}

	h.Logger.Infow("Processing serial registration", "username", req.Username)

	token, err := h.AuthService.RegisterWithSerial(req.Username, req.SerialKey)
	if err != nil {
		// ตรวจสอบว่าเป็น AppError ที่กำหนดมาจาก Service หรือไม่
		if appErr, ok := err.(*response.AppError); ok {
			h.Logger.Warnw("Registration failed with app error", "username", req.Username, "code", appErr.Code, "message", appErr.Message)
			response.Error(w, appErr.Code, appErr.Message)
			return
		}

		// Fallback กรณีเกิด Error อื่นๆ ที่ไม่ได้คาดคิด
		h.Logger.Errorw("Registration failed with unexpected error", "username", req.Username, "error", err)
		response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
		return
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	http.SetCookie(w, &cookie)

	h.Logger.Infow("User registered successfully", "username", req.Username)
	response.Success(w, http.StatusOK, "ลงทะเบียนสำเร็จ", nil)
}