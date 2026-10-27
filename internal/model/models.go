package model

type RegisterRequest struct {
	Username  string `json:"username"`
	SerialKey string `json:"serial_key"`
}

type SerialKey struct {
	ID      int    `json:"id"`
	KeyCode string `json:"key_code"`
	IsUsed  bool   `json:"is_used"`
	UserID  *int   `json:"user_id"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}