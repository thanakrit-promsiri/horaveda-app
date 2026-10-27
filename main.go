package main

import (
	"net/http"

	"horaveda-app/internal/config"
	"horaveda-app/internal/handler"
	"horaveda-app/internal/service"
)

func main() {
	// โหลดคอนฟิกและเซอร์วิสพื้นฐาน
	cfg := config.LoadConfig()
	defer cfg.Logger.Sync()

	// สร้าง Service และ Handler
	authService := service.NewAuthService(cfg.SupabaseClient, cfg.Logger, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)

	// กำหนด Routes
	http.HandleFunc("/api/v1/auth/register-serial", authHandler.HandleSerialRegister)
	// กำหนด Routes เพิ่มเติม

	http.HandleFunc("/api/v1/auth/verify", authHandler.HandleVerifyAuth) // เพิ่มเส้นทางเช็กสิทธิ์
	http.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)     // เพิ่มเส้นทางออกจากระบบ

	cfg.Logger.Infow("Server is running", "port", cfg.Port)
	cfg.Logger.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
