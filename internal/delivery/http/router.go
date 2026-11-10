package http

import (
	"fmt"
	"net/http"

	"horaveda-app/internal/delivery/http/handler"
	"horaveda-app/internal/delivery/http/middleware"

	"go.uber.org/zap"
)

// NewRouter ทำหน้าที่ลงทะเบียน Routes ทั้งหมดของแอปพลิเคชัน พร้อมใส่ Log ติดตามคำขอ
func NewRouter(authHandler *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware, log *zap.SugaredLogger) *http.ServeMux {
	mux := http.NewServeMux()

	// Helper สำหรับพิมพ์ Log ทุกครั้งที่มี Request เข้ามา
	logRequest := func(next http.HandlerFunc, routeName string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Infow("Incoming HTTP Request", "route", routeName, "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			next(w, r)
		}
	}

	// Public Routes (ไม่ต้องล็อกอินก็เข้าถึงได้ พร้อมใส่ Log)
	mux.HandleFunc("POST /api/v1/auth/register-serial", logRequest(authHandler.HandleSerialRegister, "RegisterSerial"))
	mux.HandleFunc("POST /api/v1/auth/logout", logRequest(authHandler.HandleLogout, "Logout"))
	mux.HandleFunc("GET /api/v1/auth/serial-keys", logRequest(authHandler.HandleGetSerialKeysStatus, "GetSerialKeysStatus"))
	mux.HandleFunc("POST /api/v1/auth/serial-keys", logRequest(authHandler.HandleGetSerialKeysStatus, "PostSerialKeysStatus"))

	// Protected Routes (ต้องผ่านการตรวจสอบสิทธิ์ผ่าน Auth Middleware ก่อน พร้อมใส่ Log)
	mux.HandleFunc("GET /api/v1/auth/verify", logRequest(authMiddleware.RequireAuth(authHandler.HandleVerifyAuth), "VerifyAuth"))
	mux.HandleFunc("POST /api/v1/hora/predict", logRequest(authMiddleware.RequireAuth(authHandler.HandleHoraPredict), "HoraPredict"))
	mux.HandleFunc("POST /api/v1/hora/prompt", logRequest(authMiddleware.RequireAuth(authHandler.HandleHoraPredict), "HoraPrompt"))
	mux.HandleFunc("POST /api/v1/hora/transitpredict", logRequest(authMiddleware.RequireAuth(authHandler.HandleHoraTransitPredict), "HoraTransitPredict"))

	// 12 เส้นครอบรายเดือน ประจำปี พ.ศ. 2570 (ค.ศ. 2027): POST /api/v1/hora/transitpredict_1 ถึง transitpredict_12
	for m := 1; m <= 12; m++ {
		month := m
		pattern := fmt.Sprintf("POST /api/v1/hora/transitpredict_%d", month)
		routeName := fmt.Sprintf("HoraTransitPredict_%d", month)
		mux.HandleFunc(pattern, logRequest(authMiddleware.RequireAuth(authHandler.HandleHoraTransitPredictMonth(month)), routeName))
	}

	// Admin Routes (ต้องผ่านการตรวจสอบ Admin API Key ผ่าน Header X-Admin-Key หรือ Authorization Bearer)
	adminKey := ""
	if authHandler != nil && authHandler.Cfg != nil {
		adminKey = authHandler.Cfg.AdminAPIKey
	}
	mux.HandleFunc("POST /api/v1/auth/revoke-user", logRequest(authMiddleware.RequireAdminKey(adminKey, authHandler.HandleRevokeUser), "RevokeUserPost"))
	mux.HandleFunc("DELETE /api/v1/auth/revoke-user", logRequest(authMiddleware.RequireAdminKey(adminKey, authHandler.HandleRevokeUser), "RevokeUserDelete"))

	return mux
}
