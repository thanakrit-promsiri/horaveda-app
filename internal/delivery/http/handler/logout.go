package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"
)

// HandleLogout ล้าง Cookie auth_token พร้อมทั้งถอนสิทธิ์การใช้งาน Serial Key และลบบัญชีผู้ใช้ออกจากระบบ (เหมือน Revoke User)
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 1. ล้าง Cookie auth_token เสมอ (กำหนด Secure: true ใน Struct เพื่อผ่านเกณฑ์ความปลอดภัย SonarCloud)
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	if h.Cfg != nil && !h.Cfg.CookieSecure {
		cookie.Secure = h.Cfg.CookieSecure
	}
	http.SetCookie(w, &cookie)

	// 2. ดึง username จาก Context (กรณีผ่าน Middleware) หรือจาก Token ใน Cookie / Authorization Header
	var claimsUsername string
	if claims, ok := middleware.GetUserClaims(r.Context()); ok {
		if u, exists := claims["username"].(string); exists {
			claimsUsername = strings.TrimSpace(u)
		}
	}

	if claimsUsername == "" && h.AuthService != nil {
		var tokenValue string
		if c, err := r.Cookie("auth_token"); err == nil && strings.TrimSpace(c.Value) != "" {
			tokenValue = strings.TrimSpace(c.Value)
		} else {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				tokenValue = strings.TrimSpace(authHeader[7:])
			}
		}

		if tokenValue != "" {
			if claims, err := h.AuthService.VerifyTokenMap(tokenValue); err == nil {
				if u, exists := claims["username"].(string); exists {
					claimsUsername = strings.TrimSpace(u)
				}
			}
		}
	}

	// 3. รองรับการส่ง {"username": "..."} มาใน Request Body ด้วย
	var bodyUsername string
	if r.Body != nil {
		var req domain.RevokeUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			h.Logger.Warnw("Invalid logout request JSON payload", "error", err)
			response.Error(w, http.StatusBadRequest, "รูปแบบข้อมูล JSON ไม่ถูกต้อง")
			return
		}
		bodyUsername = strings.TrimSpace(req.Username)
	}

	if bodyUsername != "" {
		if !usernameRegex.MatchString(bodyUsername) {
			h.Logger.Warnw("Invalid username in logout request", "username", bodyUsername)
			response.Error(w, http.StatusBadRequest, "กรุณาระบุ username ที่ถูกต้อง (เฉพาะอักษรโรมันและตัวเลข)")
			return
		}
		if claimsUsername != "" && !strings.EqualFold(bodyUsername, claimsUsername) {
			h.Logger.Warnw("Logout username does not match authenticated token", "body_username", bodyUsername, "token_username", claimsUsername)
			response.Error(w, http.StatusForbidden, "username ไม่ตรงกับบัญชีที่เข้าสู่ระบบ")
			return
		}
	}

	username := claimsUsername
	if username == "" {
		username = bodyUsername
	}

	// 4. หากมีข้อมูล username ให้ทำการ Revoke Serial Key และลบ User ออกจากทั้ง public.users และ Supabase Auth
	if username != "" && h.AuthService != nil {
		h.Logger.Infow("Processing user revocation and deletion on logout", "username", username, "remote_addr", r.RemoteAddr)

		result, err := h.AuthService.RevokeUserAndSerial(username)
		if err != nil {
			var appErr *response.AppError
			if errors.As(err, &appErr) {
				h.Logger.Warnw("User revocation on logout failed with app error", "username", username, "code", appErr.Code, "message", appErr.Message)
				response.Error(w, appErr.Code, appErr.Message)
				return
			}

			h.Logger.Errorw("User revocation on logout failed with unexpected error", "username", username, "error", err)
			response.Error(w, http.StatusInternalServerError, "เกิดข้อผิดพลาดภายในระบบ")
			return
		}

		h.Logger.Infow("User logged out, serial key revoked and user deleted successfully", "username", username, "revoked_keys", result.RevokedKeys)
		response.Success(w, http.StatusOK, "ออกจากระบบ ถอนสิทธิ์การใช้งาน Serial Key และลบบัญชีผู้ใช้สำเร็จ", result)
		return
	}

	h.Logger.Infow("User logged out successfully", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
	response.Success(w, http.StatusOK, "ออกจากระบบสำเร็จ", nil)
}
