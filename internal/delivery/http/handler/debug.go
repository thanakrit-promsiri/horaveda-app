package handler

import (
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
