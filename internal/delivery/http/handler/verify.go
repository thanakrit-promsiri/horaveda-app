package handler

import (
	"net/http"

	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleVerifyAuth(w http.ResponseWriter, r *http.Request) {
	// ดึงข้อมูล claims ที่แนบมาจาก Auth Middleware ผ่าน Context
	claims, ok := middleware.GetUserClaims(r.Context())
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
