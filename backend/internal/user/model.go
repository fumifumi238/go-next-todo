package user

import "time"

// User はユーザーのデータベース構造体を表します。
// JSONタグ: クライアントとの通信用
// bindingタグ: Ginでのリクエストバリデーション用
type User struct {
	ID        int       `json:"id,omitempty"`
	Username  string    `json:"username" binding:"required"`
	Email     string    `json:"email" binding:"required,email"` // "email"バリデーションを追加
	PasswordHash string `json:"-"` // JSON出力しない。DBに保存するハッシュ化されたパスワード
	Role      string    `json:"role"` // "user" または "admin"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"` // 生パスワード
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"` // 生パスワード
}
