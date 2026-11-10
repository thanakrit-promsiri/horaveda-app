package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// DailyRotateWriter จัดการหมุนเวียนไฟล์ Log รายวัน (Daily Rotation)
// - แยกไฟล์ Log รายวันตามรูปแบบ: logs/app-YYYY-MM-DD.log
// - เขียนควบคู่ลง logs/app.log สำหรับการดู Log ประจำวันล่าสุดอย่างสะดวก
// - หมุนเวียนไฟล์ใหม่ทันทีที่ข้ามวัน (Midnight 00:00:00 หรือเมื่อมีการบันทึกในวันใหม่)
// - หากใน 1 วัน ไฟล์มีขนาดเกิน MaxSize (เช่น 100 MB) lumberjack จะช่วยหมุนเวียนไฟล์ตามขนาดและบีบอัด .gz
// - ทำความสะอาดไฟล์ Log เก่าที่อายุเกิน MaxAge (เช่น 28 วัน) โดยอัตโนมัติ
type DailyRotateWriter struct {
	mu            sync.Mutex
	dir           string             // โฟลเดอร์เก็บ log (เช่น "logs")
	prefix        string             // คำนำหน้าชื่อไฟล์ (เช่น "app")
	maxSize       int                // ขนาดสูงสุดต่อไฟล์ (MB)
	maxBackups    int                // จำนวนไฟล์เก่าสูงสุด
	maxAge        int                // จำนวนวันสูงสุดที่เก็บรักษาไฟล์ (วัน)
	compress      bool               // บีบอัดไฟล์เก่า (.gz) หรือไม่
	currentDay    string             // วันที่ปัจจุบันรูปแบบ "2006-01-02"
	currentLogger *lumberjack.Logger // lumberjack instance ของวันนี้
	activeFile    *os.File           // logs/app.log ประจำวันล่าสุด
	nowFunc       func() time.Time   // สำหรับ mock เวลาใน unit test
	stopTimer     *time.Timer        // จับเวลาเที่ยงคืน
	closed        bool               // สถานะการปิด writer
}

// NewDailyRotateWriter สร้างตัวจัดการ Daily Rotate Log
func NewDailyRotateWriter(dir, prefix string, maxSize, maxBackups, maxAge int, compress bool) *DailyRotateWriter {
	if dir == "" {
		dir = resolveLogDir("logs")
	} else {
		dir = resolveLogDir(dir)
	}
	if prefix == "" {
		prefix = "app"
	}
	if maxSize <= 0 {
		maxSize = 100
	}
	if maxBackups <= 0 {
		maxBackups = 30
	}
	if maxAge <= 0 {
		maxAge = 28
	}

	w := &DailyRotateWriter{
		dir:        dir,
		prefix:     prefix,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}

	// เริ่มต้นตั้งค่าสำหรับวันนี้
	_ = w.rotateToDay(w.now().Format("2006-01-02"))
	w.scheduleNextMidnight()

	return w
}

func (w *DailyRotateWriter) now() time.Time {
	w.mu.Lock()
	fn := w.nowFunc
	w.mu.Unlock()
	if fn != nil {
		return fn()
	}
	return time.Now()
}

// SetNowFunc กำหนดฟังก์ชันเวลาสำหรับทดสอบ
func (w *DailyRotateWriter) SetNowFunc(fn func() time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.nowFunc = fn
}

// Write บันทึก Log ลงทั้งไฟล์ประจำวัน (app-YYYY-MM-DD.log) และไฟล์ปัจจุบัน (app.log)
func (w *DailyRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := w.now().Format("2006-01-02")
	if w.currentLogger == nil || today != w.currentDay {
		if err := w.rotateToDay(today); err != nil {
			return 0, err
		}
	}

	// 1. บันทึกลงไฟล์ประจำวัน logs/app-YYYY-MM-DD.log (ผ่าน lumberjack เพื่อคุมขนาด 100MB ภายในวัน)
	n, err = w.currentLogger.Write(p)

	// 2. บันทึกลง logs/app.log คู่ขนานกัน เพื่อให้ดู log ปัจจุบันได้ง่าย
	if w.activeFile != nil {
		_, _ = w.activeFile.Write(p)
	}

	return n, err
}

