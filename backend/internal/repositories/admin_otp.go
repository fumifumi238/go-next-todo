package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"go-next-todo/backend/internal/models"
)

type AdminOTPRepository struct {
	DB DataStore
}

func NewAdminOTPRepository(db DataStore) *AdminOTPRepository {
	return &AdminOTPRepository{DB: db}
}

func (r *AdminOTPRepository) CreateOTP(userID int, otp string, expiresAt time.Time) error {
	query := "INSERT INTO admin_otps (user_id, otp, expires_at, created_at) VALUES (?, ?, ?, ?)"
	_, err := r.DB.Exec(query, userID, otp, expiresAt, time.Now())
	if err != nil {
		log.Printf("Failed to insert admin OTP: %v", err)
		return fmt.Errorf("could not insert admin OTP: %w", err)
	}
	return nil
}

func (r *AdminOTPRepository) FindValidOTP(userID int, otp string) (*models.AdminOTP, error) {
	query := "SELECT id, user_id, otp, expires_at, created_at FROM admin_otps WHERE user_id = ? AND otp = ? AND expires_at > ?"
	var o models.AdminOTP
	err := r.DB.QueryRow(query, userID, otp, time.Now()).Scan(
		&o.ID,
		&o.UserID,
		&o.OTP,
		&o.ExpiresAt,
		&o.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired OTP")
		}
		log.Printf("Failed to query admin OTP: %v", err)
		return nil, fmt.Errorf("could not query admin OTP: %w", err)
	}
	return &o, nil
}

func (r *AdminOTPRepository) DeleteOTP(id int) error {
	query := "DELETE FROM admin_otps WHERE id = ?"
	_, err := r.DB.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete admin OTP: %v", err)
		return fmt.Errorf("could not delete admin OTP: %w", err)
	}
	return nil
}

func (r *AdminOTPRepository) CleanupExpiredOTPs() error {
	query := "DELETE FROM admin_otps WHERE expires_at < ?"
	_, err := r.DB.Exec(query, time.Now())
	if err != nil {
		log.Printf("Failed to cleanup expired admin OTPs: %v", err)
		return fmt.Errorf("could not cleanup expired admin OTPs: %w", err)
	}
	return nil
}
