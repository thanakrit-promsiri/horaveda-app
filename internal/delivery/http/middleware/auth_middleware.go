package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"horaveda-app/internal/service"
	"horaveda-app/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// contextKey ชนิดข้อมูลเฉพาะสำหรับใช้เป็น Key ใน Context ตามแนวทางปฏิบัติที่ดีของ Go
type contextKey string

// UserClaimsKey คีย์มาตรฐานสำหรับจัดเก็บและดึงข้อมูล Claims ของผู้ใช้จาก Context
const UserClaimsKey contextKey = "userClaims"

// GetUserClaims ดึงข้อมูล Claims ของผู้ใช้ออกจาก Context โดยรองรับทั้ง jwt.MapClaims และ map[string]interface{}
func GetUserClaims(ctx context.Context) (jwt.MapClaims, bool) {
	for _, key := range []interface{}{UserClaimsKey, "userClaims"} {
		val := ctx.Value(key)
		if val == nil {
			continue
		}
		if claims, ok := val.(jwt.MapClaims); ok {
			return claims, true
		}
		if rawMap, ok := val.(map[string]interface{}); ok {
			return jwt.MapClaims(rawMap), true
		}
	}
	return nil, false
}

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
		var tokenValue string
		if cookie, err := r.Cookie("auth_token"); err == nil && strings.TrimSpace(cookie.Value) != "" {
			tokenValue = strings.TrimSpace(cookie.Value)
		} else {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				tokenValue = strings.TrimSpace(authHeader[7:])
			}
		}

		if tokenValue == "" {
			m.Logger.Warnw("Authentication failed: auth_token cookie not found", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			response.Error(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อนใช้งาน")
			return
		}

		// 2. รับ claims กลับมาจากการ Verify
		claims, err := m.AuthService.VerifyToken(tokenValue)
		if err != nil {
			m.Logger.Warnw("Authentication failed: invalid or expired token", "error", err, "remote_addr", r.RemoteAddr)
			var appErr *response.AppError
			if errors.As(err, &appErr) {
				response.Error(w, appErr.Code, appErr.Message)
				return
			}
			response.Error(w, http.StatusUnauthorized, "Token ไม่ถูกต้องหรือหมดอายุ")
			return
		}

		m.Logger.Infow("Authentication successful", "path", r.URL.Path)

		// 3. แนบ claims ไปกับ Context ของ Request เพื่อส่งต่อให้ Handler
		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)

		// 4. ส่ง Request ที่มี Context ใหม่ไปยัง Handler ถัดไป
		next(w, r.WithContext(ctx))
	}
}

// RequireAdminKey ตรวจสอบสิทธิ์ระดับผู้ดูแลระบบผ่าน Header X-Admin-Key หรือ Authorization: Bearer <key>
// โดยเปรียบเทียบคีย์แบบ Constant-Time เพื่อป้องกันการโจมตีแบบ Timing Attack
func (m *AuthMiddleware) RequireAdminKey(adminKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminKey == "" {
			m.Logger.Errorw("Admin endpoint called but AdminAPIKey is not configured", "path", r.URL.Path)
			response.Error(w, http.StatusInternalServerError, "ระบบยังไม่ได้ตั้งค่าคีย์ผู้ดูแลระบบ")
			return
		}

		providedKey := strings.TrimSpace(r.Header.Get("X-Admin-Key"))
		if providedKey == "" {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				providedKey = strings.TrimSpace(authHeader[7:])
			}
		}

		if providedKey == "" {
			m.Logger.Warnw("Admin authentication failed: missing X-Admin-Key or Bearer token", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			response.Error(w, http.StatusUnauthorized, "กรุณาระบุ X-Admin-Key หรือ Authorization Bearer สำหรับผู้ดูแลระบบ")
			return
		}

		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(adminKey)) != 1 {
			m.Logger.Warnw("Admin authentication failed: invalid admin key", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			response.Error(w, http.StatusForbidden, "ไม่มีสิทธิ์เข้าถึงส่วนของผู้ดูแลระบบ (Invalid Admin Key)")
			return
		}

		m.Logger.Infow("Admin authentication successful", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		next(w, r)
	}
}
