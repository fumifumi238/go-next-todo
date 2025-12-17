package models

import "time"

type AdminOTP struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	OTP       string    `json:"otp"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
