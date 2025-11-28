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

	// JWTシークレットをテスト用に初期化
	os.Setenv("JWT_SECRET", "test_very_secret_jwt_key_here")
	InitJWTSecretForTest() // main.go で定義したヘルパー関数を呼び出す

	// ヘルスチェック
	r.GET("/api/hello", helloHandler)
	r.GET("/api/dbcheck", func(c *gin.Context) { dbCheckHandler(c, testDB) })

	// TODO関連ハンドラー
	// These routes are now protected by AuthMiddleware in main.go
	// They should be called within the authorized group in tests as well,
	// or have token explicitly added for non-group calls.
	r.POST("/api/register", func(c *gin.Context) { registerHandler(c, testUserRepo) })
	r.POST("/api/login", func(c *gin.Context) { loginHandler(c, testUserRepo) })

	// 認証ミドルウェアが適用されるルートグループ
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware()) // main.go で定義した実際のAuthMiddlewareを適用
	{
		authorized.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, testTodoRepo) })
		authorized.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, testTodoRepo) })
		authorized.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, testTodoRepo) })
		authorized.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, testTodoRepo) })
		authorized.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, testTodoRepo) })
		authorized.GET("/api/protected", ProtectedHandler)
	}

	return r, testDB, testTodoRepo, testUserRepo, nil
}

// 💡 JWTトークンを取得するヘルパー関数
func loginAndGetToken(t *testing.T, r *gin.Engine, email, password string) string {
	loginCredentials := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonValue, _ := json.Marshal(loginCredentials)

	loginReq, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonValue))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	assert.Equal(t, http.StatusOK, loginW.Code, "Login failed: Expected HTTP Status Code 200 OK")
	var loginResponse map[string]string
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	assert.NoError(t, err, "Failed to unmarshal login response")
	tokenString, exists := loginResponse["token"]
	assert.True(t, exists, "Expected JWT token from login response")
	assert.NotEmpty(t, tokenString, "Expected JWT token not to be empty")
	return tokenString
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

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	newTodo := todoPkg.Todo{
		UserID:    1, // normal_userのIDは1
		Title:     "Test Todo",
		Completed: false,
	}
	jsonValue, _ := json.Marshal(newTodo)

	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")
	var createdTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &createdTodo)
	assert.NoError(t, err, "Response should be a valid JSON todo object")

	assert.NotZero(t, createdTodo.ID, "Expected a non-zero Todo ID")
	assert.Equal(t, newTodo.Title, createdTodo.Title, "Expected title to match")
	assert.False(t, createdTodo.Completed, "Expected completed to be false")
	assert.NotZero(t, createdTodo.CreatedAt, "Expected CreatedAt to be set")
	assert.NotZero(t, createdTodo.UpdatedAt, "Expected UpdatedAt to be set")
	assert.Equal(t, newTodo.UserID, createdTodo.UserID, "Expected UserID to be 1")
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

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	// 既存のTODOを挿入
	todo1 := &todoPkg.Todo{UserID: 1, Title: "Test Todo 1", Completed: false}
	todo2 := &todoPkg.Todo{UserID: 1, Title: "Test Todo 2", Completed: true}
	createdTodo1, err := todoRepo.Create(todo1)
	assert.NoError(t, err)
	time.Sleep(2 * time.Second) // 異なる CreatedAt タイムスタンプを保証するために一時停止
	createdTodo2, err := todoRepo.Create(todo2)
	assert.NoError(t, err)

	// リクエストを作成 (💡 JWTトークンをヘッダーに追加)
	req, _ := http.NewRequest("GET", "/api/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP Status Code 200 OK")
	var todos []todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &todos)
	assert.NoError(t, err, "Response should be a valid JSON array")
	assert.Len(t, todos, 2, "Expected 2 todos in the response")

	// CreatedAtの新しい順にソートされていることを確認
	assert.Equal(t, createdTodo2.Title, todos[0].Title, "Expected 'Test Todo 2' to be the first todo")
	assert.Equal(t, createdTodo1.Title, todos[1].Title, "Expected 'Test Todo 1' to be the second todo")
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

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	// テスト用のToDoを作成 (UserIDも設定)
	newTodo := todoPkg.Todo{Title: "Specific Todo", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Specific Todo", responseTodo.Title)
	assert.Equal(t, newTodo.UserID, responseTodo.UserID, "Expected UserID to match") // UserIDのアサーションを追加
}

