package handler

import (
	"horaveda-app/internal/config"
	"horaveda-app/internal/service"

	"go.uber.org/zap"
)

type AuthHandler struct {
	AuthService        *service.AuthService
	HoraService        *service.HoraService
	AnythingLLMService *service.AnythingLLMService // เชื่อมต่อ AnythingLLM สำหรับพยากรณ์พื้นดวง (1 User : 1 Workspace : 1 Thread)
	Logger             *zap.SugaredLogger
	Cfg                *config.Config // <-- เพิ่มฟิลด์ config ตรงนี้
}

func NewAuthHandler(authService *service.AuthService, logger *zap.SugaredLogger, cfg *config.Config) *AuthHandler {
	horaAPIURL := ""
	anythingLLMURL, anythingLLMKey := "", ""
	anythingLLMTimeout := service.DefaultAnythingLLMTimeout
	if cfg != nil {
		horaAPIURL = cfg.HoraAPIURL
		anythingLLMURL = cfg.AnythingLLMAPIURL
		anythingLLMKey = cfg.AnythingLLMAPIKey
		if cfg.AnythingLLMTimeout > 0 {
			anythingLLMTimeout = cfg.AnythingLLMTimeout
		}
	}
	return &AuthHandler{
		AuthService:        authService,
		HoraService:        service.NewHoraService(horaAPIURL, logger, nil),
		AnythingLLMService: service.NewAnythingLLMService(anythingLLMURL, anythingLLMKey, anythingLLMTimeout, logger, nil),
		Logger:             logger,
		Cfg:                cfg, // <-- กำหนดค่า config
	}
}
