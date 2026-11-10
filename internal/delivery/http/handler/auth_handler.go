package handler

import (
	"strings"

	"horaveda-app/internal/config"
	"horaveda-app/internal/domain"
	"horaveda-app/internal/service"

	"go.uber.org/zap"
)

type AuthHandler struct {
	AuthService        domain.AuthUsecase
	HoraService        domain.HoraUsecase
	AnythingLLMService domain.AnythingLLMUsecase // เชื่อมต่อ AnythingLLM สำหรับพยากรณ์พื้นดวง (1 User : 1 Workspace : 1 Thread)
	Logger             *zap.SugaredLogger
	Cfg                *config.Config
}

// NewAuthHandlerWithServices สร้าง Handler โดยรับ Dependencies ที่ฉีด (Inject) มาจากภายนอกตามหลัก Clean Architecture
func NewAuthHandlerWithServices(
	authService domain.AuthUsecase,
	horaService domain.HoraUsecase,
	anythingLLMService domain.AnythingLLMUsecase,
	logger *zap.SugaredLogger,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		AuthService:        authService,
		HoraService:        horaService,
		AnythingLLMService: anythingLLMService,
		Logger:             logger,
		Cfg:                cfg,
	}
}

func NewAuthHandler(authService domain.AuthUsecase, logger *zap.SugaredLogger, cfg *config.Config) *AuthHandler {
	horaAPIURL := ""
	horaTransitAPIURL := ""
	anythingLLMURL, anythingLLMKey := "", ""
	anythingLLMTimeout := service.DefaultAnythingLLMTimeout
	if cfg != nil {
		horaAPIURL = cfg.HoraAPIURL
		horaTransitAPIURL = cfg.HoraTransitAPIURL
		anythingLLMURL = cfg.AnythingLLMAPIURL
		anythingLLMKey = cfg.AnythingLLMAPIKey
		if cfg.AnythingLLMTimeout > 0 {
			anythingLLMTimeout = cfg.AnythingLLMTimeout
		}
	}
	return NewAuthHandlerWithServices(
		authService,
		service.NewHoraServiceWithTransit(horaAPIURL, horaTransitAPIURL, logger, nil),
		service.NewAnythingLLMService(anythingLLMURL, anythingLLMKey, anythingLLMTimeout, logger, nil),
		logger,
		cfg,
	)
}

// logLLMFallback บันทึก Log แจ้งเตือนและคืนข้อความแนะนำกรณี AnythingLLM ไม่พร้อมใช้งาน
func (h *AuthHandler) logLLMFallback(predictionType, username string, err error) {
	anythingURL := ""
	if h.Cfg != nil {
		anythingURL = h.Cfg.AnythingLLMAPIURL
	}
	guide := "ตรวจสอบว่า AnythingLLM Desktop/Docker เปิดทำงานอยู่หรือไม่ และพอร์ตเปิดรับการเชื่อมต่อจริงหรือไม่"
	if err != nil && (strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "abort")) {
		guide = "เชื่อมต่อ AnythingLLM ได้สำเร็จแล้ว แต่โมเดล LLM ภายใน AnythingLLM ตอบกลับข้อผิดพลาด (503 Service Unavailable) ให้ตรวจสอบการตั้งค่า AI Provider ในหน้า Settings > LLM Preference ของ AnythingLLM"
	}
	h.Logger.Warnw("AnythingLLM "+predictionType+" prediction unavailable, falling back to prompt only",
		"username", username,
		"anythingllm_url", anythingURL,
		"fallback_action", "HTTP 200 OK with prompt and warning message",
		"investigation_guide", guide,
		"error", err,
	)
}
