package middleware

import (
	"context"
	"net/http"

	"horaveda-app/internal/service"
	"horaveda-app/pkg/response"

	"go.uber.org/zap"
)

type AuthMiddleware struct {
	AuthService *service.AuthService
	Logger      *zap.SugaredLogger
}

func NewAuthMiddleware(authService *service.AuthService, logger *zap.SugaredLogger) *AuthMiddleware {
	return &AuthMiddleware{
		AuthService: authService,
		Logger:      logger,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			m.Logger.Warnw("Authentication failed: auth_token cookie not found", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			response.Error(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อนใช้งาน")
			return
		}

		// 2. รับ claims กลับมาจากการ Verify
		claims, err := m.AuthService.VerifyToken(cookie.Value)
		if err != nil {
			m.Logger.Warnw("Authentication failed: invalid or expired token", "error", err, "remote_addr", r.RemoteAddr)
			if appErr, ok := err.(*response.AppError); ok {
				response.Error(w, appErr.Code, appErr.Message)
				return
			}
			response.Error(w, http.StatusUnauthorized, "Token ไม่ถูกต้องหรือหมดอายุ")
			return
		}

		m.Logger.Infow("Authentication successful", "path", r.URL.Path)

		// 3. แนบ claims ไปกับ Context ของ Request เพื่อส่งต่อให้ Handler
		ctx := context.WithValue(r.Context(), "userClaims", claims)
		
		// 4. ส่ง Request ที่มี Context ใหม่ไปยัง Handler ถัดไป
		next(w, r.WithContext(ctx))
	}
}