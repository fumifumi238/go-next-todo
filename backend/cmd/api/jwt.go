// backend/cmd/api/jwt.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWT署名のためのシークレットキー
var jwtSecret []byte

// InitJWTSecret はJWTシークレットを初期化します。
func initJWTSecret() {

	// jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	jwtSecret = []byte(secret)
	// ここに出力されるログを確認してください
	log.Printf("DEBUG: initJWTSecret initialized with length %d bytes. First 10 bytes: %x", len(jwtSecret), jwtSecret[:min(10, len(jwtSecret))])

}

// Claims JWTクレームの構造体
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken はJWTトークンを生成します。
func GenerateToken(userID uint, email, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // 24時間有効
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}
	return tokenString, nil
}

// AuthMiddleware はJWTトークンを検証し、ユーザー情報をコンテキストに設定するミドルウェアです。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		// "Bearer " プレフィックスを削除
		if !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}
		tokenString = tokenString[len("Bearer "):]

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			log.Printf("JWT parse error: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			c.Set("user_id", int(claims.UserID)) // intにキャスト
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

// InitJWTSecretForTest はテスト用にjwtSecretを初期化します。
// main_test.go から呼び出されることを想定しています。
func InitJWTSecretForTest() {
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))

}
