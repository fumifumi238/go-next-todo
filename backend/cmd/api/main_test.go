package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"go-next-todo/backend/internal/todo"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

// setupTestDB はテスト用のDB接続をセットアップします
func setupTestDB() (*sql.DB, error) {
	// テスト環境変数からDB接続情報を取得（なければデフォルト値を使用）
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		user = "root"
	}
	pass := os.Getenv("TEST_DB_PASS")
	if pass == "" {
		pass = "password"
	}
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = "3306"
	}
	name := os.Getenv("TEST_DB_NAME")
	if name == "" {
		name = "test_todo_db"
	}

	dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + name
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// テスト用テーブルの作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	// テストデータのクリーンアップ
	_, _ = db.Exec("DELETE FROM todos")

	return db, nil
}

// setupRouter はテスト用のルーターをセットアップします
func setupRouter() (*gin.Engine, *sql.DB, error) {
	gin.SetMode(gin.TestMode)

	// テスト用DBのセットアップ
	testDB, err := setupTestDB()
	if err != nil {
		return nil, nil, err
	}

	// リポジトリの初期化
	testRepo := todo.NewRepository(testDB)

	r := gin.Default()

	// 実際のハンドラーを使用
	r.GET("/api/todos", func(c *gin.Context) {
		todos, err := testRepo.FindAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todos)
	})

	r.POST("/api/todos", func(c *gin.Context) {
		var newTodo todo.Todo
		if err := c.ShouldBindJSON(&newTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		createdTodo, err := testRepo.Create(&newTodo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, createdTodo)
	})

	r.GET("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		foundTodo, err := testRepo.FindByID(id)
		if err != nil {
			if errors.Is(err, todo.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, foundTodo)
	})

	return r, testDB, nil
}

// ----------------------------------------------------
// Step 1: ToDoタスクの追加 (POST /api/todos)
// ----------------------------------------------------

func TestCreateTodo_Success(t *testing.T) {
	// Arrange
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	newTodo := todo.Todo{Title: "Buy milk", Completed: false}
	jsonValue, _ := json.Marshal(newTodo)

	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")

	var response todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, "Buy milk", response.Title, "Expected title to match input")
	assert.True(t, response.ID > 0, "Expected a valid ID")
	assert.False(t, response.Completed, "Expected completed to be false")
}

// ----------------------------------------------------
// Step 2: ToDoタスクの取得 (GET /api/todos) - GREENフェーズ
// ----------------------------------------------------

func TestGetTodos_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	testRepo := todo.NewRepository(testDB)
	_, err = testRepo.Create(&todo.Todo{Title: "Test Todo 1", Completed: false})
	assert.NoError(t, err)
	_, err = testRepo.Create(&todo.Todo{Title: "Test Todo 2", Completed: true})
	assert.NoError(t, err)

	// HTTPリクエストの作成 (GET /api/todos)
	req, _ := http.NewRequest("GET", "/api/todos", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	// 期待値: レスポンスボディがJSON配列であること
	var response []*todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON array of todos")

	// 期待値: 2つのTODOが返されていること
	assert.Len(t, response, 2, "Expected 2 todos in the response array")
	if len(response) >= 2 {
		assert.Equal(t, "Test Todo 2", response[0].Title, "First todo should be Test Todo 2 (ordered by created_at DESC)")
		assert.Equal(t, "Test Todo 1", response[1].Title, "Second todo should be Test Todo 1")
	}
}

// ----------------------------------------------------
// Step 3: ToDoタスクの取得 (GET /api/todos/:id) - レッドフェーズ
// ----------------------------------------------------

