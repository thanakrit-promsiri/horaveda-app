package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/natefinch/lumberjack"
	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	SupabaseClient *supabase.Client
	Logger         *zap.SugaredLogger
	Port           string
	JWTSecret      []byte
	CORSOrigin     string // <-- เพิ่มฟิลด์นี้เข้าไป
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	// 1. ตั้งค่า Lumberjack สำหรับจัดการไฟล์ Log (ตัดไฟล์อัตโนมัติเมื่อครบ 10MB)
	fileWriter := &lumberjack.Logger{
		Filename:   "logs/app.log", // บันทึกไว้ที่โฟลเดอร์ logs ไฟล์ app.log
		MaxSize:    10,             // ขนาดสูงสุดหน่วยเป็น MB ก่อนจะตัดไฟล์ใหม่
		MaxBackups: 3,              // เก็บไฟล์เก่าสำรองไว้สูงสุด 3 ไฟล์
		MaxAge:     28,             // เก็บรักษาไฟล์เก่าไว้ไม่เกิน 28 วัน
		Compress:   true,           // บีบอัดไฟล์ log เก่าเป็น zip เพื่อประหยัดพื้นที่
	}

	// 2. ตั้งค่า Zap Core ให้พ่นออกทั้งหน้าจอ (Stdout) และลงไฟล์ (fileWriter) พร้อมกัน
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()) // ในไฟล์เก็บบันทึกแบบ JSON

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), zap.DebugLevel),
	)

	logger := zap.New(core, zap.AddCaller())
	sugar := logger.Sugar()

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

	return &Config{
		SupabaseClient: client,
		Logger:         sugar,
		Port:           port,
		JWTSecret:      []byte("YOUR_SUPER_SECRET_KEY_CHANGE_THIS"),
		CORSOrigin:     corsOrigin, // <-- กำหนดค่าที่นี่
	}
}