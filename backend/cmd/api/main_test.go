package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	todoPkg "go-next-todo/backend/internal/todo"
	userPkg "go-next-todo/backend/internal/user"
)

// setupTestDB はテスト用のデータベース接続を確立し、テーブルを作成し、テストデータを投入します。
func setupTestDB() (*sql.DB, error) {
	err := godotenv.Load("../../../.env") // ルート直下の .env を指定
	if err != nil {
		log.Printf("Warning: Could not load .env file for tests: %v", err)
	}

	dbUser := os.Getenv("TEST_DB_USER")
	dbPass := os.Getenv("TEST_DB_PASS")
	dbHost := os.Getenv("TEST_DB_HOST")
	dbPort := os.Getenv("TEST_DB_PORT")
	dbName := os.Getenv("TEST_DB_NAME")

	if dbUser == "" || dbPass == "" || dbHost == "" || dbPort == "" || dbName == "" {
		return nil, fmt.Errorf("database environment variables are not set. USER: %s, PASS: %s, HOST: %s, PORT: %s, NAME: %s", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 既存のテーブルを削除 (テストのたびにクリーンな状態にするため)
	// Foreign Key Constraint があるため、todos -> users の順で削除
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=0;"); err != nil {
		log.Printf("Failed to disable foreign key checks: %v", err)
	}
	if _, err := db.Exec("TRUNCATE TABLE todos"); err != nil {
		log.Printf("Failed to truncate todos table (it might not exist yet): %v", err)
	}
	if _, err := db.Exec("TRUNCATE TABLE users"); err != nil {
		log.Printf("Failed to truncate users table (it might not exist yet): %v", err)
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=1;"); err != nil {
		log.Printf("Failed to enable foreign key checks: %v", err)
	}

	// ユーザーテーブルの作成
	createUserTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createUserTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// ToDoテーブルの作成
	createTodoTableSQL := `
	CREATE TABLE IF NOT EXISTS todos (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createTodoTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create todos table: %w", err)
	}

	// テストユーザーの挿入
	userRepo := userPkg.NewRepository(db)
	hashedPasswordUser, _ := userPkg.HashPassword("password123")
	normalUser := userPkg.User{
		Username:     "normal_user",
		Email:        "normal_user@example.com",
		PasswordHash: hashedPasswordUser,
		Role:         "user",
	}
	if _, err := userRepo.Create(&normalUser); err != nil {
		// すでに存在する場合でもエラーにしない
		log.Printf("Failed to create normal_user (might exist, or duplicate entry): %v", err)
	}

	hashedPasswordAdmin, _ := userPkg.HashPassword("adminpass")
	adminUser := userPkg.User{
		Username:     "admin_user",
		Email:        "admin@example.com",
		PasswordHash: hashedPasswordAdmin,
		Role:         "admin",
	}
	if _, err := userRepo.Create(&adminUser); err != nil {
		log.Printf("Failed to create admin_user (might exist, or duplicate entry): %v", err)
	}

	log.Println("Successfully set up test database!")
	return db, nil
}

// setupRouter はテスト用のGinルーターとリポジトリをセットアップします。
// main.goのルーティング設定と同じものを、テスト用のリポジトリを注入する形で再構築します。
func setupRouter() (*gin.Engine, *sql.DB, *todoPkg.Repository, *userPkg.Repository, error) {
	// Ginをテストモードに設定
	gin.SetMode(gin.TestMode)

	testDB, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup test database: %w", err)
	}

	testTodoRepo := todoPkg.NewRepository(testDB)
	testUserRepo := userPkg.NewRepository(testDB)

	r := gin.Default()

	// ------------------------------------
	// CORS設定をルーターに適用 (main.go と同じ設定)
	// ------------------------------------
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// ------------------------------------
	// main.go のハンドラーをクロージャでラッピングして登録
	// ------------------------------------

	// ヘルスチェック
	r.GET("/api/hello", helloHandler) // helloHandlerは引数を取らないので直接指定
	r.GET("/api/dbcheck", func(c *gin.Context) { dbCheckHandler(c, testDB) })

	// TODO関連ハンドラー
	r.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, testTodoRepo) })
	r.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, testTodoRepo) })
	r.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, testTodoRepo) })
	r.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, testTodoRepo) })
	r.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, testTodoRepo) })

	// ユーザー登録ハンドラー
	r.POST("/api/register", func(c *gin.Context) { registerHandler(c, testUserRepo) })

	// ユーザーログインハンドラー
	r.POST("/api/login", func(c *gin.Context) { loginHandler(c, testUserRepo) })

	return r, testDB, testTodoRepo, testUserRepo, nil
}

// ------------------------------------
// Step 1: ToDo作成 (POST /api/todos) - グリーンフェーズ
// ------------------------------------

func TestCreateTodo_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テスト用のToDoデータ
	newTodo := map[string]interface{}{
		"title":     "Test Todo",
		"completed": false,
		"user_id":   1, // テストユーザーID
	}
	jsonValue, _ := json.Marshal(newTodo)

	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")

	var responseTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.NotZero(t, responseTodo.ID, "Expected a non-zero Todo ID")
	assert.Equal(t, "Test Todo", responseTodo.Title, "Expected title to match")
	assert.False(t, responseTodo.Completed, "Expected completed to be false")
	assert.NotZero(t, responseTodo.CreatedAt, "Expected CreatedAt to be set")
	assert.NotZero(t, responseTodo.UpdatedAt, "Expected UpdatedAt to be set")
	assert.Equal(t, 1, responseTodo.UserID, "Expected UserID to be 1")
}

// ------------------------------------
// Step 2: ToDo一覧取得 (GET /api/todos) - グリーンフェーズ
// ------------------------------------

func TestGetTodos_Success(t *testing.T) {
	// Arrange: ルーターの準備とテストデータの投入
	r, testDB, todoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テスト用のToDoをいくつか作成 (ユーザーID=1)
	todo1 := todoPkg.Todo{Title: "Test Todo 1", Completed: false, UserID: 1}
	_, err = todoRepo.Create(&todo1)
	assert.NoError(t, err)

	time.Sleep(2 * time.Second) // created_at が異なることを保証するため

	todo2 := todoPkg.Todo{Title: "Test Todo 2", Completed: true, UserID: 1}
	_, err = todoRepo.Create(&todo2)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/todos", nil)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")

	var todos []todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &todos)
	assert.NoError(t, err, "Response should be a valid JSON array")
	assert.Len(t, todos, 2, "Expected 2 todos in the response")

	// 作成日時で降順にソートされることを期待 (最新のものが最初)
	assert.Equal(t, "Test Todo 2", todos[0].Title)
	assert.Equal(t, "Test Todo 1", todos[1].Title)
}

// ------------------------------------
// Step 3: 特定のToDo取得 (GET /api/todos/:id) - グリーンフェーズ
// ------------------------------------

func TestGetTodoByID_Success(t *testing.T) {
	r, testDB, todoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// テスト用のToDoを作成
	newTodo := todoPkg.Todo{Title: "Specific Todo", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Specific Todo", responseTodo.Title)
}

func TestGetTodoByID_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	req, _ := http.NewRequest("GET", "/api/todos/99999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

// ------------------------------------
// Step 4: ToDo更新 (PUT /api/todos/:id) - グリーンフェーズ
// ------------------------------------

func TestUpdateTodo_Success(t *testing.T) {
	r, testDB, todoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 更新対象のToDoを作成
	originalTodo := todoPkg.Todo{Title: "Original Todo", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&originalTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	// 短時間スリープを追加して、UpdatedAt の差を確実に作る
	time.Sleep(1 * time.Second) // 1秒スリープ

	// 更新データ
	updatedData := map[string]interface{}{
		"title":     "Updated Todo",
		"completed": true,
		"user_id":   1, // UserIDは更新されないが、リクエストボディに含める
	}
	jsonValue, _ := json.Marshal(updatedData)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/todos/%d", createdTodo.ID), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Updated Todo", responseTodo.Title)
	assert.True(t, responseTodo.Completed)
	assert.True(t, responseTodo.UpdatedAt.After(createdTodo.UpdatedAt), "UpdatedAt should be updated after the original CreatedAt") // メッセージを明確化
}

func TestUpdateTodo_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	updatedData := map[string]interface{}{
		"title":     "Non Existent",
		"completed": true,
		"user_id":   1,
	}
	jsonValue, _ := json.Marshal(updatedData)

	req, _ := http.NewRequest("PUT", "/api/todos/99999", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

// ------------------------------------
// Step 5: ToDo削除 (DELETE /api/todos/:id) - グリーンフェーズ
// ------------------------------------

func TestDeleteTodo_Success(t *testing.T) {
	r, testDB, todoRepo, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 削除対象のToDoを作成
	newTodo := todoPkg.Todo{Title: "Todo to delete", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// 削除されたことを確認 (再取得でNotFoundになるはず)
	_, err = todoRepo.FindByID(createdTodo.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, todoPkg.ErrTodoNotFound))
}

func TestDeleteTodo_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	req, _ := http.NewRequest("DELETE", "/api/todos/99999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

// ----------------------------------------------------
// Step 6: ユーザー登録 (POST /api/register) - グリーンフェーズ
// ----------------------------------------------------

func TestRegisterUser_InvalidInput(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 不完全なユーザー情報 (パスワードなし)
	invalidUser := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		// "password": "", // 意図的に省略
	}
	jsonValue, _ := json.Marshal(invalidUser)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected HTTP Status Code 400 Bad Request for invalid input")

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.Contains(t, response["error"], "Invalid request payload", "Expected error message for invalid payload")
}

// ----------------------------------------------------
// Step 7: ユーザーログイン (POST /api/login) - グリーンフェーズ (修正済み)
// ----------------------------------------------------

func TestLoginUser_Success(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter() // testUserRepo は直接使わないので _ で無視
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// setupTestDB で作成されたユーザーの認証情報を使用
	loginCredentials := map[string]string{
		"email":    "normal_user@example.com", // setupTestDB で作成されたユーザーのメールアドレス
		"password": "password123",            // setupTestDB で作成されたユーザーのパスワード
	}
	jsonValue, _ := json.Marshal(loginCredentials)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 💡 期待値: ステータスコード 200 OK (ログイン成功)
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK for successful login")

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.Contains(t, response, "token", "Expected response to contain a JWT token") // JWTトークンが含まれていることを確認
	assert.NotEmpty(t, response["token"], "Expected JWT token not to be empty")       // JWTトークンが空でないことを確認
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	// Arrange: ルーターの準備
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 無効な認証情報 (存在しないユーザー、または間違ったパスワード)
	invalidCredentials := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}
	jsonValue, _ := json.Marshal(invalidCredentials)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 💡 期待値: ステータスコード 401 Unauthorized (認証失敗)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected HTTP Status Code 401 Unauthorized for invalid credentials")

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be a valid JSON object")
	assert.Contains(t, response["error"], "Invalid credentials", "Expected error message 'Invalid credentials'")
}
