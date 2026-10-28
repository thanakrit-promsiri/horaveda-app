package main

import (
	"net/http"

	"horaveda-app/internal/config"
	httpdelivery "horaveda-app/internal/delivery/http"
	"horaveda-app/internal/delivery/http/handler"
	"horaveda-app/internal/delivery/http/middleware"
	"horaveda-app/internal/repository"
	"horaveda-app/internal/service"
)

func main() {
	// โหลดคอนฟิกและเซอร์วิสพื้นฐาน
	cfg := config.LoadConfig()
	defer cfg.Logger.Sync()

	// บันทึก Log แจ้งเตือนว่าระบบกำลังเริ่มต้นโหลดคอมโพเนนต์
	cfg.Logger.Infow("กำลังเริ่มต้นระบบและเชื่อมต่อ Dependency Injection")

	// สร้าง Repository และ Service ตามลำดับ (Dependency Injection)
	authRepo := repository.NewAuthRepository(cfg.SupabaseClient, cfg.Logger)
	authService := service.NewAuthService(authRepo, cfg.Logger, cfg.JWTSecret)

	// ส่ง cfg เข้าไปใน NewAuthHandler เพื่อให้ Handler เข้าถึงค่า Config ได้
	authHandler := handler.NewAuthHandler(authService, cfg.Logger, cfg)

	// สร้าง Instance ของ Auth Middleware พร้อมส่ง cfg.Logger เข้าไป
	authMiddleware := middleware.NewAuthMiddleware(authService, cfg.Logger)

	// เรียกใช้งาน Router โดยส่ง authHandler, authMiddleware และ cfg.Logger เข้าไป
	mux := httpdelivery.NewRouter(authHandler, authMiddleware, cfg.Logger)

	// นำ Global CORS Middleware มาครอบ Mux Router โดยดึงค่า Origin จาก config
	corsHandler := middleware.CORSMiddleware(cfg.CORSOrigin)(mux)

	// บันทึก Log แจ้งสถานะพอร์ตที่เซิร์ฟเวอร์กำลังจะรัน
	cfg.Logger.Infow("Server is running", "port", cfg.Port)

	// รัน HTTP Server (ผ่าน corsHandler) และดักจับ Error กรณีที่ Server ล่มหรือเกิดข้อผิดพลาด
	if err := http.ListenAndServe(":"+cfg.Port, corsHandler); err != nil {
		cfg.Logger.Errorw("Server terminated with error", "error", err)
		cfg.Logger.Fatal(err)
	}
}
