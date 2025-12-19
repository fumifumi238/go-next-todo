// Package testutil はテスト用のdbを設定します。
package testutil

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var testNormalEmail, testAdminEmail string

func SetupTestDB(t *testing.T) (*sql.Tx, *gin.Engine, *repositories.TodoRepository, *repositories.UserRepository) {
    dbUser := os.Getenv("TEST_DB_USER")
    dbPass := os.Getenv("TEST_DB_PASS")
    dbHost := os.Getenv("TEST_DB_HOST")
    dbPort := os.Getenv("TEST_DB_PORT")
    dbName := os.Getenv("TEST_DB_NAME")

    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
    db, err := sql.Open("mysql", dsn)
    require.NoError(t, err)
    require.NoError(t, db.Ping())

    tx, err := db.Begin()
    require.NoError(t, err)

    t.Cleanup(func() {
        tx.Rollback()
        db.Close()
    })

    userRepo := repositories.NewUserRepository(tx)
    todoRepo := repositories.NewTodoRepository(tx)

    router := SetupTestRouter(t, tx)

    return tx, router, todoRepo, userRepo
}
// SetupTestRouter はテスト用のGinルーターとリポジトリをセットアップします。
func SetupTestRouter(t *testing.T, db repositories.DataStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	// リポジトリ
	todoRepo := repositories.NewTodoRepository(db)
	userRepo := repositories.NewUserRepository(db)
	resetTokenRepo := repositories.NewMySQLResetTokenRepo(db)
	adminOTPRepo := repositories.NewAdminOTPRepository(db)

	// サービス
	todoService := services.NewTodoService(todoRepo)
	userService := services.NewUserService(userRepo, resetTokenRepo)
	jwtService := services.NewJWTService()
	adminOTPService := services.NewAdminOTPService(adminOTPRepo)

	// ハンドラー
	userHandler := handlers.NewUserHandler(userService, jwtService, adminOTPService)
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
	r.POST("/api/forgot-password", userHandler.ForgotPasswordHandler)
	r.POST("/api/reset-password/:token", userHandler.ResetPasswordHandler)

	authorized := r.Group("/")

	authorized.Use(routes.AuthMiddleware(jwtService))
	{
		authorized.GET("/api/todos", todoHandler.GetTodosHandler)
		authorized.GET("/api/todos/:id", todoHandler.GetTodoByIDHandler)
		authorized.POST("/api/todos", todoHandler.CreateTodoHandler)
		authorized.PUT("/api/todos/:id", todoHandler.UpdateTodoHandler)
		authorized.DELETE("/api/todos/:id", todoHandler.DeleteTodoHandler)
		authorized.GET("/api/protected", userHandler.ProtectedHandler)

		// 管理者専用ルート
		admin := authorized.Group("/api/admin")
		{
			admin.GET("/users", userHandler.FindAllUsersHandler)
			admin.GET("/todos", todoHandler.FindAllTodosAdminHandler)
			admin.DELETE("/todos/:id", todoHandler.DeleteTodoAdminHandler)
		}
	}
	return r
}

func CreateTestUser(t *testing.T, userRepo *repositories.UserRepository, username, email, password, role string) *models.User {
	hashedPassword, err := repositories.HashPassword(password)
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

// CreateTestTodo はテスト用のTODOを作成し、データベースに保存します。
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

func GetTestEmails() (string, string) {
	return testNormalEmail, testAdminEmail
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
