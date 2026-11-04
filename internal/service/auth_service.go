package service

import (
	"fmt"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/internal/repository"
	"horaveda-app/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type AuthService struct {
	Repo      repository.AuthRepository
	Logger    *zap.SugaredLogger
	JWTSecret []byte
}

func NewAuthService(repo repository.AuthRepository, log *zap.SugaredLogger, jwtSecret []byte) *AuthService {
	return &AuthService{
		Repo:      repo,
		Logger:    log,
		JWTSecret: jwtSecret,
	}
}

func (s *AuthService) RegisterWithSerial(username, serialKey string) (string, error) {
	s.Logger.Infow("กำลังค้นหา Serial Key ในฐานข้อมูล", "input_serial", serialKey)

	// 1. ค้นหา Serial Key ผ่าน Repository
	keyData, err := s.Repo.FindSerialKey(serialKey)
	if err != nil {
		return "", response.NewNotFoundError("ไม่พบรหัสซีเรียลนี้ในระบบ", err)
	}

	if keyData.IsUsed {
		return "", response.NewBadRequestError("รหัสซีเรียลนี้ถูกใช้งานไปแล้ว", nil)
	}

	// 2. สร้าง User ในตาราง public.users
	user, err := s.Repo.CreateUser(username)
	if err != nil {
		return "", response.NewBadRequestError("ชื่อผู้ใช้นี้ถูกใช้งานแล้ว กรุณาใช้ชื่ออื่น", err)
	}

	// Helper สำหรับ Rollback ลบข้อมูล User ในตาราง public.users
	rollbackUser := func() {
		if rbErr := s.Repo.DeleteUser(user.ID); rbErr != nil {
			s.Logger.Errorw("Rollback failed: could not delete user from public.users", "user_id", user.ID, "error", rbErr)
		}
	}

	// 3. สร้าง User ใน Supabase Auth (auth.users)
	email := username + "@garbagecollection.info"
	authUserID, err := s.Repo.CreateSupabaseAuthUser(email, serialKey, username)
	if err != nil {
		s.Logger.Errorw("Failed to create Supabase auth user, initiating rollback", "error", err, "user_id", user.ID)
		rollbackUser()
		return "", response.NewInternalServerError("ไม่สามารถสร้างบัญชีผู้ใช้ได้", err)
	}
	s.Logger.Infow("สร้าง Auth User ใน Supabase สำเร็จ", "email", email, "auth_user_id", authUserID)

	// Helper สำหรับ Rollback ลบบัญชีใน Supabase Auth (auth.users)
	rollbackAuthUser := func() {
		if authUserID != "" {
			if rbErr := s.Repo.DeleteSupabaseAuthUser(authUserID); rbErr != nil {
				s.Logger.Errorw("Rollback failed: could not delete Supabase auth user", "auth_user_id", authUserID, "error", rbErr)
			}
		}
	}

	// 4. อัปเดตสถานะ Serial Key ว่าถูกใช้งานแล้ว
	err = s.Repo.UpdateSerialKeyStatus(keyData.ID, user.ID)
	if err != nil {
		s.Logger.Errorw("Failed to update serial key status, initiating rollback", "error", err, "key_id", keyData.ID, "user_id", user.ID)
		rollbackAuthUser()
		rollbackUser()
		return "", response.NewInternalServerError("เกิดข้อผิดพลาดในการอัปเดตสถานะคีย์", err)
	}

	// 5. สร้าง JWT Token
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": username,
		"email":    email,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.JWTSecret)
	if err != nil {
		s.Logger.Errorw("Failed to sign JWT token, initiating rollback", "error", err, "user_id", user.ID)
		if rbErr := s.Repo.RollbackSerialKeyStatus(keyData.ID); rbErr != nil {
			s.Logger.Errorw("Rollback failed: could not revert serial key status", "key_id", keyData.ID, "error", rbErr)
		}
		rollbackAuthUser()
		rollbackUser()
		return "", response.NewInternalServerError("ไม่สามารถสร้าง Token ได้", err)
	}

	return tokenString, nil
}

