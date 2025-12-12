// Package repositories はデータベース操作を行うリポジトリを提供します。
package repositories

import (
	"database/sql"
	"errors"
	"log"

	"go-next-todo/backend/internal/models"
)

var ErrResetTokenNotFound = errors.New("reset token not found")

type ResetTokenRepository interface {
	Save(token *models.PasswordResetToken) error
	FindByToken(token string) (*models.PasswordResetToken, error)
	MarkUsed(id uint) error
	CleanupExpired() error
}

type MySQLResetTokenRepo struct {
	DB *sql.DB
}

func NewMySQLResetTokenRepo(db *sql.DB) *MySQLResetTokenRepo {
	return &MySQLResetTokenRepo{DB: db}
}

func (r *MySQLResetTokenRepo) Save(t *models.PasswordResetToken) error {
	_, err := r.DB.Exec(
		"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?, ?, ?)",
		t.UserID, t.Token, t.ExpiresAt,
	)
	return err
}

func (r *MySQLResetTokenRepo) FindByToken(token string) (*models.PasswordResetToken, error) {
	log.Println("[FindByToken] START ----------------------------")
	log.Println("[FindByToken] Searching token:", token)

	query := "SELECT id, user_id, token, expires_at, used_at FROM password_reset_tokens WHERE token = ?"
	log.Println("[FindByToken] SQL:", query)

	row := r.DB.QueryRow(query, token)

	var pr models.PasswordResetToken
	var usedAt sql.NullTime

	err := row.Scan(&pr.ID, &pr.UserID, &pr.Token, &pr.ExpiresAt, &usedAt)

	// --- Scan エラーの詳細表示 ---
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("[FindByToken] No rows found for token:", token)
			return nil, ErrResetTokenNotFound
		}

		log.Println("[FindByToken] SCAN ERROR:", err)
		return nil, err
	}

	// --- 取得されたデータのログ ---
	log.Println("[FindByToken] Found record:")
	log.Println("    ID        =", pr.ID)
	log.Println("    UserID    =", pr.UserID)
	log.Println("    Token     =", pr.Token)
	log.Println("    ExpiresAt =", pr.ExpiresAt)

	if usedAt.Valid {
		pr.UsedAt = &usedAt.Time
		log.Println("    UsedAt    =", pr.UsedAt)
	} else {
		log.Println("    UsedAt    = NULL")
	}

	log.Println("[FindByToken] END ------------------------------")

	return &pr, nil
}

func (r *MySQLResetTokenRepo) CleanupExpired() error {
	_, err := r.DB.Exec(`
		DELETE FROM password_reset_tokens
		WHERE used_at IS NOT NULL
		   OR expires_at < NOW()
	`)
	if err != nil {
		log.Println("[CleanupExpired] ERROR:", err)
		return err
	}

	log.Println("[CleanupExpired] Expired or used tokens cleaned")
	return nil
}

func (r *MySQLResetTokenRepo) MarkUsed(id uint) error {
	_, err := r.DB.Exec(
		"UPDATE password_reset_tokens SET used_at = NOW() WHERE id = ?",
		id,
	)
	return err
}
