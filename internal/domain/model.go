package domain

type RegisterRequest struct {
	Username  string `json:"username"`
	SerialKey string `json:"serial_key"`
}

type RevokeUserRequest struct {
	Username string `json:"username"`
}

type RevokeUserResponse struct {
	Username      string   `json:"username"`
	SupabaseEmail string   `json:"supabase_email"`
	RevokedKeys   []string `json:"revoked_keys"`
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

type HoraPredictRequest struct {
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Day       int     `json:"day"`
	HourThai  float64 `json:"hour_thai"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type HoraCalculateReq struct {
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Day         int     `json:"day"`
	HourThai    float64 `json:"hour_thai"`
	TimeZone    float64 `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	HouseSystem string  `json:"house_system"`
}

type HoraPromptRequest struct {
	CalculateReq HoraCalculateReq `json:"calculateReq"`
}

type HoraPromptResponse struct {
	Status string `json:"status"`
	Prompt string `json:"prompt"`
}
