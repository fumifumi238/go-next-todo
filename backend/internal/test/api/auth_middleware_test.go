package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-next-todo/backend/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	_, r, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "testuser", "test@example.com", "password123", "user")
	token, err := testutil.LoginAndGetToken(t, r, "test@example.com", "password123")
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Access granted", response["message"])
	assert.NotZero(t, response["user_id"]) // user_idが存在することを確認
	assert.Equal(t, "test@example.com", response["email"])
	assert.Equal(t, "user", response["role"])
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	_, r, _, _ := testutil.SetupTestDB(t)

	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token") // 不正なトークン
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid or expired token")
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	_, r, _, _ := testutil.SetupTestDB(t)

	req, _ := http.NewRequest("GET", "/api/protected", nil) // トークンなし
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Authorization header required")
}
