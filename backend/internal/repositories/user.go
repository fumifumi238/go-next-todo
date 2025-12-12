// Package repositories はデータベース操作を行うリポジトリを提供します。
package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"

	"go-next-todo/backend/internal/models"

	"golang.org/x/crypto/bcrypt" // パスワードのハッシュ化用
)

// UserRepository はデータベース操作を行うための構造体です。
type UserRepository struct {
	DB *sql.DB
}

// NewUserRepository は新しいUserRepositoryインスタンスを作成します。
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// HashPassword は与えられたパスワードをbcryptでハッシュ化します。
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// VerifyPassword はハッシュ化されたパスワードと平文のパスワードを比較します。
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrUserNotFound   = errors.New("user not found")
)

// Create は新しいユーザーをデータベースに挿入します。
func (r *UserRepository) Create(u *models.User) (*models.User, error) {
	query := "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)"
	result, err := r.DB.Exec(query, u.Username, u.Email, u.PasswordHash, u.Role)
	if err != nil {
		// MySQLの重複エントリーエラーコード1062をチェック
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			return nil, ErrDuplicateEmail // カスタムエラーを返す
		}
		log.Printf("Failed to insert user: %v", err)
		return nil, fmt.Errorf("could not insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}
	u.ID = int(id)

	return u, nil
}

// FindByEmail はメールアドレスでユーザーを検索します。
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	query := "SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE email = ?"
	var u models.User
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
			return nil, ErrUserNotFound
		}
		log.Printf("Failed to query user by email: %v", err)
		return nil, fmt.Errorf("could not query user: %w", err)
	}
	return &u, nil
}

// UpdatePassword はユーザーのパスワードを更新します。
func (r *UserRepository) UpdatePassword(userID uint, newHash string) error {
	res, err := r.DB.Exec("UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newHash, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}
