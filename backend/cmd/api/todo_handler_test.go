// backend/cmd/api/todo_handlers_test.go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-next-todo/backend/internal/todo"
)

func TestCreateTodo_Success(t *testing.T) {
	db, r, _, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	newTodo := todo.Todo{
		UserID:    1, // normal_userのIDは1
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
	var createdTodo todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &createdTodo)
	assert.NoError(t, err, "Response should be a valid JSON todo object")

	assert.NotZero(t, createdTodo.ID, "Expected a non-zero Todo ID")
	assert.Equal(t, newTodo.Title, createdTodo.Title, "Expected title to match")
	assert.False(t, createdTodo.Completed, "Expected completed to be false")
	assert.NotZero(t, createdTodo.CreatedAt, "Expected CreatedAt to be set")
	assert.NotZero(t, createdTodo.UpdatedAt, "Expected UpdatedAt to be set")
	assert.Equal(t, newTodo.UserID, createdTodo.UserID, "Expected UserID to be 1")
}

func TestCreateTodo_AuthenticatedUserSuccess(t *testing.T) {
	db, r, _, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	require.NotEmpty(t, token)

	newTodo := todo.Todo{
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

	var createdTodo todo.Todo
	err = json.Unmarshal(resp.Body.Bytes(), &createdTodo)
	require.NoError(t, err)
	require.NotZero(t, createdTodo.ID)
	require.Equal(t, newTodo.Title, createdTodo.Title)
	require.Equal(t, newTodo.Completed, createdTodo.Completed)
	require.Equal(t, 1, createdTodo.UserID) // normal_userのIDが1であることを確認
	require.WithinDuration(t, time.Now(), createdTodo.CreatedAt, 5*time.Second)
	require.WithinDuration(t, time.Now(), createdTodo.UpdatedAt, 5*time.Second)

	var dbTodo todo.Todo
	err = db.QueryRow("SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE id = ?", createdTodo.ID).Scan(
		&dbTodo.ID, &dbTodo.UserID, &dbTodo.Title, &dbTodo.Completed, &dbTodo.CreatedAt, &dbTodo.UpdatedAt,
	)
	require.NoError(t, err)
	require.Equal(t, createdTodo.ID, dbTodo.ID)
	require.Equal(t, createdTodo.UserID, dbTodo.UserID)
	require.Equal(t, createdTodo.Title, dbTodo.Title)
	require.Equal(t, createdTodo.Completed, dbTodo.Completed)
}

func TestGetTodos_Success(t *testing.T) {
	db, r, todoRepo, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	todo1 := &todo.Todo{UserID: 1, Title: "Test Todo 1", Completed: false}
	todo2 := &todo.Todo{UserID: 1, Title: "Test Todo 2", Completed: true}
	createdTodo1, err := todoRepo.Create(todo1)
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	createdTodo2, err := todoRepo.Create(todo2)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")
	var todos []todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &todos)
	assert.NoError(t, err, "Response should be a valid JSON array")
	assert.Len(t, todos, 2, "Expected 2 todos in the response")

	assert.Equal(t, createdTodo2.Title, todos[0].Title, "Expected 'Test Todo 2' to be the first todo")
	assert.Equal(t, createdTodo1.Title, todos[1].Title, "Expected 'Test Todo 1' to be the second todo")
}

func TestGetTodoByID_Success(t *testing.T) {
	db, r, todoRepo, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	newTodo := todo.Todo{Title: "Specific Todo", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Specific Todo", responseTodo.Title)
	assert.Equal(t, newTodo.UserID, responseTodo.UserID, "Expected UserID to match")
}

func TestGetTodoByID_NotFound(t *testing.T) {
	db, r, _, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/todos/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

func TestUpdateTodo_Success(t *testing.T) {
	db, r, todoRepo, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	originalTodo := todo.Todo{Title: "Original Todo", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&originalTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	time.Sleep(1 * time.Second)

	updatedData := map[string]interface{}{
		"title":     "Updated Todo",
		"completed": true,
		"user_id":   1,
	}
	jsonValue, _ := json.Marshal(updatedData)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/todos/%d", createdTodo.ID), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Updated Todo", responseTodo.Title)
	assert.True(t, responseTodo.Completed)
	assert.True(t, responseTodo.UpdatedAt.After(createdTodo.UpdatedAt), "UpdatedAt should be updated after the original CreatedAt")
	assert.Equal(t, originalTodo.UserID, responseTodo.UserID, "Expected UserID to remain the same")
}

func TestUpdateTodo_NotFound(t *testing.T) {
	db, r, _, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	updatedData := map[string]interface{}{
		"title":     "Non Existent Todo",
		"completed": true,
		"user_id":   1,
	}
	jsonValue, _ := json.Marshal(updatedData)

	req, _ := http.NewRequest("PUT", "/api/todos/99999", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

func TestDeleteTodo_Success(t *testing.T) {
	db, r, todoRepo, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	newTodo := todo.Todo{Title: "Todo to Delete", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err = todoRepo.FindByID(createdTodo.ID)
	assert.Error(t, err, "Expected an error as todo should be deleted")
	assert.True(t, errors.Is(err, todo.ErrTodoNotFound), "Expected ErrTodoNotFound after deletion")
}

func TestDeleteTodo_NotFound(t *testing.T) {
	db, r, _, _ := setupTestDB(t)
	defer db.Close()

	token, err := loginAndGetToken(t, r, "normal_user@example.com", "password123")
	require.NoError(t, err)

	req, _ := http.NewRequest("DELETE", "/api/todos/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}
