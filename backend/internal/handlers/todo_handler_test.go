// backend/cmd/api/todo_handlers_test.go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
	"go-next-todo/backend/testutil"
)

func TestCreateTodo_Success(t *testing.T) {
	_, r, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	token, err := testutil.LoginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	newTodo := models.Todo{
		Title:     "Test Todo",
		Completed: false,
	}
	jsonValue, _ := json.Marshal(newTodo)

	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")
	var createdTodo models.Todo
	err = json.Unmarshal(w.Body.Bytes(), &createdTodo)
	assert.NoError(t, err, "Response should be a valid JSON todo object")

	assert.NotZero(t, createdTodo.ID, "Expected a non-zero Todo ID")
	assert.Equal(t, newTodo.Title, createdTodo.Title, "Expected title to match")
	assert.False(t, createdTodo.Completed, "Expected completed to be false")
	assert.NotZero(t, createdTodo.CreatedAt, "Expected CreatedAt to be set")
	assert.NotZero(t, createdTodo.UpdatedAt, "Expected UpdatedAt to be set")
	assert.NotZero(t, createdTodo.UserID, "Expected UserID to be set")
}

func TestCreateTodo_AuthenticatedUserSuccess(t *testing.T) {
	db, r, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	token, err := testutil.LoginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	require.NotEmpty(t, token)

	newTodo := models.Todo{
		Title:     "Authenticated Test Todo",
		Completed: false,
	}
	jsonTodo, _ := json.Marshal(newTodo)

	req := httptest.NewRequest(http.MethodPost, "/api/todos", bytes.NewBuffer(jsonTodo))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)

	var createdTodo models.Todo
	err = json.Unmarshal(resp.Body.Bytes(), &createdTodo)
	require.NoError(t, err)
	require.NotZero(t, createdTodo.ID)
	require.Equal(t, newTodo.Title, createdTodo.Title)
	require.Equal(t, newTodo.Completed, createdTodo.Completed)
	require.NotZero(t, createdTodo.UserID)
	require.WithinDuration(t, time.Now(), createdTodo.CreatedAt, 5*time.Second)
	require.WithinDuration(t, time.Now(), createdTodo.UpdatedAt, 5*time.Second)

	var dbTodo models.Todo
	err = db.QueryRow("SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE id = ?", createdTodo.ID).Scan(
		&dbTodo.ID, &dbTodo.UserID, &dbTodo.Title, &dbTodo.Completed, &dbTodo.CreatedAt, &dbTodo.UpdatedAt,
	)
	require.NoError(t, err)
	require.Equal(t, createdTodo.ID, dbTodo.ID)
	require.Equal(t, createdTodo.UserID, dbTodo.UserID)
	require.Equal(t, createdTodo.Title, dbTodo.Title)
	require.Equal(t, createdTodo.Completed, dbTodo.Completed)
}

