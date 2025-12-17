package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
	"go-next-todo/backend/internal/services"
)

// UserHandler はユーザー関連のハンドラーを管理します。
type UserHandler struct {
	userService     *services.UserService
	jwtService      *services.JWTService
	adminOTPService *services.AdminOTPService
}

// NewUserHandler は新しいUserHandlerを作成します。
func NewUserHandler(userService *services.UserService, jwtService *services.JWTService, adminOTPService *services.AdminOTPService) *UserHandler {
	return &UserHandler{userService: userService, jwtService: jwtService, adminOTPService: adminOTPService}
}

// RegisterHandler はユーザー登録を処理します。
func (h *UserHandler) RegisterHandler(c *gin.Context) {
	var req models.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.userService.RegisterUser(req)
	if err != nil {
		if err == repositories.ErrDuplicateEmail {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// LoginHandler はユーザーログインを処理します。
func (h *UserHandler) LoginHandler(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.userService.AuthenticateUser(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.jwtService.GenerateToken(uint(user.ID), user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": user.ID, "role": user.Role})
}

// ProtectedHandler は認証テスト用のハンドラーです。
func (h *UserHandler) ProtectedHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found in token claims"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in token claims"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Access granted",
		"user_id": userID,
		"email":   userEmail,
		"role":    userRole,
	})
}

// ForgotPasswordHandler はパスワードリセットリクエストを処理します。
func (h *UserHandler) ForgotPasswordHandler(c *gin.Context) {
	var req models.UserForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	err := h.userService.ForgotPasswordUser(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password reset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset email sent"})
}

func (h *UserHandler) ResetPasswordHandler(c *gin.Context) {
	var req models.UserResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	token := c.Param("token")

	err := h.userService.ResetPasswordUser(token, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// FindAllUsersHandler はすべてのユーザーを取得します（管理者専用）。
func (h *UserHandler) FindAllUsersHandler(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in token claims"})
		return
	}

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin role required"})
		return
	}

	users, err := h.userService.FindAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// AdminLoginStep1 は管理者ログインのステップ1（パスワード認証 + OTP送信）を処理します。
func (h *UserHandler) AdminLoginStep1(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.userService.AuthenticateUser(req)
	if err != nil || user.Role != "admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	otp, err := h.adminOTPService.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP generation failed"})
		return
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	err = h.adminOTPService.CreateOTP(user.ID, otp, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP save failed"})
		return
	}

	err = h.adminOTPService.SendOTP(user.Email, otp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email send failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to email", "user_id": user.ID, "role": user.Role})
}

// AdminLoginStep2 は管理者ログインのステップ2（OTP検証 + JWT発行）を処理します。
func (h *UserHandler) AdminLoginStep2(c *gin.Context) {
	var req struct {
		UserID int    `json:"user_id" binding:"required"`
		OTP    string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	err := h.adminOTPService.ValidateAndDeleteOTP(req.UserID, req.OTP)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// JWT発行（emailは不要なので空）
	token, err := h.jwtService.GenerateToken(uint(req.UserID), "", "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
