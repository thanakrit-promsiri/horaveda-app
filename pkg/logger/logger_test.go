package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	log := NewLogger()
	if log == nil {
		t.Fatal("expected non-nil SugaredLogger")
	}
	log.Info("test logger initialized")
}

func TestDailyRotateWriter_DayTransition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-rotate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	day1, _ := time.Parse("2006-01-02", "2026-11-06")
	day2, _ := time.Parse("2006-01-02", "2026-11-07")
	simulatedTime := day1

	w := NewDailyRotateWriter(tmpDir, "testapp", 10, 5, 7, false)
	w.SetNowFunc(func() time.Time {
		return simulatedTime
	})
	defer w.Close()

	// 1. เขียน Log ในวันแรก (Day 1)
	_, err = w.Write([]byte("message from day 1\n"))
	if err != nil {
		t.Fatalf("failed to write day 1 log: %v", err)
	}
	_ = w.Sync()

	day1File := filepath.Join(tmpDir, "testapp-2026-11-06.log")
	b1, err := os.ReadFile(day1File)
	if err != nil {
		t.Fatalf("expected day 1 log file to exist at %s: %v", day1File, err)
	}
	if !strings.Contains(string(b1), "message from day 1") {
		t.Errorf("day 1 log content mismatch: %s", string(b1))
	}

	// 2. จำลองเวลาข้ามเข้าสู่วันที่ 2 (Day 2)
	simulatedTime = day2
	_, err = w.Write([]byte("message from day 2\n"))
	if err != nil {
		t.Fatalf("failed to write day 2 log: %v", err)
	}
	_ = w.Sync()

	day2File := filepath.Join(tmpDir, "testapp-2026-11-07.log")
	b2, err := os.ReadFile(day2File)
	if err != nil {
		t.Fatalf("expected day 2 log file to exist at %s: %v", day2File, err)
	}
	if !strings.Contains(string(b2), "message from day 2") {
		t.Errorf("day 2 log content mismatch: %s", string(b2))
	}

	// ตรวจสอบว่า Day 1 ยังคงอยู่และไม่ถูกเขียนทับ
	b1Check, _ := os.ReadFile(day1File)
	if strings.Contains(string(b1Check), "message from day 2") {
		t.Errorf("day 1 file should not contain day 2 log")
	}

	// ตรวจสอบว่า testapp.log มี log ของวันปัจจุบัน
	activeFile := filepath.Join(tmpDir, "testapp.log")
	bActive, err := os.ReadFile(activeFile)
	if err != nil {
		t.Fatalf("expected active log file to exist: %v", err)
	}
	if !strings.Contains(string(bActive), "message from day 2") {
		t.Errorf("active log file should contain day 2 log: %s", string(bActive))
	}
}

func TestDailyRotateWriter_CleanOldLogs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-clean-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	now := time.Now()
	w := NewDailyRotateWriter(tmpDir, "app", 10, 5, 2, false)
	defer w.Close()

	// สร้างไฟล์ log จำลอง 3 ไฟล์: ปัจจุบัน, 1 วันก่อน, 5 วันก่อน
	oldFile := filepath.Join(tmpDir, "app-2026-10-01.log")
	recentFile := filepath.Join(tmpDir, "app-2026-11-05.log")
	_ = os.WriteFile(oldFile, []byte("old log"), 0644)
	_ = os.WriteFile(recentFile, []byte("recent log"), 0644)

	// แก้ไข modification time ของไฟล์เก่าให้เป็น 5 วันก่อน
	fiveDaysAgo := now.AddDate(0, 0, -5)
	_ = os.Chtimes(oldFile, fiveDaysAgo, fiveDaysAgo)

	// เรียก cleanOldLogs
	w.cleanOldLogs()

	// ตรวจสอบว่าไฟล์ที่เก่ากว่า 2 วันถูกลบออกไป
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old log file to be deleted")
	}
	// ไฟล์ที่ยังไม่เกินกำหนดต้องยังคงอยู่
	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Errorf("expected recent log file to remain")
	}
}

func TestResolveLogDir(t *testing.T) {
	// Path สัมบูรณ์ต้องคืนค่าเดิม
	abs := filepath.Join(os.TempDir(), "logs")
	if got := resolveLogDir(abs); got != abs {
		t.Errorf("expected %s, got %s", abs, got)
	}

	// Path สัมพัทธ์ต้อง resolve ได้ folder logs
	rel := resolveLogDir("logs")
	if !strings.HasSuffix(rel, "logs") {
		t.Errorf("expected path ending with 'logs', got %s", rel)
	}
}
