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

	// สร้าง Repository และ Service ตามลำดับ (Dependency Injection ตามหลัก Clean Architecture)
	authRepo := repository.NewAuthRepository(cfg.SupabaseClient, cfg.Logger)
	authService := service.NewAuthService(authRepo, cfg.Logger, cfg.JWTSecret)
	horaService := service.NewHoraServiceWithTransit(cfg.HoraAPIURL, cfg.HoraTransitAPIURL, cfg.Logger, nil)
	anythingLLMService := service.NewAnythingLLMService(cfg.AnythingLLMAPIURL, cfg.AnythingLLMAPIKey, cfg.AnythingLLMTimeout, cfg.Logger, nil)

	// ฉีด (Inject) Services ทั้งหมดเข้าไปใน Handler โดยไม่ให้ Handler สร้าง Service เอง
	authHandler := handler.NewAuthHandlerWithServices(authService, horaService, anythingLLMService, cfg.Logger, cfg)

	// สร้าง Instance ของ Auth Middleware พร้อมส่ง cfg.Logger เข้าไป
	authMiddleware := middleware.NewAuthMiddleware(authService, cfg.Logger)

	// เรียกใช้งาน Router โดยส่ง authHandler, authMiddleware และ cfg.Logger เข้าไป
	mux := httpdelivery.NewRouter(authHandler, authMiddleware, cfg.Logger)

	// นำ Global CORS Middleware มาครอบ Mux Router โดยดึงค่า Origin จาก config
	corsHandler := middleware.CORSMiddleware(cfg.CORSOrigin)(mux)

	// กำหนดค่า HTTP Server พร้อม Timeout มาตรฐานเพื่อป้องกัน Slowloris Attack
	// WriteTimeout ต้องนานกว่าเวลารอ AnythingLLM (+ เวลาคำนวณพื้นดวง) ไม่เช่นนั้นคำตอบจะถูกตัดกลางทาง
	writeTimeout := 15 * time.Second
	if llmWrite := cfg.AnythingLLMTimeout + 30*time.Second; llmWrite > writeTimeout {
		writeTimeout = llmWrite
	}
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: writeTimeout,
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

	// ตรวจสอบสถานะการเชื่อมต่อ AnythingLLM เมื่อเริ่มต้นระบบ (แบบ Non-blocking ใน Background)
	go func() {
		time.Sleep(500 * time.Millisecond)
		if anythingLLMService != nil && anythingLLMService.IsEnabled() {
			probeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := anythingLLMService.Ping(probeCtx); err != nil {
				cfg.Logger.Warnw("AnythingLLM is unreachable at startup (AI predictions will fall back to prompt-only until service is up)",
					"url", anythingLLMService.BaseURL,
					"error", err,
					"troubleshooting", "1) ตรวจสอบว่า AnythingLLM เปิดทำงานอยู่หรือไม่ 2) ตรวจสอบว่าเปิดโหมด Developer API และใช้พอร์ต 3001 หรือไม่",
				)
			} else {
				cfg.Logger.Infow("AnythingLLM connection verified successfully", "url", anythingLLMService.BaseURL)
			}
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
