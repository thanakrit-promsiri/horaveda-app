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
	// Username ใช้เป็นชื่ออ้างอิง Workspace บน AnythingLLM (หากไม่ส่งมาจะใช้ username จาก JWT แทน)
	Username  string  `json:"username,omitempty"`
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

// HoraPredictResponse ผลลัพธ์ที่ส่งกลับไปยัง Client ของ /api/v1/hora/predict
type HoraPredictResponse struct {
	Status        string `json:"status"`
	Prompt        string `json:"prompt"`                   // พิกัดดวงดาวจาก Lahiri Prompt API
	Prediction    string `json:"prediction"`               // คำทำนายพื้นดวงชะตาจาก AnythingLLM
	WorkspaceSlug string `json:"workspace_slug,omitempty"` // Workspace ประจำผู้ใช้บน AnythingLLM
	ThreadSlug    string `json:"thread_slug,omitempty"`    // Thread ประจำผู้ใช้บน AnythingLLM
	Warning       string `json:"warning,omitempty"`        // ข้อความแจ้งเตือนกรณีระบบ AI ทำนายไม่พร้อมใช้งาน (Graceful Fallback)
}

// AnythingLLMPrediction ผลลัพธ์จากการสนทนากับ AnythingLLM (1 User : 1 Workspace : 1 Thread)
type AnythingLLMPrediction struct {
	WorkspaceSlug string
	ThreadSlug    string
	TextResponse  string
}
