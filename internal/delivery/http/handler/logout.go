package handler

import (
	"net/http"

	"horaveda-app/pkg/response"
)

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	secureCookie := false
	if h.Cfg != nil {
		secureCookie = h.Cfg.CookieSecure
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, &cookie)

	h.Logger.Infow("User logged out successfully", "remote_addr", r.RemoteAddr, "path", r.URL.Path)

	response.Success(w, http.StatusOK, "ออกจากระบบสำเร็จ", nil)
}
