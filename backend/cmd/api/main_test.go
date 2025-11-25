package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	// timeパッケージがCreateTodo内で使われているため追加
	todoPkg "go-next-todo/backend/internal/todo"
	userPkg "go-next-todo/backend/internal/user"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

// setupTestDB はテスト用のDB接続をセットアップします
func setupTestDB() (*sql.DB, error) {

	user := os.Getenv("TEST_DB_USER")
	pass := os.Getenv("TEST_DB_PASS")
	host := os.Getenv("TEST_DB_HOST")
	port := os.Getenv("TEST_DB_PORT")
	name := os.Getenv("TEST_DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name) // parseTime=trueを追加
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

// ... setupTestDB 関数の途中 ...

	// テスト用テーブルの作成
	// users テーブルの作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role ENUM('user', 'admin') NOT NULL DEFAULT 'user',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`) // 💡 セミコロンを削除
	if err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err) // エラーメッセージをより具体的に
	}

	// todos テーブルの作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            CONSTRAINT fk_user_id
                FOREIGN KEY (user_id)
                REFERENCES users(id)
                ON DELETE CASCADE
		)
	`) // 💡 セミコロンを削除
	if err != nil {
		return nil, fmt.Errorf("failed to create todos table: %w", err) // エラーメッセージをより具体的に
	}


	// テストデータのクリーンアップ
    _, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0") // 一時的に制約を無効化
	_, _ = db.Exec("TRUNCATE TABLE todos")
	_, _ = db.Exec("TRUNCATE TABLE users")
    _, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1") // 再び有効化

    // 💡 テスト用のユーザーを作成
    // パスワードはテストなので単純なものでOK
    hashedPassword, _ := userPkg.HashPassword("testpassword")
    _, err = db.Exec("INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
        "testuser", "test@example.com", hashedPassword, "user")
    if err != nil {
        return nil, fmt.Errorf("failed to create test user: %w", err)
    }
    _, err = db.Exec("INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
        "adminuser", "admin@example.com", hashedPassword, "admin")
    if err != nil {
        return nil, fmt.Errorf("failed to create admin user: %w", err)
    }

	return db, nil
}

// setupRouter はテスト用のルーターとDB接続、リポジトリをセットアップします
func setupRouter() (*gin.Engine, *sql.DB, *todoPkg.Repository, *userPkg.Repository, error) {
	gin.SetMode(gin.TestMode)

	testDB, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	testTodoRepo := todoPkg.NewRepository(testDB)
	testUserRepo := userPkg.NewRepository(testDB)

	r := gin.Default()

	// ------------------------------------
	// 💡 既存のTODO関連ハンドラー (テスト用)
	// ------------------------------------
	r.GET("/api/todos", func(c *gin.Context) {
		todos, err := testTodoRepo.FindAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todos)
	})

	r.POST("/api/todos", func(c *gin.Context) {
		var newTodo todoPkg.Todo
		if err := c.ShouldBindJSON(&newTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		createdTodo, err := testTodoRepo.Create(&newTodo)
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
		foundTodo, err := testTodoRepo.FindByID(id)
		if err != nil {
			if errors.Is(err, todoPkg.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, foundTodo)
	})

	r.PUT("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		var updateTodo todoPkg.Todo
		if err := c.ShouldBindJSON(&updateTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedTodo, err := testTodoRepo.Update(id, &updateTodo)
		if err != nil {
			if errors.Is(err, todoPkg.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updatedTodo)
	})

	r.DELETE("/api/todos/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		err = testTodoRepo.Delete(id)
		if err != nil {
			if errors.Is(err, todoPkg.ErrTodoNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// ------------------------------------
	// 💡 ユーザー登録ハンドラーのダミー設定（まだ実装はしない）
	// ------------------------------------
	r.POST("/api/register", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not Implemented"})
	})

	return r, testDB, testTodoRepo, testUserRepo, nil
}

// ----------------------------------------------------\
// Step 1: ToDoタスクの追加 (POST /api/todos)
// ----------------------------------------------------

func TestCreateTodo_Success(t *testing.T) {
	// Arrange
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	newTodo := todoPkg.Todo{UserID:1,Title: "Buy milk", Completed: false}
	jsonValue, _ := json.Marshal(newTodo)

	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")

	var response todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, "Buy milk", response.Title, "Expected title to match input")
	assert.True(t, response.ID > 0, "Expected a valid ID")
	assert.False(t, response.Completed, "Expected completed to be false")
}

// ----------------------------------------------------\
// Step 2: ToDoタスクの取得 (GET /api/todos)
// ----------------------------------------------------

func TestGetTodos_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, testTodoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	_, err = testTodoRepo.Create(&todoPkg.Todo{UserID:1,Title: "Test Todo 1", Completed: false})
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	_, err = testTodoRepo.Create(&todoPkg.Todo{UserID:1,Title: "Test Todo 2", Completed: true})
	assert.NoError(t, err)

	// HTTPリクエストの作成 (GET /api/todos)
	req, _ := http.NewRequest("GET", "/api/todos", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	var response []*todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON array of todos")

	assert.Len(t, response, 2, "Expected 2 todos in the response array")
	if len(response) >= 2 {
		assert.Equal(t, "Test Todo 2", response[0].Title, "First todo should be Test Todo 2 (ordered by created_at DESC)")
		assert.Equal(t, "Test Todo 1", response[1].Title, "Second todo should be Test Todo 1")
	}
}

// ----------------------------------------------------\
// Step 3: ToDoタスクの取得 (GET /api/todos/:id)
// ----------------------------------------------------

func TestGetTodoByID_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, testTodoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	createdTodo, err := testTodoRepo.Create(&todoPkg.Todo{UserID:1,Title: "Get This Todo", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// HTTPリクエストの作成 (GET /api/todos/:id)
	req, _ := http.NewRequest("GET", "/api/todos/"+strconv.Itoa(todoID), nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	var response todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, todoID, response.ID, "Expected ID to match")
	assert.Equal(t, "Get This Todo", response.Title, "Expected title to match")
}

func TestGetTodoByID_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter() // testTodoRepoは使わないので_で無視
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
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}

// ----------------------------------------------------\
// Step 4: ToDoタスクの更新 (PUT /api/todos/:id)
// ----------------------------------------------------

func TestUpdateTodo_Success(t *testing.T) {
	// Arrange
	r, testDB, testTodoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	createdTodo, err := testTodoRepo.Create(&todoPkg.Todo{UserID:1,Title: "Original Title", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// 更新用のデータ
	updateTodo := todoPkg.Todo{Title: "Updated Title", Completed: true}
	jsonValue, _ := json.Marshal(updateTodo)

	req, _ := http.NewRequest("PUT", "/api/todos/"+strconv.Itoa(todoID), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	var response todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON todo object")
	assert.Equal(t, todoID, response.ID, "Expected ID to match")
	assert.Equal(t, "Updated Title", response.Title, "Expected title to be updated")
	assert.True(t, response.Completed, "Expected completed to be updated to true")
}

func TestUpdateTodo_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter() // testTodoRepoは使わないので_で無視
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 更新用のデータ
	updateTodo := todoPkg.Todo{Title: "Updated Title", Completed: true}
	jsonValue, _ := json.Marshal(updateTodo)

	req, _ := http.NewRequest("PUT", "/api/todos/99999", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}

// ----------------------------------------------------\
// Step 5: ToDoタスクの削除 (DELETE /api/todos/:id)
// ----------------------------------------------------

func TestDeleteTodo_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, testTodoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テストデータの準備: まずTODOを作成
	createdTodo, err := testTodoRepo.Create(&todoPkg.Todo{UserID:1,Title: "Delete This Todo", Completed: false})
	assert.NoError(t, err)
	todoID := createdTodo.ID

	// HTTPリクエストの作成 (DELETE /api/todos/:id)
	req, _ := http.NewRequest("DELETE", "/api/todos/"+strconv.Itoa(todoID), nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.True(t, w.Code == http.StatusNoContent || w.Code == http.StatusOK, "Expected HTTP Status Code 204 or 200")

	// 削除されたことを確認: 再度取得を試みる
	_, err = testTodoRepo.FindByID(todoID)
	assert.Error(t, err, "Todo should be deleted")
	assert.True(t, errors.Is(err, todoPkg.ErrTodoNotFound), "Error should be ErrTodoNotFound")
}

func TestDeleteTodo_NotFound(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter() // testTodoRepoは使わないので_で無視
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// HTTPリクエストの作成 (存在しないID)
	req, _ := http.NewRequest("DELETE", "/api/todos/99999", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected HTTP Status Code 404 Not Found")
}

// ----------------------------------------------------
// Step 6: ユーザー登録 (POST /api/register) - レッドフェーズ
// ----------------------------------------------------

func TestRegisterUser_InvalidInput(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter() // testTodoRepo, testUserRepo はこのテストでは直接使わないので _ で無視
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 無効なリクエストボディ（usernameが欠落）
	invalidUserJSON := []byte(`{"email": "test@example.com", "password": "password123"}`)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(invalidUserJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 501 Not Implemented (ダミーハンドラーのため)
	assert.Equal(t, http.StatusNotImplemented, w.Code, "Expected HTTP Status Code 501 Not Implemented (for dummy handler)")

	// 期待値: エラーレスポンスボディ
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.Contains(t, response["error"], "Not Implemented", "Expected error message from dummy handler")
}
