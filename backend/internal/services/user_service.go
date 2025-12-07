package services

import (
	"fmt"
	"log"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
)

// UserService はユーザー関連のビジネスロジックを扱います。
type UserService struct {
	userRepo *repositories.UserRepository
}

// NewUserService は新しいUserServiceを作成します。
func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
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