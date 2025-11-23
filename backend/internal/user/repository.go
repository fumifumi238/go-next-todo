package user

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt" // パスワードのハッシュ化用
)

// Repository はデータベース操作を行うための構造体です。
type Repository struct {
	DB *sql.DB
}

// NewRepository は新しいRepositoryインスタンスを作成します。
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// Create は新しいユーザーをデータベースに挿入します。
func (r *Repository) Create(u *User) (*User, error) {
	// パスワードをハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	u.PasswordHash = string(hashedPassword)

	query := "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)"
	result, err := r.DB.Exec(query, u.Username, u.Email, u.PasswordHash, u.Role)
	if err != nil {
		log.Printf("Failed to insert user: %v", err)
		return nil, fmt.Errorf("could not insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}
	u.ID = int(id)
	u.CreatedAt = time.Now() // DBで自動設定されるが、ここではテスト用に設定
	u.UpdatedAt = time.Now() // DBで自動設定されるが、ここではテスト用に設定
	u.Password = "" // パスワード情報は返さない

	return u, nil
}

// FindByEmail はメールアドレスでユーザーを検索します。
func (r *Repository) FindByEmail(email string) (*User, error) {
	query := "SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE email = ?"
	var u User
	err := r.DB.QueryRow(query, email).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found with email %s", email)
		}
		log.Printf("Failed to query user by email: %v", err)
		return nil, fmt.Errorf("could not query user: %w", err)
	}
	return &u, nil
}

// VerifyPassword はプレーンテキストのパスワードとハッシュ化されたパスワードを比較します。
func (r *Repository) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
