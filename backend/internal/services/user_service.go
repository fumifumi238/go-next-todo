package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
)

// UserService はユーザー関連のビジネスロジックを扱います。
type UserService struct {
	userRepo       *repositories.UserRepository
	resetTokenRepo repositories.ResetTokenRepository
}

// NewUserService は新しいUserServiceを作成します。
func NewUserService(userRepo *repositories.UserRepository, resetTokenRepo repositories.ResetTokenRepository) *UserService {
	return &UserService{userRepo: userRepo, resetTokenRepo: resetTokenRepo}
}

// RegisterUser はユーザーを登録します。
func (s *UserService) RegisterUser(req models.UserRegisterRequest) (*models.User, error) {
	hashedPassword, err := repositories.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
	}

	createdUser, err := s.userRepo.Create(newUser)
	if err != nil {
		return nil, err
	}
	createdUser.PasswordHash = "" // レスポンスにパスワードを含めない
	return createdUser, nil
}

// AuthenticateUser はユーザーを認証し、成功したらユーザーを返します。
func (s *UserService) AuthenticateUser(req models.UserLoginRequest) (*models.User, error) {
	foundUser, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}

	if err := repositories.VerifyPassword(foundUser.PasswordHash, req.Password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	foundUser.PasswordHash = "" // レスポンスにパスワードを含めない
	return foundUser, nil
}

func (s *UserService) ForgotPasswordUser(email string) error {
	// 1. ユーザーが存在するか確認
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// メール存在しない → バレないように成功扱い
		log.Printf("email not found but returning OK: %s", email)
		return nil
	}

	// 2. パスワードリセット用のトークンを生成
	token, err := generateResetToken()
	if err != nil {
		log.Printf("Failed to generate reset token: %v", err)
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// 3. トークンをデータベースに保存（有効期限1時間）
	resetToken := &models.PasswordResetToken{
		UserID:    uint(user.ID),
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err = s.resetTokenRepo.Save(resetToken)
	if err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	// 4. フロントのリセットURLにトークンをセット

	resetURL := fmt.Sprintf("%s/reset-password/%s", frontendURL, token)

	// 5. メール送信
	err = s.sendPasswordResetEmail(email, resetURL)
	if err != nil {
		log.Printf("failed to send reset email: %v", err)
	}

	return nil
}

// generateResetToken はパスワードリセット用のランダムトークンを生成します。
func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ResetPasswordUser はトークンを使ってパスワードをリセットします。
func (s *UserService) ResetPasswordUser(token, newPassword string) error {
	// 1. トークンを検証
	resetToken, err := s.resetTokenRepo.FindByToken(token)

	if err != nil {
		return fmt.Errorf("invalid or expired token")
	}

	// 2. トークンが有効か確認
	if time.Now().After(resetToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	if resetToken.UsedAt != nil {
		return fmt.Errorf("token already used")
	}

	// 3. パスワードをハッシュ化
	hashedPassword, err := repositories.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. ユーザーのパスワードを更新
	err = s.userRepo.UpdatePassword(resetToken.UserID, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 5. トークンを使用済みにマーク
	err = s.resetTokenRepo.MarkUsed(resetToken.ID)
	if err != nil {
		log.Printf("Failed to mark token as used: %v", err)
		// 失敗しても続行
	}

	return nil
}

func (s *UserService) sendPasswordResetEmail(email, resetURL string) error {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	to := []string{email}

	smtpHost := "sandbox.smtp.mailtrap.io"
	smtpPort := "2525"

	// 件名と本文
	message := []byte(fmt.Sprintf(
		"Subject: パスワードリセット\r\n\r\n以下のURLからパスワードを再設定してください。\r\n%s",
		resetURL,
	))

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		// Mailtrap が無くてもテストできるように成功扱いにする
		log.Printf("Failed to send reset email: %v", err)
		return nil
	}

	return nil
}
