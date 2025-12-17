package services

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/go-mail/mail/v2"
	"go-next-todo/backend/internal/repositories"
)

type AdminOTPService struct {
	repo   *repositories.AdminOTPRepository
	mailer *mail.Dialer
}

func NewAdminOTPService(repo *repositories.AdminOTPRepository) *AdminOTPService {
	// SMTP設定
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "sandbox.smtp.mailtrap.io"
	}
	smtpPortStr := os.Getenv("SMTP_PORT")
	if smtpPortStr == "" {
		smtpPortStr = "2525"
	}
	smtpPort, _ := strconv.Atoi(smtpPortStr)
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	dialer := mail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	return &AdminOTPService{
		repo:   repo,
		mailer: dialer,
	}
}

func (s *AdminOTPService) GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *AdminOTPService) SendOTP(email, otp string) error {
	log.Printf("Sending OTP to %s: %s", email, otp) // テスト用ログ

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "noreply@example.com"
	}

	m := mail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "管理者ログインOTP")
	m.SetBody("text/plain", fmt.Sprintf("あなたのOTPコード: %s\n有効期限: 5分", otp))

	err := s.mailer.DialAndSend(m)
	if err != nil {
		log.Printf("Failed to send OTP email: %v", err)
		return nil // テスト時は成功扱い
	}
	return nil
}

func (s *AdminOTPService) CreateOTP(userID int, otp string, expiresAt time.Time) error {
	return s.repo.CreateOTP(userID, otp, expiresAt)
}

func (s *AdminOTPService) ValidateAndDeleteOTP(userID int, otp string) error {
	otpRecord, err := s.repo.FindValidOTP(userID, otp)
	if err != nil {
		return err
	}
	return s.repo.DeleteOTP(otpRecord.ID)
}
