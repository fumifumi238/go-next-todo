// backend/cmd/api/test_setup.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"go-next-todo/backend/internal/todo"
	"go-next-todo/backend/internal/user"
)

// setupTestDB はテスト用のデータベース接続を確立し、テーブルを作成し、テストデータを投入します。
func setupTestDB(t *testing.T) (*sql.DB, *gin.Engine, *todo.Repository, *user.Repository) {
	err := godotenv.Load("../../../.env") // ルート直下の .env を指定
	if err != nil {
		log.Printf("Warning: Could not load .env file for tests: %v", err)
	}

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

// loginAndGetToken はテスト用のヘルパー関数で、指定された資格情報でログインし、JWTトークンを返します。
func loginAndGetToken(t *testing.T, r *gin.Engine, email, password string) (string, error) {
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
	return tokenString, err
}

// truncateTables はテスト実行前にDBのテーブルを空にする
func truncateTables(db *sql.DB) {
	// Foreign Key Constraint があるため、todos -> users の順で削除
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=0;"); err != nil {
		log.Fatalf("Failed to disable foreign key checks: %v", err)
	}
	if _, err := db.Exec("TRUNCATE TABLE todos"); err != nil {
		log.Fatalf("Failed to truncate todos table: %v", err)
	}
	if _, err := db.Exec("TRUNCATE TABLE users"); err != nil {
		log.Fatalf("Failed to truncate users table: %v", err)
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=1;"); err != nil {
		log.Fatalf("Failed to enable foreign key checks: %v", err)
	}
}
