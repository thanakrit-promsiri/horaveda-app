package handler

import (
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
