package handler

import (
	"horaveda-app/internal/config"
	"horaveda-app/internal/service"

	"go.uber.org/zap"
)

type AuthHandler struct {
	AuthService *service.AuthService
	Logger      *zap.SugaredLogger
	Cfg         *config.Config // <-- เพิ่มฟิลด์ config ตรงนี้
}

func NewAuthHandler(authService *service.AuthService, logger *zap.SugaredLogger, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
		Logger:      logger,
		Cfg:         cfg, // <-- กำหนดค่า config
	}
}