// Sync flush ข้อมูลลงดิสก์
func (w *DailyRotateWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.activeFile != nil {
		_ = w.activeFile.Sync()
	}
	return nil
}

// Close ปิดไฟล์และ lumberjack instance พร้อมหยุดตัวจับเวลา
func (w *DailyRotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	if w.stopTimer != nil {
		w.stopTimer.Stop()
		w.stopTimer = nil
	}
	if w.activeFile != nil {
		_ = w.activeFile.Close()
		w.activeFile = nil
	}
	if w.currentLogger != nil {
		return w.currentLogger.Close()
	}
	return nil
}

func (w *DailyRotateWriter) rotateToDay(today string) error {
	if err := os.MkdirAll(w.dir, 0755); err != nil {
		return err
	}

	if w.currentLogger != nil {
		_ = w.currentLogger.Close()
		w.currentLogger = nil
	}
	if w.activeFile != nil {
		_ = w.activeFile.Close()
		w.activeFile = nil
	}

	// ถ้าข้ามวัน ให้ truncate logs/app.log เริ่มต้นวันใหม่
	// แต่ถ้าวันเดียวกัน (เช่น เพิ่งเปิดแอป) ให้ append ต่อ
	flag := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if w.currentDay != "" && w.currentDay != today {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	activePath := filepath.Join(w.dir, w.prefix+".log")
	if af, err := os.OpenFile(activePath, flag, 0644); err == nil {
		w.activeFile = af
	}

	w.currentDay = today
	dailyPath := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, today))
	w.currentLogger = &lumberjack.Logger{
		Filename:   dailyPath,
		MaxSize:    w.maxSize,
		MaxBackups: w.maxBackups,
		MaxAge:     w.maxAge,
		Compress:   w.compress,
	}

	go w.cleanOldLogs()
	return nil
}

func (w *DailyRotateWriter) scheduleNextMidnight() {
	now := w.now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	duration := nextMidnight.Sub(now)

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	if w.stopTimer != nil {
		w.stopTimer.Stop()
	}
	w.stopTimer = time.AfterFunc(duration, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.closed {
			return
		}
		today := w.now().Format("2006-01-02")
		if today != w.currentDay {
			_ = w.rotateToDay(today)
		}
		w.scheduleNextMidnight()
	})
	w.mu.Unlock()
}

func (w *DailyRotateWriter) cleanOldLogs() {
	w.mu.Lock()
	maxAge := w.maxAge
	dir := w.dir
	prefix := w.prefix
	w.mu.Unlock()

	if maxAge <= 0 {
		return
	}
	cutoff := w.now().AddDate(0, 0, -maxAge)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix+"-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// resolveLogDir ค้นหาโฟลเดอร์ logs ที่สัมพันธ์กับ Project Root เสมอ
func resolveLogDir(defaultDir string) string {
	if filepath.IsAbs(defaultDir) {
		return defaultDir
	}
	dir, err := os.Getwd()
	if err == nil {
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return filepath.Join(dir, defaultDir)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return defaultDir
}

// NewLogger สร้าง *zap.SugaredLogger พร้อมระบบ Daily Log Rotation
func NewLogger() *zap.SugaredLogger {
	// ตั้งค่า Daily Log Rotation (หมุนเวียนไฟล์ตามวัน YYYY-MM-DD + เก็บย้อนหลัง 28 วัน)
	w := NewDailyRotateWriter("", "app", 100, 30, 28, true)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	coreEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// ส่ง Log ออกทั้งไปที่ Daily Log Rotation (ไฟล์) และออกหน้าจอ Console พร้อมกัน
	core := zapcore.NewTee(
		zapcore.NewCore(coreEncoder, zapcore.AddSync(w), zap.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), zap.InfoLevel),
	)

	baseLogger := zap.New(core, zap.AddCaller())
	return baseLogger.Sugar()
}
