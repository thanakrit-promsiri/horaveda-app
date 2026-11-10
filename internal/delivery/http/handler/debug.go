package handler

import (
	"errors"
	"net/http"

	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleDebugGetAllKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.AuthService.GetAllKeys()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(w, http.StatusOK, "ดึงข้อมูลคีย์ทั้งหมดสำเร็จ", keys)
}

// HandleGetSerialKeysStatus ตรวจสอบสถานะ Serial Key ทั้งหมดว่าคีย์ใดกำลังใช้งานอยู่ (used) หรือว่าง (available) โดยไม่ต้องใช้ Token
func (h *AuthHandler) HandleGetSerialKeysStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.Logger.Warnw("Method not allowed", "method", r.Method, "remote_addr", r.RemoteAddr)
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	result, err := h.AuthService.GetSerialKeysStatus()
	if err != nil {
		var appErr *response.AppError
		if errors.As(err, &appErr) {
			response.Error(w, appErr.Code, appErr.Message)
			return
		}
		response.Error(w, http.StatusInternalServerError, "ไม่สามารถดึงข้อมูลสถานะ Serial Key ได้")
		return
	}

	response.Success(w, http.StatusOK, "ดึงข้อมูลสถานะ Serial Key สำเร็จ", result)
}
