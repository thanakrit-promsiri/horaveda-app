package repository

import (
	"encoding/json"
	"errors"
	"fmt"

	"horaveda-app/internal/domain"

	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/supabase-go"
	"go.uber.org/zap"
)

type AuthRepository interface {
	FindSerialKey(serialKey string) (*domain.SerialKey, error)
	CreateUser(username string) (*domain.User, error)
	CreateSupabaseAuthUser(email, password, username string) error
	UpdateSerialKeyStatus(keyID int, userID int) error
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

	json.Unmarshal(data, &keys)
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

	json.Unmarshal(userData, &createdUsers)
	if len(createdUsers) == 0 {
		r.logger.Errorw("Failed to parse created user data", "username", username)
		return nil, errors.New("ไม่สามารถสร้างบัญชีผู้ใช้ได้")
	}

	r.logger.Infow("Successfully created user in database", "username", username, "user_id", createdUsers[0].ID)
	return &createdUsers[0], nil
}

func (r *authRepository) CreateSupabaseAuthUser(email, password, username string) error {
	emailConfirm := true
	_, err := r.client.Auth.AdminCreateUser(types.AdminCreateUserRequest{
		Email:        email,
		Password:     &password,
		EmailConfirm: emailConfirm,
		UserMetadata: map[string]interface{}{
			"username": username,
		},
	})
	if err != nil {
		r.logger.Errorw("Failed to create auth user in Supabase", "email", email, "error", err)
		return errors.New("ไม่สามารถสร้าง Auth User ได้: " + err.Error())
	}

	r.logger.Infow("Successfully created Supabase auth user", "email", email)
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

func (r *authRepository) GetAllSerialKeys() ([]map[string]interface{}, error) {
	data, _, err := r.client.From("serial_keys").Select("*", "exact", false).Execute()
	if err != nil {
		r.logger.Errorw("Failed to fetch all serial keys", "error", err)
		return nil, err
	}

	var keys []map[string]interface{}
	json.Unmarshal(data, &keys)
	
	r.logger.Infow("Successfully fetched all serial keys", "count", len(keys))
	return keys, nil
}