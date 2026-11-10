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

// SerialKeyStatusItem ข้อมูลสถานะของแต่ละ Serial Key
type SerialKeyStatusItem struct {
	ID       int    `json:"id"`
	KeyCode  string `json:"key_code"`
	IsUsed   bool   `json:"is_used"`
	Status   string `json:"status"` // "used" (ใช้อยู่) หรือ "available" (ว่าง)
	UserID   *int   `json:"user_id"`
	Username string `json:"username,omitempty"`
}

// SerialKeyStatusResponse ผลลัพธ์สรุปสถานะ Serial Key ทั้งหมดในระบบ (ใช้อยู่ / ว่าง)
type SerialKeyStatusResponse struct {
	TotalKeys      int                   `json:"total_keys"`
	AvailableCount int                   `json:"available_count"`
	UsedCount      int                   `json:"used_count"`
	AvailableKeys  []string              `json:"available_keys"`
	UsedKeys       []string              `json:"used_keys"`
	Keys           []SerialKeyStatusItem `json:"keys"`
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

// HoraTransitPredictRequest รับค่าเพื่อคำนวณและพยากรณ์ดวงชะตาดาวจรประจำเดือน
type HoraTransitPredictRequest struct {
	// Username ใช้เป็นชื่ออ้างอิง Workspace บน AnythingLLM (หากไม่ส่งมาจะใช้ username จาก JWT แทน)
	Username     string  `json:"username,omitempty"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	Day          int     `json:"day"`
	HourThai     float64 `json:"hour_thai"`
	TimeZone     float64 `json:"time_zone,omitempty"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	HouseSystem  string  `json:"house_system,omitempty"`
	TransitYear  int     `json:"transit_year"`
	TransitMonth int     `json:"transit_month"`
}

// HoraTransitCalculateReq โครงสร้างส่งไปยัง Lahiri Transit Dasha Prompt API
type HoraTransitCalculateReq struct {
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	Day          int     `json:"day"`
	HourThai     float64 `json:"hour_thai"`
	TimeZone     float64 `json:"time_zone"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	HouseSystem  string  `json:"house_system"`
	TransitYear  int     `json:"transit_year"`
	TransitMonth int     `json:"transit_month"`
}

// HoraTransitPromptRequest ห่อ calculateReq ส่งไปยัง Lahiri Transit Dasha Prompt API
type HoraTransitPromptRequest struct {
	CalculateReq HoraTransitCalculateReq `json:"calculateReq"`
}

// HoraTransitPredictResponse ผลลัพธ์ที่ส่งกลับไปยัง Client ของ /api/v1/hora/transitpredict
type HoraTransitPredictResponse struct {
	Status           string `json:"status"`
	NatalPrompt      string `json:"natal_prompt"`                 // พิกัดดาวชะตากำเนิดจาก Lahiri Prompt API
	TransitPrompt    string `json:"transit_prompt"`               // พิกัดดาวจรและทศาจาก Lahiri Transit Dasha Prompt API
	TransitMonth     int    `json:"transit_month"`                // เดือนของดาวจร (1-12)
	TransitYear      int    `json:"transit_year"`                 // ปีของดาวจร (ค.ศ. หรือ พ.ศ.)
	TransitMonthThai string `json:"transit_month_thai"`           // ชื่อเดือนและปีภาษาไทย (เช่น "ธันวาคม พ.ศ. 2569")
	Prediction       string `json:"prediction"`                   // คำทำนายดาวจรประจำเดือนจาก AnythingLLM
	WorkspaceSlug    string `json:"workspace_slug,omitempty"`     // Workspace ประจำผู้ใช้บน AnythingLLM
	ThreadSlug       string `json:"thread_slug,omitempty"`        // Thread ประจำการทำนายดาวจรบน AnythingLLM
	Warning          string `json:"warning,omitempty"`            // ข้อความแจ้งเตือนกรณีระบบ AI ทำนายไม่พร้อมใช้งาน (Graceful Fallback)
}

// AnythingLLMPrediction ผลลัพธ์จากการสนทนากับ AnythingLLM (1 User : 1 Workspace : 1 Thread)
type AnythingLLMPrediction struct {
	WorkspaceSlug string
	ThreadSlug    string
	TextResponse  string
}
