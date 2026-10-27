package response

import (
	"encoding/json"
	"net/http"
)

// Response โครงสร้าง JSON กลางสำหรับส่งกลับหา Client (Frontend)
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Success ส่ง Response เมื่อการทำงานสำเร็จ (HTTP 200, 201 ฯลฯ)
func Success(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error ส่ง Response เมื่อเกิดข้อผิดพลาด (HTTP 400, 401, 500 ฯลฯ)
func Error(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: message,
	})
}