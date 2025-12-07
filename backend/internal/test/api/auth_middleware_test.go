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
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	token, err := testutil.LoginAndGetToken(t, r, "normal_user@example.com", "password123")
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
	assert.Equal(t, float64(1), response["user_id"]) // user_idはfloat64でデコードされる
	assert.Equal(t, "normal_user@example.com", response["email"])
	assert.Equal(t, "user", response["role"])
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

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
	db, r, _, _ := testutil.SetupTestDB(t)
	defer db.Close()

	req, _ := http.NewRequest("GET", "/api/protected", nil) // トークンなし
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Authorization header required")
}
