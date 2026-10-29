package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// กำหนดค่า HTTP Server พร้อม Timeout มาตรฐานเพื่อป้องกัน Slowloris Attack
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ดักจับสัญญาณปิดระบบ (SIGINT, SIGTERM) เพื่อทำ Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		// บันทึก Log แจ้งสถานะพอร์ตที่เซิร์ฟเวอร์กำลังจะรัน
		cfg.Logger.Infow("Server is running", "port", cfg.Port)

		// รัน HTTP Server (ผ่าน corsHandler) และดักจับ Error กรณีที่ Server ล่มหรือเกิดข้อผิดพลาด
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cfg.Logger.Errorw("Server terminated with error", "error", err)
			cfg.Logger.Fatal(err)
		}
	}()

	<-ctx.Done()
	cfg.Logger.Infow("Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		cfg.Logger.Errorw("Server forced to shutdown", "error", err)
	}
}
