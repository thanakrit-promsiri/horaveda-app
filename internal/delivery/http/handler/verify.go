package handler

import (
	"net/http"

	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleVerifyAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", h.Cfg.CORSOrigin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ดึงข้อมูล claims ที่แนบมาจาก Auth Middleware ผ่าน Context
	claims, ok := r.Context().Value("userClaims").(map[string]interface{})
	var username string
	if ok {
		if u, exists := claims["username"].(string); exists {
			username = u
		}
	}

	// บันทึก Log ผ่าน Logger ของ Handler
	h.Logger.Infow("Authentication verified successfully", "username", username, "path", r.URL.Path)

	response.Success(w, http.StatusOK, "ตรวจสอบสิทธิ์สำเร็จ", map[string]interface{}{
		"username": username,
	})
}
