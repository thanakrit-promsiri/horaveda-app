package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"
)

// HandleRevokeUser ถอนสิทธิ์การใช้งาน Serial Key และลบข้อมูลผู้ใช้ออกจากตาราง users และ Supabase Auth
func (h *AuthHandler) HandleRevokeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req domain.RevokeUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || !usernameRegex.MatchString(req.Username) {
		h.Logger.Warnw("Invalid revoke user request payload", "error", err, "username", req.Username)
		response.Error(w, http.StatusBadRequest, "กรุณาระบุ username ที่ถูกต้อง (เฉพาะอักษรโรมันและตัวเลข)")
		return
	}

	h.Logger.Infow("Processing user revocation and deletion", "username", req.Username)

	result, err := h.AuthService.RevokeUserAndSerial(req.Username)
	if err != nil {
		var appErr *response.AppError
		if errors.As(err, &appErr) {
			h.Logger.Warnw("User revocation failed with app error", "username", req.Username, "code", appErr.Code, "message", appErr.Message)
			response.Error(w, appErr.Code, appErr.Message)
			return
		}

		h.Logger.Errorw("User revocation failed with unexpected error", "username", req.Username, "error", err)
		response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
		return
	}

	h.Logger.Infow("User and serial key revoked successfully", "username", req.Username, "revoked_keys", result.RevokedKeys)
	response.Success(w, http.StatusOK, "ถอนสิทธิ์การใช้งาน Serial Key และลบบัญชีผู้ใช้สำเร็จ", result)
}
