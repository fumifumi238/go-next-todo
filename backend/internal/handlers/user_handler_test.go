// backend/internal/handlers/user_handlers_test.go
package handlers_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-next-todo/backend/testutil"

	"github.com/stretchr/testify/assert"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
)

func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func TestRegisterUser_Success(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	newUserData := map[string]string{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "newpassword",
	}
	jsonValue, _ := json.Marshal(newUserData)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")

	var responseUser models.User
	err := json.Unmarshal(w.Body.Bytes(), &responseUser)
	assert.NoError(t, err, "Response should be a valid JSON user object")
	assert.NotZero(t, responseUser.ID, "Expected a non-zero User ID")
	assert.Equal(t, "newuser", responseUser.Username, "Expected username to match")
	assert.Equal(t, "newuser@example.com", responseUser.Email, "Expected email to match")
	assert.Equal(t, "user", responseUser.Role, "Expected default role to be 'user'")
	assert.Empty(t, responseUser.PasswordHash, "Password hash should not be returned in response")
}

func TestRegisterUser_InvalidInput(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	invalidUserData := map[string]string{
		"username": "invaliduser",
		"email":    "invalid@example.com",
	}
	jsonValue, _ := json.Marshal(invalidUserData)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected HTTP Status Code 400 Bad Request")
	var response map[string]string

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request payload")
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	db, r, _, userRepo := testutil.SetupTestDB(t)
	defer db.Close()

	existingUser := models.User{
		Username:     "existing",
		Email:        "duplicate@example.com",
		PasswordHash: "hashedpass",
		Role:         "user",
	}
	_, err := userRepo.Create(&existingUser)
	assert.NoError(t, err)

	duplicateUserData := map[string]string{
		"username": "anotheruser",
		"email":    "duplicate@example.com",
		"password": "somepassword",
	}
	jsonValue, _ := json.Marshal(&duplicateUserData)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code, "Expected HTTP Status Code 409 Conflict for duplicate email")
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Username or email already exists", "Expected 'Username or email already exists' error")
}

func TestLoginUser_Success(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	loginCredentials := map[string]string{
		"email":    "normal_user@example.com",
		"password": "password123",
	}
	jsonValue, _ := json.Marshal(loginCredentials)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "token", "Expected response to contain a 'token'")
	token, ok := response["token"].(string)
	assert.True(t, ok, "Token should be a string")
	assert.NotEmpty(t, token, "Expected token to be non-empty")
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	loginCredentials := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}
	jsonValue, _ := json.Marshal(loginCredentials)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected HTTP Status Code 401 Unauthorized")
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid credentials", "Expected 'Invalid credentials' error")
}

func TestResetPassword_Success(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	// トークンを作成
	resetTokenRepo := repositories.NewMySQLResetTokenRepo(db)
	token, _ := generateResetToken()
	resetToken := &models.PasswordResetToken{
		UserID:    1,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := resetTokenRepo.Save(resetToken)
	assert.NoError(t, err)

	resetData := map[string]string{
		"password": "NewPassword123!",
	}
	jsonValue, _ := json.Marshal(resetData)

	req, _ := http.NewRequest("POST", "/api/reset-password/"+token, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Password reset successfully")
}

func TestForgotPassword_Success(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	forgotData := map[string]string{
		"email": "normal_user@example.com",
	}
	jsonValue, _ := json.Marshal(forgotData)

	req, _ := http.NewRequest("POST", "/api/forgot-password", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Password reset email sent")
}

func TestForgotPassword_InvalidEmail(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	forgotData := map[string]string{
		"email": "invalid-email",
	}
	jsonValue, _ := json.Marshal(forgotData)

	req, _ := http.NewRequest("POST", "/api/forgot-password", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected HTTP Status Code 400 Bad Request")
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request payload")
}
