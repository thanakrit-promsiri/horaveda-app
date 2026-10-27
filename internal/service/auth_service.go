package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"horaveda-app/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
)

type AuthService struct {
	Client    *supabase.Client
	Logger    *zap.SugaredLogger
	JWTSecret []byte
}

func NewAuthService(client *supabase.Client, logger *zap.SugaredLogger, jwtSecret []byte) *AuthService {
	return &AuthService{
		Client:    client,
		Logger:    logger,
		JWTSecret: jwtSecret,
	}
}

func (s *AuthService) RegisterWithSerial(username, serialKey string) (string, error) {
	// 1. ค้นหา Serial Key
	s.Logger.Infow("กำลังค้นหา Serial Key ในฐานข้อมูล", "input_serial", serialKey)
	var keys []model.SerialKey
	data, _, err := s.Client.From("serial_keys").Select("*", "exact", false).Eq("key_code", serialKey).Execute()
	if err != nil {
		s.Logger.Errorw("Database query error (serial_keys)", "error", err)
		return "", errors.New("server error")
	}

	json.Unmarshal(data, &keys)
	if len(keys) == 0 {
		return "", errors.New("รหัสซีเรียลไม่ถูกต้อง")
	}

	keyData := keys[0]
	if keyData.IsUsed {
		return "", errors.New("รหัสซีเรียลนี้ถูกใช้งานไปแล้ว")
	}

	// 2. สร้าง User ในตาราง public.users ปกติ
	var createdUsers []model.User
	userData, _, err := s.Client.From("users").Insert(map[string]interface{}{
		"username": username,
	}, false, "", "", "").Execute()

	if err != nil {
		s.Logger.Errorw("Failed to insert user", "username", username, "error", err)
		return "", errors.New("ชื่อผู้ใช้นี้ถูกใช้งานแล้ว กรุณาใช้ชื่ออื่น")
	}

	json.Unmarshal(userData, &createdUsers)
	if len(createdUsers) == 0 {
		return "", errors.New("ไม่สามารถสร้างบัญชีผู้ใช้ได้")
	}

	userID := createdUsers[0].ID

	// 3. สั่งสร้าง User ใน Supabase Auth (auth.users) ผ่าน Admin API
	email := username + "@garbagecollection.info"
	pwd := serialKey
	emailConfirm := true

	_, err = s.Client.Auth.AdminCreateUser(types.AdminCreateUserRequest{
		Email:        email,
		Password:     &pwd,
		EmailConfirm: emailConfirm,
		UserMetadata: map[string]interface{}{
			"username": username,
		},
	})

	if err != nil {
		s.Logger.Errorw("Failed to create auth user in Supabase", "email", email, "error", err)
		return "", errors.New("ไม่สามารถสร้าง Auth User ได้: " + err.Error())
	}
	s.Logger.Infow("สร้าง Auth User ใน Supabase สำเร็จ", "email", email)

	// 4. อัปเดตสถานะ Serial Key ว่าถูกใช้งานแล้วและผูก user_id
	_, _, err = s.Client.From("serial_keys").Update(map[string]interface{}{
		"is_used": true,
		"user_id": userID,
	}, "", "").Eq("id", fmt.Sprintf("%d", keyData.ID)).Execute()

	if err != nil {
		s.Logger.Errorw("Failed to update serial key status", "key_id", keyData.ID, "error", err)
		return "", errors.New("เกิดข้อผิดพลาดในการอัปเดตสถานะคีย์")
	}

	// 5. สร้าง JWT Token ของระบบเราเอง
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"email":    email,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.JWTSecret)
	if err != nil {
		s.Logger.Errorw("Failed to sign JWT token", "error", err)
		return "", errors.New("ไม่สามารถสร้าง Token ได้")
	}

	return tokenString, nil
}

func (s *AuthService) GetAllKeys() ([]map[string]interface{}, error) {
	data, _, err := s.Client.From("serial_keys").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, err
	}

	var keys []map[string]interface{}
	json.Unmarshal(data, &keys)
	return keys, nil
}

// VerifyToken ทำหน้าที่ตรวจสอบความถูกต้องของ JWT Token และคืนค่า claims ออกมา
func (s *AuthService) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	// 1. Parse และตรวจสอบ Token โดยใช้ JWTSecret ที่ตั้งไว้
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// ตรวจสอบ Signing Method ว่าถูกต้องตามที่ใช้เซ็น (HS256) หรือไม่
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// 2. ตรวจสอบว่า Token สมบูรณ์และใช้งานได้จริงหรือไม่ พร้อมแปลงเป็น MapClaims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