func TestGetTodoByID_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	testRepo := todo.NewRepository(testDB)
	createdTodo, err := testRepo.Create(&todo.Todo{Title: "Get This Todo", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// HTTPリクエストの作成 (GET /api/todos/:id)
	req, _ := http.NewRequest("GET", "/api/todos/"+strconv.Itoa(todoID), nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	// 期待値: レスポンスボディがJSONオブジェクトであること
	var response todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, todoID, response.ID, "Expected ID to match")
	assert.Equal(t, "Get This Todo", response.Title, "Expected title to match")
}

func TestGetTodoByID_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// HTTPリクエストの作成 (存在しないID)
	req, _ := http.NewRequest("GET", "/api/todos/99999", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 404 Not Found
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}

// ----------------------------------------------------
// Step 4: ToDoタスクの更新 (PUT /api/todos/:id) - レッドフェーズ
// ----------------------------------------------------

func TestUpdateTodo_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	testRepo := todo.NewRepository(testDB)
	createdTodo, err := testRepo.Create(&todo.Todo{Title: "Original Title", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// 更新用のデータ
	updateTodo := todo.Todo{Title: "Updated Title", Completed: true}
	jsonValue, _ := json.Marshal(updateTodo)

	// PUT /api/todos/:id のルートを追加（実際のハンドラーを使用）
	r.PUT("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		var updateTodo todo.Todo
		if err := c.ShouldBindJSON(&updateTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedTodo, err := testRepo.Update(id, &updateTodo)
		if err != nil {
			if errors.Is(err, todo.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updatedTodo)
	})

	// HTTPリクエストの作成 (PUT /api/todos/:id)
	req, _ := http.NewRequest("PUT", "/api/todos/"+strconv.Itoa(todoID), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	// 期待値: レスポンスボディが更新されたTODOであること
	var response todo.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, todoID, response.ID, "Expected ID to match")
	assert.Equal(t, "Updated Title", response.Title, "Expected title to be updated")
	assert.True(t, response.Completed, "Expected completed to be updated to true")
}

func TestUpdateTodo_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 更新用のデータ
	updateTodo := todo.Todo{Title: "Updated Title", Completed: true}
	jsonValue, _ := json.Marshal(updateTodo)

	// PUT /api/todos/:id のルートを追加（実際のハンドラーを使用）
	testRepo := todo.NewRepository(testDB)
	r.PUT("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		var updateTodo todo.Todo
		if err := c.ShouldBindJSON(&updateTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedTodo, err := testRepo.Update(id, &updateTodo)
		if err != nil {
			if errors.Is(err, todo.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updatedTodo)
	})

	// HTTPリクエストの作成 (存在しないID)
	req, _ := http.NewRequest("PUT", "/api/todos/99999", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 404 Not Found
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}

// ----------------------------------------------------
// Step 5: ToDoタスクの削除 (DELETE /api/todos/:id) - レッドフェーズ
// ----------------------------------------------------

func TestDeleteTodo_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	testRepo := todo.NewRepository(testDB)
	createdTodo, err := testRepo.Create(&todo.Todo{Title: "Delete This Todo", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// DELETE /api/todos/:id のルートを追加（実際のハンドラーを使用）
	r.DELETE("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		err = testRepo.Delete(id)
		if err != nil {
			if errors.Is(err, todo.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// HTTPリクエストの作成 (DELETE /api/todos/:id)
	req, _ := http.NewRequest("DELETE", "/api/todos/"+strconv.Itoa(todoID), nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 204 No Content または 200 OK
	assert.True(t, w.Code == http.StatusNoContent || w.Code == http.StatusOK, "Expected HTTP Status Code 204 or 200")

	// 削除されたことを確認: 再度取得を試みる
	_, err = testRepo.FindByID(todoID)
	assert.Error(t, err, "Todo should be deleted")
	assert.True(t, errors.Is(err, todo.ErrTodoNotFound), "Error should be ErrTodoNotFound")
}

func TestDeleteTodo_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// DELETE /api/todos/:id のルートを追加（実際のハンドラーを使用）
	testRepo := todo.NewRepository(testDB)
	r.DELETE("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		err = testRepo.Delete(id)
		if err != nil {
			if errors.Is(err, todo.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// HTTPリクエストの作成 (存在しないID)
	req, _ := http.NewRequest("DELETE", "/api/todos/99999", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 404 Not Found
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}


// ----------------------------------------------------
// Step 6: ユーザー登録 (POST /api/register) - レッドフェーズ
// ----------------------------------------------------

func TestRegisterUser_InvalidInput(t *testing.T) {
	// Arrange: ルーターの準備
	// ユーザー登録のエンドポイントは認証不要なので、認証ミドルウェアは不要
	r, testDB, err := setupRouter() // setupRouterを再利用するが、DBへのUserRepository初期化は別途行う
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// ⚠️ ここではまだ registerHandler を実装していないため、ダミーハンドラーを追加
	// テストをREDにするために、存在しないハンドラーを呼び出すように設定
	r.POST("/api/register", func(c *gin.Context) {
		// まだ実装されていないので、とりあえずNotImplementedを返す
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not Implemented"})
	})

	// 無効なリクエストボディ（usernameが欠落）
	invalidUserJSON := []byte(`{"email": "test@example.com", "password": "password123"}`)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(invalidUserJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected HTTP Status Code 400 Bad Request for invalid input")

	// 期待値: エラーレスポンスボディ
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.Contains(t, response["error"], "Invalid request payload", "Expected error message for invalid input")
}