func TestGetTodosHandler_Authorization(t *testing.T) {
	// データベースとルーターをセットアップ
	_, router, _, userRepo := testutil.SetupTestDB(t) // todoRepo を使用

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	// これらのユーザーでログインしてトークンを取得
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123") // t を追加
	require.NoError(t, err)
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass") // t を追加
	require.NoError(t, err)

	// ノーマルユーザーのTODOを作成
	todo1 := testutil.CreateTestTodo(t, router, tokenNormal, "Normal User Todo 1", false)
	todo2 := testutil.CreateTestTodo(t, router, tokenNormal, "Normal User Todo 2", true)

	// 管理者ユーザーのTODOを作成 (テストのために作成)
	_ = testutil.CreateTestTodo(t, router, tokenAdmin, "Admin User Todo 1", false)

	// --- Test Case 1: ノーマルユーザーが自分のTODOをすべて取得できること ---
	t.Run("Normal user can get their own todos", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/todos", nil)
		req.Header.Set("Authorization", "Bearer "+tokenNormal)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var todos []*models.Todo
		err := json.Unmarshal(resp.Body.Bytes(), &todos)
		require.NoError(t, err)
		require.Len(t, todos, 2) // 自分のTODOが2つ
		require.Contains(t, []string{todos[0].Title, todos[1].Title}, todo1.Title)
		require.Contains(t, []string{todos[0].Title, todos[1].Title}, todo2.Title)
	})

	// --- Test Case 2: 認証されていないユーザーがTODOを取得できないこと ---
	t.Run("Unauthorized user cannot get todos", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/todos", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

func TestGetTodoByIDHandler_Authorization(t *testing.T) {
	_, router, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	// ログインしてトークンを取得
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass")
	require.NoError(t, err)

	// 別のユーザーを作成してそのトークンを取得
	_ = testutil.CreateTestUser(t, userRepo, "otheruser_for_id123", "other_for_id@example.com", "password123", "user")
	tokenOther, err := testutil.LoginAndGetToken(t, router, "other_for_id@example.com", "password123")
	require.NoError(t, err)

	todoNormalUser := testutil.CreateTestTodo(t, router, tokenNormal, "Normal User Todo", false)
	todoOtherUser := testutil.CreateTestTodo(t, router, tokenOther, "Other User Todo", false)

	// --- Test Case 1: 自分のTODOは取得できること ---
	t.Run("User can get their own todo by ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/todos/%d", todoNormalUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenNormal)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		var fetchedTodo models.Todo
		err := json.Unmarshal(resp.Body.Bytes(), &fetchedTodo)
		require.NoError(t, err)
		require.Equal(t, todoNormalUser.ID, fetchedTodo.ID)
	})

	// --- Test Case 2: 他人のTODOは取得できないこと ---
	t.Run("User cannot get another user's todo by ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenNormal) // NormalユーザーがOtherユーザーのTODOにアクセス
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})

	// --- Test Case 3: 管理者は他人のTODOも取得できること ---
	t.Run("Admin can get any user's todo by ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		var fetchedTodo models.Todo
		err := json.Unmarshal(resp.Body.Bytes(), &fetchedTodo)
		require.NoError(t, err)
		require.Equal(t, todoOtherUser.ID, fetchedTodo.ID)
	})
}

func TestUpdateTodoHandler_Authorization(t *testing.T) {
	_, router, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass")
	require.NoError(t, err)

	otherUser := testutil.CreateTestUser(t, userRepo, "otheruser_for_update123", "other_for_update@example.com", "password123", "user")
	tokenOther, err := testutil.LoginAndGetToken(t, router, "other_for_update@example.com", "password123")
	require.NoError(t, err)

	todoNormalUser := testutil.CreateTestTodo(t, router, tokenNormal, "Normal User Todo to Update", false)
	todoOtherUser := testutil.CreateTestTodo(t, router, tokenOther, "Other User Todo to Update", false)

	// --- Test Case 1: 自分のTODOは更新できること ---
	t.Run("User can update their own todo", func(t *testing.T) {
		updatePayload := `{"title": "Updated My Todo", "completed": true}`
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/todos/%d", todoNormalUser.ID), strings.NewReader(updatePayload))
		req.Header.Set("Authorization", "Bearer "+tokenNormal)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		var updatedTodo models.Todo
		err := json.Unmarshal(resp.Body.Bytes(), &updatedTodo)
		require.NoError(t, err)
		require.Equal(t, "Updated My Todo", updatedTodo.Title)
		require.True(t, updatedTodo.Completed)
		require.Equal(t, todoNormalUser.UserID, updatedTodo.UserID) // UserIDが変わらないことを確認
	})

	// --- Test Case 2: 他人のTODOは更新できないこと ---
	t.Run("User cannot update another user's todo", func(t *testing.T) {
		updatePayload := `{"title": "Try to Update Other Todo", "completed": true}`
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), strings.NewReader(updatePayload))
		req.Header.Set("Authorization", "Bearer "+tokenNormal) // NormalユーザーがOtherユーザーのTODOを更新しようとする
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})

	// --- Test Case 3: 管理者は他人のTODOも更新できること ---
	t.Run("Admin can update any user's todo", func(t *testing.T) {
		updatePayload := `{"title": "Admin Updated Other Todo", "completed": true}`
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), strings.NewReader(updatePayload))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		var updatedTodo models.Todo
		err := json.Unmarshal(resp.Body.Bytes(), &updatedTodo)
		require.NoError(t, err)
		require.Equal(t, "Admin Updated Other Todo", updatedTodo.Title)
		require.True(t, updatedTodo.Completed)
		require.Equal(t, otherUser.ID, updatedTodo.UserID) // 元の所有者が変わらないことを確認
	})
}

