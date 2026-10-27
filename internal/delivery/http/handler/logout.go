package handler

import (
	"net/http"

	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://hora.garbagecollection.info")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, &cookie)

	h.Logger.Infow("User logged out successfully", "remote_addr", r.RemoteAddr, "path", r.URL.Path)

	response.Success(w, http.StatusOK, "ออกจากระบบสำเร็จ", nil)
}