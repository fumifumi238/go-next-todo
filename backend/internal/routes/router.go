// Package routesはroutingを行います。
package routes

import (
	"database/sql"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-next-todo/backend/internal/handlers"
	"go-next-todo/backend/internal/repositories"
	"go-next-todo/backend/internal/services"
)

// SetupRouter はGinルーターをセットアップし、すべてのエンドポイントを登録します。
func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// CORS対策
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// リポジトリ
	todoRepo := repositories.NewTodoRepository(db)
	userRepo := repositories.NewUserRepository(db)
	resetRepo := repositories.NewMySQLResetTokenRepo(db)

	// サービス
	todoService := services.NewTodoService(todoRepo)
	userService := services.NewUserService(userRepo, resetRepo)
	jwtService := services.NewJWTService()

	// ハンドラー
	userHandler := handlers.NewUserHandler(userService, jwtService)
	todoHandler := handlers.NewTodoHandler(todoService)

	// ルーティング
	r.GET("/api/hello", HelloHandler)
	r.GET("/api/dbcheck", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Database connection failed", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Database connection is healthy"})
	})
	r.POST("/api/register", userHandler.RegisterHandler)
	r.POST("/api/login", userHandler.LoginHandler)
	r.POST("/api/forgot-password", userHandler.ForgotPasswordHandler)
	r.POST("/api/reset-password/:token", userHandler.ResetPasswordHandler)
	r.POST("/api/reset-password", userHandler.ResetPasswordHandler)

	authorized := r.Group("/")
	authorized.Use(AuthMiddleware(jwtService))
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

func HelloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Go Backend!"})
}
