package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"horaveda-app/pkg/logger"

	"github.com/joho/godotenv"
	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
)

type Config struct {
	SupabaseClient *supabase.Client
	Logger         *zap.SugaredLogger
	Port           string
	JWTSecret      []byte
	CORSOrigin     string // <-- เพิ่มฟิลด์นี้เข้าไป
	CookieSecure   bool   // กำหนดค่า Secure ของ Cookie ให้สอดคล้องกันทั้งระบบ
	AdminAPIKey    string // คีย์สำหรับยืนยันสิทธิ์ผู้ดูแลระบบ (Admin Endpoints)
	HoraAPIURL     string // URL สำหรับเรียกใช้บริการคำนวณพื้นดวงชะตา (Lahiri Prompt API)
	HoraTransitAPIURL string // URL สำหรับเรียกใช้บริการคำนวณดาวจร (Lahiri Transit Dasha Prompt API)

	AnythingLLMAPIURL  string        // URL ของ AnythingLLM Developer API (Default: http://localhost:3001/api/v1)
	AnythingLLMAPIKey  string        // API Key ของ AnythingLLM (ว่าง = ปิดการพยากรณ์ด้วย AnythingLLM)
	AnythingLLMTimeout time.Duration // ระยะเวลารอคำตอบจาก AnythingLLM สูงสุด
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	// เรียกใช้ Logger ส่วนกลางจาก pkg/logger เพื่อลดความซ้ำซ้อนของโค้ด
	sugar := logger.NewLogger()

	supabaseUrl := os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
	supabaseKey := os.Getenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY")
	keyType := "Service Role Key (Bypass RLS)"

	if supabaseKey == "" {
		supabaseKey = os.Getenv("NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY")
		keyType = "Publishable/Anon Key"
	}

	if supabaseUrl == "" || supabaseKey == "" {
		sugar.Fatal("กรุณากำหนดค่า NEXT_PUBLIC_SUPABASE_URL และ Key ให้ครบถ้วนในไฟล์ .env")
	}

	sugar.Infow("กำลังเชื่อมต่อ Supabase", "key_type", keyType)

	client, err := supabase.NewClient(supabaseUrl, supabaseKey, nil)
	if err != nil {
		sugar.Fatalw("Error initializing Supabase client", "error", err)
	}
	sugar.Info("Connected to Supabase successfully!")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// ดึงค่า CORS_ORIGIN จาก .env (ถ้าไม่มีใช้ค่า default เป็นโดเมนของคุณ)
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "https://hora.garbagecollection.info"
	}

	// ดึงค่า JWT_SECRET จาก .env (หากไม่ได้ตั้งค่าไว้ จะใช้ค่า Default พร้อมแจ้งเตือนใน Log)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		sugar.Warn("ไม่ได้กำหนดค่า JWT_SECRET ใน .env ระบบจะใช้ค่าเริ่มต้นชั่วคราว (ไม่แนะนำสำหรับ Production)")
		jwtSecret = "YOUR_SUPER_SECRET_KEY_CHANGE_THIS"
	}

	// กำหนดค่า CookieSecure จากตัวแปรแวดล้อม COOKIE_SECURE หรือ APP_ENV=production
	cookieSecure := strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") ||
		strings.EqualFold(os.Getenv("APP_ENV"), "production")

	// ดึงค่า ADMIN_API_KEY จาก .env (หากไม่ได้ตั้งค่าแยกไว้ จะใช้ NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY ที่มีอยู่แล้วแทน)
	adminAPIKey := os.Getenv("ADMIN_API_KEY")
	if adminAPIKey == "" {
		adminAPIKey = os.Getenv("NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY")
	}

	// ดึงค่า HORA_API_URL จาก .env (หากไม่ได้ตั้งค่าไว้ จะใช้ค่า Default เป็น http://localhost:8083/api/v1/hora/prompt)
	horaAPIURL := os.Getenv("HORA_API_URL")
	if horaAPIURL == "" {
		horaAPIURL = "http://localhost:8083/api/v1/hora/prompt"
	}

	// ดึงค่า HORA_TRANSIT_API_URL จาก .env (หากไม่ได้ตั้งค่าไว้ จะแปลงจาก HORA_API_URL หรือใช้ Default)
	horaTransitAPIURL := os.Getenv("HORA_TRANSIT_API_URL")
	if horaTransitAPIURL == "" {
		if strings.HasSuffix(horaAPIURL, "/prompt") {
			horaTransitAPIURL = strings.TrimSuffix(horaAPIURL, "/prompt") + "/transitdasha_prompt"
		} else {
			horaTransitAPIURL = "http://localhost:8083/api/v1/hora/transitdasha_prompt"
		}
	}

	// ดึงค่าการเชื่อมต่อ AnythingLLM จาก .env
	anythingLLMAPIURL := strings.TrimSpace(os.Getenv("ANYTHINGLLM_API_URL"))
	if anythingLLMAPIURL == "" {
		anythingLLMAPIURL = "http://localhost:3001/api/v1"
	}
	anythingLLMAPIKey := strings.TrimSpace(os.Getenv("ANYTHINGLLM_API_KEY"))
	anythingLLMTimeout := 120 * time.Second
	if raw := strings.TrimSpace(os.Getenv("ANYTHINGLLM_TIMEOUT_SECONDS")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			anythingLLMTimeout = time.Duration(sec) * time.Second
		} else {
			sugar.Warnw("ค่า ANYTHINGLLM_TIMEOUT_SECONDS ไม่ถูกต้อง ระบบจะใช้ค่าเริ่มต้น 120 วินาที", "value", raw)
		}
	}

	if anythingLLMAPIKey == "" {
		sugar.Warn("ไม่ได้กำหนดค่า ANYTHINGLLM_API_KEY ใน .env ระบบจะคืนเฉพาะพิกัดดวงดาวโดยไม่มีคำทำนายจาก AnythingLLM")
	} else {
		maskedKey := "***"
		if len(anythingLLMAPIKey) > 8 {
			maskedKey = anythingLLMAPIKey[:4] + "..." + anythingLLMAPIKey[len(anythingLLMAPIKey)-4:]
		}
		sugar.Infow("โหลดค่าคอนฟิก AnythingLLM เรียบร้อย",
			"api_url", anythingLLMAPIURL,
			"api_key_masked", maskedKey,
			"timeout_seconds", int(anythingLLMTimeout.Seconds()),
		)
	}

	return &Config{
		SupabaseClient: client,
		Logger:         sugar,
		Port:           port,
		JWTSecret:      []byte(jwtSecret),
		CORSOrigin:     corsOrigin, // <-- กำหนดค่าที่นี่
		CookieSecure:   cookieSecure,
		AdminAPIKey:    adminAPIKey,
		HoraAPIURL:     horaAPIURL,
		HoraTransitAPIURL: horaTransitAPIURL,

		AnythingLLMAPIURL:  anythingLLMAPIURL,
		AnythingLLMAPIKey:  anythingLLMAPIKey,
		AnythingLLMTimeout: anythingLLMTimeout,
	}
}