// RevokeUserAndSerial ถอนการใช้งาน Serial Key ออกจากผู้ใช้ และลบข้อมูลผู้ใช้ทั้งในตาราง public.users และ Supabase Auth (auth.users)
func (s *AuthService) RevokeUserAndSerial(username string) (*domain.RevokeUserResponse, error) {
	s.Logger.Infow("กำลังเริ่มต้นกระบวนการถอนสิทธิ์ Serial Key และลบบัญชีผู้ใช้", "username", username)

	// 1. ค้นหาผู้ใช้ในตาราง public.users ด้วย username
	user, err := s.Repo.FindUserByUsername(username)
	if err != nil {
		return nil, response.NewNotFoundError("ไม่พบชื่อผู้ใช้นี้ในระบบ", err)
	}

	// 2. ถอนสิทธิ์การใช้งาน Serial Key ทั้งหมดที่ผูกกับ user.ID (ตั้งค่า is_used = false, user_id = nil)
	// ต้องทำก่อนลบแถวใน public.users เพื่อไม่ให้ติด Foreign Key Constraint
	revokedKeyRecords, err := s.Repo.RevokeSerialKeysByUserID(user.ID)
	if err != nil {
		s.Logger.Errorw("Failed to revoke serial keys for user", "username", username, "user_id", user.ID, "error", err)
		return nil, response.NewInternalServerError("ไม่สามารถถอนสิทธิ์การใช้งานรหัสซีเรียลได้", err)
	}

	revokedKeyCodes := make([]string, 0, len(revokedKeyRecords))
	for _, k := range revokedKeyRecords {
		revokedKeyCodes = append(revokedKeyCodes, k.KeyCode)
	}

	// Helper สำหรับผูก Serial Key กลับคืนให้ผู้ใช้ (Saga Rollback กรณีลบข้อมูลผู้ใช้ล้มเหลว)
	restoreRevokedKeys := func() {
		for _, k := range revokedKeyRecords {
			if rbErr := s.Repo.UpdateSerialKeyStatus(k.ID, user.ID); rbErr != nil {
				s.Logger.Errorw("Rollback failed: could not re-bind serial key to user", "key_id", k.ID, "user_id", user.ID, "error", rbErr)
			}
		}
	}

	// 3. ลบผู้ใช้จากตาราง public.users
	if err := s.Repo.DeleteUser(user.ID); err != nil {
		s.Logger.Errorw("Failed to delete user from public.users, rolling back serial keys", "username", username, "user_id", user.ID, "error", err)
		restoreRevokedKeys()
		return nil, response.NewInternalServerError("ไม่สามารถลบข้อมูลผู้ใช้ออกจากฐานข้อมูลได้", err)
	}

	// 4. ลบบัญชีผู้ใช้บน Supabase Auth (auth.users) เช่น testuser01@garbagecollection.info
	email := username + "@garbagecollection.info"
	if err := s.Repo.DeleteSupabaseAuthUserByEmail(email); err != nil {
		s.Logger.Errorw("Failed to delete user from Supabase Auth, rolling back public.users and serial_keys", "email", email, "error", err)
		if recreatedUser, rbErr := s.Repo.CreateUser(username); rbErr == nil && recreatedUser != nil {
			for _, k := range revokedKeyRecords {
				_ = s.Repo.UpdateSerialKeyStatus(k.ID, recreatedUser.ID)
			}
		}
		return nil, response.NewInternalServerError("ไม่สามารถลบบัญชีผู้ใช้บน Supabase Auth ได้", err)
	}

	s.Logger.Infow("ถอนสิทธิ์ Serial Key และลบบัญชีผู้ใช้สำเร็จ", "username", username, "email", email, "revoked_keys", revokedKeyCodes)

	return &domain.RevokeUserResponse{
		Username:      username,
		SupabaseEmail: email,
		RevokedKeys:   revokedKeyCodes,
	}, nil
}

func (s *AuthService) GetAllKeys() ([]map[string]interface{}, error) {
	keys, err := s.Repo.GetAllSerialKeys()
	if err != nil {
		return nil, response.NewInternalServerError("ไม่สามารถดึงข้อมูลคีย์ทั้งหมดได้", err)
	}
	return keys, nil
}

func (s *AuthService) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.JWTSecret, nil
	})

	if err != nil {
		return nil, response.NewUnauthorizedError("Invalid token", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, response.NewUnauthorizedError("Invalid token", nil)
}
