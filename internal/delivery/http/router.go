package http

import (
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

	// Protected Routes (ต้องผ่านการตรวจสอบสิทธิ์ผ่าน Auth Middleware ก่อน พร้อมใส่ Log)
	mux.HandleFunc("GET /api/v1/auth/verify", logRequest(authMiddleware.RequireAuth(authHandler.HandleVerifyAuth), "VerifyAuth"))

	return mux
}