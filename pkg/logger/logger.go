package logger

import (
	"os" // เพิ่ม import os สำหรับใช้งาน os.Stdout

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger() *zap.SugaredLogger {
	// ตั้งค่า Log Rotation ด้วย lumberjack
	w := &lumberjack.Logger{
		Filename:   "logs/app.log", // ตำแหน่งไฟล์ Log
		MaxSize:    100,            // ขนาดไฟล์สูงสุดก่อนจะทำการ Rotate (หน่วยเป็น Megabytes)
		MaxBackups: 3,              // จำนวนไฟล์ Log เก่าที่ต้องการเก็บรักษาไว้
		MaxAge:     28,             // จำนวนวันสูงสุดที่จะเก็บไฟล์ Log เก่าไว้ (หน่วยเป็นวัน)
		Compress:   true,           // ต้องการให้บีบอัดไฟล์เก่าเป็น .gz หรือไม่ (true = บีบอัด)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	coreEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// ส่ง Log ออกทั้งไปที่ Log Rotation (ไฟล์) และออกหน้าจอ Console พร้อมกัน
	core := zapcore.NewTee(
		zapcore.NewCore(coreEncoder, zapcore.AddSync(w), zap.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), zap.InfoLevel),
	)

	baseLogger := zap.New(core, zap.AddCaller())
	return baseLogger.Sugar()
}