// backend/cmd/api/test_setup.go
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

	"github.com/stretchr/testify/require"

	"go-next-todo/backend/internal/todo"
	"go-next-todo/backend/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// setupTestDB はテスト用のデータベース接続を確立し、テーブルを作成し、テストデータを投入します。
func setupTestDB(t *testing.T) (*sql.DB, *gin.Engine, *todo.Repository, *user.Repository) {
err := godotenv.Load("../../../.env")
	dbUser := os.Getenv("TEST_DB_USER")
	dbPass := os.Getenv("TEST_DB_PASS")
	dbHost := os.Getenv("TEST_DB_HOST")
	dbPort := os.Getenv("TEST_DB_PORT")
	dbName := os.Getenv("TEST_DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping database: %v", err)
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
		t.Fatalf("Failed to create users table: %v", err)
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
		t.Fatalf("Failed to create todos table: %v", err)
	}

	// テストユーザーの挿入
	userRepo := user.NewRepository(db)
	hashedPasswordUser, _ := user.HashPassword("password123")
	normalUser := user.User{
		Username:     "normal_user",
		Email:        "normal_user@example.com",
		PasswordHash: hashedPasswordUser,
		Role:         "user",
	}
	if _, err := userRepo.Create(&normalUser); err != nil {
		log.Printf("Failed to create normal_user (might exist, or duplicate entry): %v", err)
	}

	hashedPasswordAdmin, _ := user.HashPassword("adminpass")
	adminUser := user.User{
		Username:     "admin_user",
		Email:        "admin@example.com",
		PasswordHash: hashedPasswordAdmin,
		Role:         "admin",
	}
	if _, err := userRepo.Create(&adminUser); err != nil {
		log.Printf("Failed to create admin_user (might exist, or duplicate entry): %v", err)
	}

	log.Println("Successfully set up test database!")

	// Ginルーターのセットアップ
	router := setupTestRouter(t, db) // テスト専用のルーターセットアップ関数
	todoRepo := todo.NewRepository(db)

	return db, router, todoRepo, userRepo
}

// setupTestRouter はテスト用のGinルーターとリポジトリをセットアップします。
func setupTestRouter(t *testing.T, db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	testTodoRepo := todo.NewRepository(db)
	testUserRepo := user.NewRepository(db)

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	InitJWTSecretForTest() // jwt.go で定義されたテスト用関数を呼び出す

	r.GET("/api/hello", helloHandler)
	r.GET("/api/dbcheck", func(c *gin.Context) { dbCheckHandler(c, db) })

	r.POST("/api/register", func(c *gin.Context) { registerHandler(c, testUserRepo) })
	r.POST("/api/login", func(c *gin.Context) { loginHandler(c, testUserRepo) })

	authorized := r.Group("/")
	authorized.Use(AuthMiddleware()) // jwt.go で定義されたミドルウェアを適用
	{
		authorized.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, testTodoRepo) })
		authorized.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, testTodoRepo) })
		authorized.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, testTodoRepo) })
		authorized.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, testTodoRepo) })
		authorized.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, testTodoRepo) })
		authorized.GET("/api/protected", ProtectedHandler)
	}

	return r
}

func createTestUser(t *testing.T, userRepo *user.Repository, username, email, password, role string) *user.User {
	hashedPassword, err := user.HashPassword(password)
	require.NoError(t, err)

	newUser := user.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         role,
	}

	createdUser, err := userRepo.Create(&newUser)
	require.NoError(t, err)
	require.NotNil(t, createdUser)
	require.NotEqual(t, 0, createdUser.ID)
	return createdUser
}
// createTestTodo はテスト用のTODOを作成し、データベースに保存します。
func createTestTodo(t *testing.T, router *gin.Engine, token, title string, completed bool) *todo.Todo {
	todoPayload := map[string]interface{}{
		"title":     title,
		"completed": completed,
	}
	body, _ := json.Marshal(todoPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/todos", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code, "TODO作成に失敗しました: %s", resp.Body.String())

	var createdTodo todo.Todo
	err := json.Unmarshal(resp.Body.Bytes(), &createdTodo)
	require.NoError(t, err)
	return &createdTodo
}

func loginAndGetToken(t *testing.T, router *gin.Engine, email, password string) (string, error) {
	loginPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(loginPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(body)) // ルーターのパスに合わせて /login に変更
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %s", resp.Code, resp.Body.String())
	}

	var loginRes map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &loginRes)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	token, ok := loginRes["token"]
	if !ok {
		return "", errors.New("token not found in login response")
	}
	return token, nil
}
