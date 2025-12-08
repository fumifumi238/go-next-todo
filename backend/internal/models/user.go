package models

import "time"

// User はユーザーのデータベース構造体を表します。
// JSONタグ: クライアントとの通信用
// bindingタグ: Ginでのリクエストバリデーション用
type User struct {
	ID           int       `json:"id,omitempty"`
	Username     string    `json:"username" binding:"required,min=8"`        // 8文字以上
	Email        string    `json:"email" binding:"required,email"`           // email形式
	PasswordHash string    `json:"-"`                                        // JSONに出さない
	Role         string    `json:"role" binding:"required,oneof=user admin"` // user または admin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRegisterRequest struct {
	Username string `json:"username" binding:"required,min=8"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"` // 生パスワード
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"` // 生パスワード
}

type UserForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetToken struct {
	ID        uint       `json:"id"`
	UserID    uint       `json:"user_id"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role" binding:"required,oneof=user admin"`
}
