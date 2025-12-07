package testutil

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

	"go-next-todo/backend/internal/handlers"
	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
	"go-next-todo/backend/internal/routes"
	"go-next-todo/backend/internal/services"
	"go-next-todo/backend/internal/user"

	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// setupTestDB はテスト用のデータベース接続を確立し、テーブルを作成し、テストデータを投入します。
func SetupTestDB(t *testing.T) (*sql.DB, *gin.Engine, *repositories.TodoRepository, *repositories.UserRepository) {

	err := godotenv.Load("../../../.env")

	dbUser := os.Getenv("TEST_DB_USER")
	dbPass := os.Getenv("TEST_DB_PASS")
	dbHost := os.Getenv("TEST_DB_HOST")
	dbPort := os.Getenv("TEST_DB_PORT")
	dbName := os.Getenv("TEST_DB_NAME")

	// In Docker container, use "db" as hostname instead of 127.0.0.1
	if dbHost == "127.0.0.1" {
		dbHost = "db"
	}

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
	userRepo := repositories.NewUserRepository(db)
	hashedPasswordUser, _ := user.HashPassword("password123")
	normalUser := models.User{
		Username:     "normal_user",
		Email:        "normal_user@example.com",
		PasswordHash: hashedPasswordUser,
		Role:         "user",
	}
	if _, err := userRepo.Create(&normalUser); err != nil {
		log.Printf("Failed to create normal_user (might exist, or duplicate entry): %v", err)
	}

	hashedPasswordAdmin, _ := user.HashPassword("adminpass")
	adminUser := models.User{
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
	router := SetupTestRouter(t, db) // テスト専用のルーターセットアップ関数
	todoRepo := repositories.NewTodoRepository(db)

	return db, router, todoRepo, userRepo
}

// setupTestRouter はテスト用のGinルーターとリポジトリをセットアップします。
func SetupTestRouter(t *testing.T, db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	// リポジトリ
	todoRepo := repositories.NewTodoRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// サービス
	todoService := services.NewTodoService(todoRepo)
	userService := services.NewUserService(userRepo)
	jwtService := services.NewJWTService()

	// ハンドラー
	userHandler := handlers.NewUserHandler(userService, jwtService)
	todoHandler := handlers.NewTodoHandler(todoService)
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// r.GET("/api/hello", routes.HelloHandler)
	// r.GET("/api/dbcheck", func(c *gin.Context) { routes.DbCheckHandler(c, db) })

	r.POST("/api/register", userHandler.RegisterHandler)
	r.POST("/api/login", userHandler.LoginHandler)

	authorized := r.Group("/")

	authorized.Use(routes.AuthMiddleware(jwtService))
	{
		authorized.GET("/api/todos", todoHandler.GetTodosHandler)
		authorized.GET("/api/todos/:id", todoHandler.GetTodoByIDHandler)
		authorized.POST("/api/todos", todoHandler.CreateTodoHandler)
		authorized.PUT("/api/todos/:id", todoHandler.UpdateTodoHandler)
		authorized.DELETE("/api/todos/:id", todoHandler.DeleteTodoHandler)
		authorized.GET("/api/protected", userHandler.ProtectedHandler)
	}
	return r
}

func CreateTestUser(t *testing.T, userRepo *repositories.UserRepository, username, email, password, role string) *models.User {
	hashedPassword, err := user.HashPassword(password)
	require.NoError(t, err)

	newUser := models.User{
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
func CreateTestTodo(t *testing.T, router *gin.Engine, token, title string, completed bool) *models.Todo {
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

	var createdTodo models.Todo
	err := json.Unmarshal(resp.Body.Bytes(), &createdTodo)
	require.NoError(t, err)
	return &createdTodo
}

func LoginAndGetToken(t *testing.T, router *gin.Engine, email, password string) (string, error) {
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

	var loginRes map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &loginRes)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	token, ok := loginRes["token"].(string)
	if !ok {
		return "", errors.New("token not found or not a string in login response")
	}
	return token, nil
}
