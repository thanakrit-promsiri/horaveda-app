package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"horaveda-app/internal/model"
	"horaveda-app/internal/service"
)

type AuthHandler struct {
	AuthService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

func (h *AuthHandler) HandleSerialRegister(w http.ResponseWriter, r *http.Request) {
	// CORS Headers (หากต้องการรองรับ Credentials ข้าม Domain แนะนำให้ระบุ Origin เจาะจงแทนเครื่องหมาย *)
	w.Header().Set("Access-Control-Allow-Origin", "https://hora.garbagecollection.info") // ปรับตามพอร์ตหน้าบ้านของคุณ หรือใช้ "*" ถ้าอยู่โดเมนเดียวกัน
	w.Header().Set("Access-Control-Allow-Credentials", "true")                           // สำคัญมากเมื่อใช้ Cookie ข้าม Origin
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"})
		return
	}

	var req model.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || req.SerialKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "กรุณากรอกข้อมูลให้ครบถ้วน"})
		return
	}

	token, err := h.AuthService.RegisterWithSerial(req.Username, req.SerialKey)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if err.Error() == "server error" || err.Error() == "เกิดข้อผิดพลาดในการอัปเดตสถานะคีย์" || err.Error() == "ไม่สามารถสร้างบัญชีผู้ใช้ได้" || err.Error() == "ไม่สามารถสร้าง Token ได้" {
			statusCode = http.StatusInternalServerError
		} else if err.Error() == "ชื่อผู้ใช้นี้ถูกใช้งานแล้ว กรุณาใช้ชื่ออื่น" {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	// ตั้งค่า HttpOnly Cookie สำหรับเก็บ Token เพื่อความปลอดภัยสูงสุด ป้องกัน XSS
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,                           // ป้องกันไม่ให้ JavaScript ฝั่ง Client อ่านค่า Cookie นี้ได้
		Secure:   false,                          // ตั้งเป็น true หากรันบน HTTPS (ถ้าทดสอบบน Localhost แบบ HTTP ให้ใช้ false)
		SameSite: http.SameSiteLaxMode,           // ป้องกัน CSRF เบื้องต้น
		Expires:  time.Now().Add(24 * time.Hour), // กำหนดอายุ Cookie (เช่น 1 วัน)
	}
	http.SetCookie(w, &cookie)

	// ส่ง Response กลับไป (ไม่ต้องแนบ token ไปใน JSON Body แล้ว)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "ลงทะเบียนสำเร็จ",
	})
}

func (h *AuthHandler) HandleDebugGetAllKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	keys, err := h.AuthService.GetAllKeys()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(keys)
}

// ตรวจสอบสิทธิ์จาก HttpOnly Cookie
func (h *AuthHandler) HandleVerifyAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://hora.garbagecollection.info") // หรือ "*" ตามหน้าเว็บที่ใช้งาน
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. อ่าน Cookie ชื่อ auth_token
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized"})
		return
	}

	// 2. ตรวจสอบความถูกต้องของ JWT Token ที่อยู่ใน Cookie
	claims, err := h.AuthService.VerifyToken(cookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid token"})
		return
	}

	// 3. ส่งข้อมูล username กลับไปแสดงผลที่หน้าเว็บ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username": claims["username"],
	})
}

// ออกจากระบบ (ลบ Cookie)
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://hora.garbagecollection.info")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// สั่งเคลียร์ Cookie โดยการตั้งค่าอายุเป็นอดีต (MaxAge: -1)
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // ใช้ true บน HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, &cookie)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "ออกจากระบบสำเร็จ"})
}