func TestDeleteTodoHandler_Authorization(t *testing.T) {
	_, router, todoRepo, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass")
	require.NoError(t, err)

	_ = testutil.CreateTestUser(t, userRepo, "otheruser_for_delete123", "other_for_delete@example.com", "password123", "user")
	tokenOther, err := testutil.LoginAndGetToken(t, router, "other_for_delete@example.com", "password123")
	require.NoError(t, err)

	todoNormalUser := testutil.CreateTestTodo(t, router, tokenNormal, "Normal User Todo to Delete", false)
	todoOtherUser := testutil.CreateTestTodo(t, router, tokenOther, "Other User Todo to Delete", false)

	// --- Test Case 1: 自分のTODOは削除できること ---
	t.Run("User can delete their own todo", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/todos/%d", todoNormalUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenNormal)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusNoContent, resp.Code)
		// 削除されたことを確認
		_, err := todoRepo.FindByID(todoNormalUser.ID)
		require.ErrorIs(t, err, repositories.ErrTodoNotFound)
	})

	// --- Test Case 2: 他人のTODOは削除できないこと ---
	t.Run("User cannot delete another user's todo", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenNormal) // NormalユーザーがOtherユーザーのTODOを削除しようとする
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
		// 削除されていないことを確認
		_, err := todoRepo.FindByID(todoOtherUser.ID)
		require.NoError(t, err)
	})

	// --- Test Case 3: 管理者は他人のTODOも削除できること ---
	t.Run("Admin can delete any user's todo", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/todos/%d", todoOtherUser.ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusNoContent, resp.Code)
		// 削除されたことを確認
		_, err := todoRepo.FindByID(todoOtherUser.ID)
		require.ErrorIs(t, err, repositories.ErrTodoNotFound)
	})
}

func TestFindAllTodosAdminHandler_Success(t *testing.T) {
	_, router, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass")
	require.NoError(t, err)

	// テスト用TODOを作成
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)
	_ = testutil.CreateTestTodo(t, router, tokenNormal, "Admin Test Todo 1", false)
	_ = testutil.CreateTestTodo(t, router, tokenNormal, "Admin Test Todo 2", true)

	// 管理者でリクエスト
	req, _ := http.NewRequest(http.MethodGet, "/api/admin/todos", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAdmin)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var todos []*models.Todo
	err = json.Unmarshal(resp.Body.Bytes(), &todos)
	require.NoError(t, err)
	// 作成したTODOが取得できるはず
	require.GreaterOrEqual(t, len(todos), 2)
}

func TestFindAllTodosAdminHandler_Unauthorized(t *testing.T) {
	_, router, _, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)

	// 一般ユーザーでリクエスト（403になるはず）
	req, _ := http.NewRequest(http.MethodGet, "/api/admin/todos", nil)
	req.Header.Set("Authorization", "Bearer "+tokenNormal)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code)
}

func TestDeleteTodoAdminHandler_Success(t *testing.T) {
	_, router, todoRepo, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	testutil.CreateTestUser(t, userRepo, "admin_user", "admin@example.com", "adminpass", "admin")
	tokenAdmin, err := testutil.LoginAndGetToken(t, router, "admin@example.com", "adminpass")
	require.NoError(t, err)

	// テスト用TODOを作成
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)
	todo := testutil.CreateTestTodo(t, router, tokenNormal, "Todo to Delete by Admin", false)

	// 管理者で削除リクエスト
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/todos/%d", todo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tokenAdmin)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)

	// 削除されたことを確認
	_, err = todoRepo.FindByID(todo.ID)
	require.ErrorIs(t, err, repositories.ErrTodoNotFound)
}

func TestDeleteTodoAdminHandler_Unauthorized(t *testing.T) {
	_, router, todoRepo, userRepo := testutil.SetupTestDB(t)

	testutil.CreateTestUser(t, userRepo, "normal_user", "normal_user@example.com", "password123", "user")
	tokenNormal, err := testutil.LoginAndGetToken(t, router, "normal_user@example.com", "password123")
	require.NoError(t, err)

	// テスト用TODOを作成
	todo := testutil.CreateTestTodo(t, router, tokenNormal, "Todo to Try Delete by Normal User", false)

	// 一般ユーザーで削除リクエスト（403になるはず）
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/todos/%d", todo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+tokenNormal)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code)

	// 削除されていないことを確認
	_, err = todoRepo.FindByID(todo.ID)
	require.NoError(t, err)
}