func TestGetTodoByID_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	req, _ := http.NewRequest("GET", "/api/todos/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
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

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

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
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseTodo todoPkg.Todo
	err = json.Unmarshal(w.Body.Bytes(), &responseTodo)
	assert.NoError(t, err)
	assert.Equal(t, createdTodo.ID, responseTodo.ID)
	assert.Equal(t, "Updated Todo", responseTodo.Title)
	assert.True(t, responseTodo.Completed)
	assert.True(t, responseTodo.UpdatedAt.After(createdTodo.UpdatedAt), "UpdatedAt should be updated after the original CreatedAt")
	assert.Equal(t, originalTodo.UserID, responseTodo.UserID, "Expected UserID to remain the same") // UserIDのアサーションを追加
}

func TestUpdateTodo_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	updatedData := map[string]interface{}{
		"title":     "Non Existent Todo",
		"completed": true,
		"user_id":   1,
	}
	jsonValue, _ := json.Marshal(updatedData)

	req, _ := http.NewRequest("PUT", "/api/todos/99999", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
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

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	// 削除対象のToDoを作成
	newTodo := todoPkg.Todo{Title: "Todo to Delete", Completed: false, UserID: 1}
	createdTodo, err := todoRepo.Create(&newTodo)
	assert.NoError(t, err)
	assert.NotZero(t, createdTodo.ID)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/todos/%d", createdTodo.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// 削除されたことを確認
	_, err = todoRepo.FindByID(createdTodo.ID)
	assert.Error(t, err, "Expected an error as todo should be deleted")
	assert.True(t, errors.Is(err, todoPkg.ErrTodoNotFound), "Expected ErrTodoNotFound after deletion")
}

func TestDeleteTodo_NotFound(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 💡 ログインしてJWTトークンを取得
	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

	req, _ := http.NewRequest("DELETE", "/api/todos/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token) // 💡 JWTトークンをヘッダーに追加
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Todo not found")
}

// ------------------------------------
// User Registration Tests (POST /api/register)
// ------------------------------------

func TestRegisterUser_Success(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 新しいユーザー登録データ
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

	var responseUser userPkg.User
	err = json.Unmarshal(w.Body.Bytes(), &responseUser)
	assert.NoError(t, err, "Response should be a valid JSON user object")
	assert.NotZero(t, responseUser.ID, "Expected a non-zero User ID")
	assert.Equal(t, "newuser", responseUser.Username, "Expected username to match")
	assert.Equal(t, "newuser@example.com", responseUser.Email, "Expected email to match")
	assert.Equal(t, "user", responseUser.Role, "Expected default role to be 'user'")
	assert.Empty(t, responseUser.PasswordHash, "Password hash should not be returned in response")
}

func TestRegisterUser_InvalidInput(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 無効な入力 (パスワードなし)
	invalidUserData := map[string]string{
		"username": "invaliduser",
		"email":    "invalid@example.com",
		// "password": "" // パスワードがない
	}
	jsonValue, _ := json.Marshal(invalidUserData)

	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected HTTP Status Code 400 Bad Request")
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request payload", "Expected 'Invalid request payload' error")
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	r, testDB, _, userRepo, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 既存のメールアドレスでユーザーを作成
	existingUser := userPkg.User{
		Username:     "existing",
		Email:        "duplicate@example.com",
		PasswordHash: "hashedpass",
		Role:         "user",
	}
	_, err = userRepo.Create(&existingUser)
	assert.NoError(t, err)

	// 同じメールアドレスで再度登録を試みる
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

// ------------------------------------
// User Login Tests (POST /api/login)
// ------------------------------------

func TestLoginUser_Success(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 既存ユーザーのログイン情報 (setupTestDBで作成済み)
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
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "token", "Expected response to contain a 'token'")
	assert.NotEmpty(t, response["token"], "Expected token to be non-empty")
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	// 存在しないユーザー、または間違ったパスワード
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
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid credentials", "Expected 'Invalid credentials' error")
}

// ------------------------------------
// AuthMiddleware Tests
// ------------------------------------

func TestAuthMiddleware_ValidToken(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	token := loginAndGetToken(t, r, "normal_user@example.com", "password123")

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
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token") // 不正なトークン
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid or expired token")
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	r, testDB, _, _, err := setupRouter()
	if err != nil {
		t.Skipf("Skipping test: Failed to setup router (DB connection required): %v", err)
	}
	defer testDB.Close()

	req, _ := http.NewRequest("GET", "/api/protected", nil) // トークンなし
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Authorization header required")
}
