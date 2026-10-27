package response

import (
	"net/http"
)

// AppError โครงสร้าง Error เฉพาะของแอปพลิเคชัน
type AppError struct {
	Code    int    // HTTP Status Code (เช่น 400, 401, 404, 500)
	Message string // ข้อความที่เป็นมิตรสำหรับส่งให้ผู้ใช้/Frontend
	Err     error  // Error ต้นทาง (สำหรับไว้ใช้เก็บ Log ภายในระบบ)
}

// ทำให้ AppError implements interface `error` มาตรฐานของ Go
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// --- ฟังก์ชันผู้ช่วยสร้าง Error สำเร็จรูป (Helper Constructors) ---

func NewBadRequestError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     err,
	}
}

func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     err,
	}
}

func NewNotFoundError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: message,
		Err:     err,
	}
}

func NewInternalServerError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}