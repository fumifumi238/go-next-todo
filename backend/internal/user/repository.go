package user

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"

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

// 💡 追加: HashPassword は与えられたパスワードをbcryptでハッシュ化します。
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// 💡 追加: VerifyPassword はハッシュ化されたパスワードと平文のパスワードを比較します。
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
var ErrDuplicateEmail = errors.New("duplicate email")
// Create は新しいユーザーをデータベースに挿入します。
func (r *Repository) Create(u *User) (*User, error) {

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
	u.CreatedAt = time.Now() // DBで自動設定されるが、ここではテスト用に設定
	u.UpdatedAt = time.Now() // DBで自動設定されるが、ここではテスト用に設定

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
