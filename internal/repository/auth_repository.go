package repository

import (
	"encoding/json"
	"errors"
	"fmt"

	"horaveda-app/internal/domain"

	"github.com/google/uuid"
	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
)

type AuthRepository interface {
	FindSerialKey(serialKey string) (*domain.SerialKey, error)
	CreateUser(username string) (*domain.User, error)
	DeleteUser(userID int) error
	CreateSupabaseAuthUser(email, password, username string) (string, error)
	DeleteSupabaseAuthUser(authUserID string) error
	UpdateSerialKeyStatus(keyID int, userID int) error
	RollbackSerialKeyStatus(keyID int) error
	GetAllSerialKeys() ([]map[string]interface{}, error)
}

type authRepository struct {
	client *supabase.Client
	logger *zap.SugaredLogger
}

func NewAuthRepository(client *supabase.Client, logger *zap.SugaredLogger) AuthRepository {
	return &authRepository{client: client, logger: logger}
}

func (r *authRepository) FindSerialKey(serialKey string) (*domain.SerialKey, error) {
	var keys []domain.SerialKey
	data, _, err := r.client.From("serial_keys").Select("*", "exact", false).Eq("key_code", serialKey).Execute()
	if err != nil {
		r.logger.Errorw("Database query error (serial_keys)", "error", err)
		return nil, err
	}

	if err := json.Unmarshal(data, &keys); err != nil {
		r.logger.Errorw("Failed to parse serial keys data", "error", err)
		return nil, errors.New("เกิดข้อผิดพลาดในการแปลงข้อมูลรหัสซีเรียล")
	}
	if len(keys) == 0 {
		r.logger.Warnw("Serial key not found", "key_code", serialKey)
		return nil, errors.New("รหัสซีเรียลไม่ถูกต้อง")
	}

	r.logger.Infow("Successfully found serial key", "key_code", serialKey)
	return &keys[0], nil
}

func (r *authRepository) CreateUser(username string) (*domain.User, error) {
	var createdUsers []domain.User
	userData, _, err := r.client.From("users").Insert(map[string]interface{}{
		"username": username,
	}, false, "", "", "").Execute()

	if err != nil {
		r.logger.Errorw("Failed to insert user", "username", username, "error", err)
		return nil, errors.New("ชื่อผู้ใช้นี้ถูกใช้งานแล้ว กรุณาใช้ชื่ออื่น")
	}

	if err := json.Unmarshal(userData, &createdUsers); err != nil || len(createdUsers) == 0 {
		r.logger.Errorw("Failed to parse created user data", "username", username, "error", err)
		return nil, errors.New("ไม่สามารถสร้างบัญชีผู้ใช้ได้")
	}

	r.logger.Infow("Successfully created user in database", "username", username, "user_id", createdUsers[0].ID)
	return &createdUsers[0], nil
}

// DeleteUser ลบข้อมูลผู้ใช้จากตาราง public.users (ใช้สำหรับทำ Rollback กรณีขั้นตอนถัดไปล้มเหลว)
func (r *authRepository) DeleteUser(userID int) error {
	_, _, err := r.client.From("users").Delete("", "").Eq("id", fmt.Sprintf("%d", userID)).Execute()
	if err != nil {
		r.logger.Errorw("Failed to rollback (delete) user from database", "user_id", userID, "error", err)
		return errors.New("ไม่สามารถลบข้อมูลผู้ใช้เพื่อทำการ Rollback ได้")
	}

	r.logger.Infow("Successfully rolled back (deleted) user from database", "user_id", userID)
	return nil
}

func (r *authRepository) CreateSupabaseAuthUser(email, password, username string) (string, error) {
	emailConfirm := true
	resp, err := r.client.Auth.AdminCreateUser(types.AdminCreateUserRequest{
		Email:        email,
		Password:     &password,
		EmailConfirm: emailConfirm,
		UserMetadata: map[string]interface{}{
			"username": username,
		},
	})
	if err != nil {
		r.logger.Errorw("Failed to create auth user in Supabase", "email", email, "error", err)
		return "", errors.New("ไม่สามารถสร้าง Auth User ได้: " + err.Error())
	}

	authUserID := ""
	if resp != nil {
		authUserID = resp.ID.String()
	}

	r.logger.Infow("Successfully created Supabase auth user", "email", email, "auth_user_id", authUserID)
	return authUserID, nil
}

// DeleteSupabaseAuthUser ลบบัญชีผู้ใช้จากระบบ Supabase Auth (ใช้สำหรับทำ Rollback)
func (r *authRepository) DeleteSupabaseAuthUser(authUserID string) error {
	parsedUUID, err := uuid.Parse(authUserID)
	if err != nil {
		r.logger.Errorw("Invalid auth user UUID for rollback", "auth_user_id", authUserID, "error", err)
		return errors.New("รหัส Auth User ID ไม่ถูกต้องสำหรับการทำ Rollback")
	}

	if err := r.client.Auth.AdminDeleteUser(types.AdminDeleteUserRequest{UserID: parsedUUID}); err != nil {
		r.logger.Errorw("Failed to rollback (delete) Supabase auth user", "auth_user_id", authUserID, "error", err)
		return errors.New("ไม่สามารถลบ Auth User เพื่อทำการ Rollback ได้")
	}

	r.logger.Infow("Successfully rolled back (deleted) Supabase auth user", "auth_user_id", authUserID)
	return nil
}

func (r *authRepository) UpdateSerialKeyStatus(keyID int, userID int) error {
	_, _, err := r.client.From("serial_keys").Update(map[string]interface{}{
		"is_used": true,
		"user_id": userID,
	}, "", "").Eq("id", fmt.Sprintf("%d", keyID)).Execute()

	if err != nil {
		r.logger.Errorw("Failed to update serial key status", "key_id", keyID, "error", err)
		return errors.New("เกิดข้อผิดพลาดในการอัปเดตสถานะคีย์")
	}

	r.logger.Infow("Successfully updated serial key status to used", "key_id", keyID, "user_id", userID)
	return nil
}

// RollbackSerialKeyStatus คืนค่าสถานะของ Serial Key กลับเป็นยังไม่ถูกใช้งาน (ใช้สำหรับทำ Rollback)
func (r *authRepository) RollbackSerialKeyStatus(keyID int) error {
	_, _, err := r.client.From("serial_keys").Update(map[string]interface{}{
		"is_used": false,
		"user_id": nil,
	}, "", "").Eq("id", fmt.Sprintf("%d", keyID)).Execute()

	if err != nil {
		r.logger.Errorw("Failed to rollback serial key status", "key_id", keyID, "error", err)
		return errors.New("ไม่สามารถคืนค่าสถานะรหัสซีเรียลเพื่อทำการ Rollback ได้")
	}

	r.logger.Infow("Successfully rolled back serial key status", "key_id", keyID)
	return nil
}

func (r *authRepository) GetAllSerialKeys() ([]map[string]interface{}, error) {
	data, _, err := r.client.From("serial_keys").Select("*", "exact", false).Execute()
	if err != nil {
		r.logger.Errorw("Failed to fetch all serial keys", "error", err)
		return nil, err
	}

	var keys []map[string]interface{}
	if err := json.Unmarshal(data, &keys); err != nil {
		r.logger.Errorw("Failed to parse serial keys list", "error", err)
		return nil, err
	}

	r.logger.Infow("Successfully fetched all serial keys", "count", len(keys))
	return keys, nil
}
