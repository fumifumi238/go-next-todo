package services

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-next-todo/backend/internal/models"
)

// JWTService はJWTトークンの生成と検証を扱います。
type JWTService struct {
	secret []byte
}

// NewJWTService は新しいJWTServiceを作成します。
func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	return &JWTService{secret: []byte(secret)}
}

// GenerateToken はJWTトークンを生成します。
func (s *JWTService) GenerateToken(userID uint, email, role string) (string, error) {
	claims := &jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}
	return tokenString, nil
}

// ValidateToken はJWTトークンを検証し、クレームを返します。
func (s *JWTService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid user_id")
		}
		email, ok := claims["email"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid email")
		}
		role, ok := claims["role"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid role")
		}
		return &models.JWTClaims{
			UserID: uint(userIDFloat),
			Email:  email,
			Role:   role,
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}


// ValidatePasswordResetToken はリセットトークンを検証し user_id を返す
func (s *JWTService) ValidatePasswordResetToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// HMAC を期待
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// purpose 確認
		purpose, ok := claims["purpose"].(string)
		if !ok || purpose != "password_reset" {
			return 0, fmt.Errorf("invalid token purpose")
		}
		uidFloat, ok := claims["user_id"].(float64)
		if !ok {
			return 0, fmt.Errorf("invalid user_id in token")
		}
		return uint(uidFloat), nil
	}
	return 0, fmt.Errorf("invalid token")
}
