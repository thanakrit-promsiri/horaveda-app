package config

import (
	"os"
	"strings"

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

	return &Config{
		SupabaseClient: client,
		Logger:         sugar,
		Port:           port,
		JWTSecret:      []byte(jwtSecret),
		CORSOrigin:     corsOrigin, // <-- กำหนดค่าที่นี่
		CookieSecure:   cookieSecure,
	}
}