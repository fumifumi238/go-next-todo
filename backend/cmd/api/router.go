// backend/cmd/api/router.go
package main

import (
	"database/sql"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-next-todo/backend/internal/todo"
	"go-next-todo/backend/internal/user"
)

// setupRouter はGinルーターをセットアップし、すべてのエンドポイントを登録します。
func setupRouter(db *sql.DB, todoRepo *todo.Repository, userRepo *user.Repository) *gin.Engine {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// ルーティングの設定 (クロージャを使用してリポジトリをハンドラーに注入)
	r.GET("/api/hello", helloHandler)
	r.GET("/api/dbcheck", func(c *gin.Context) { dbCheckHandler(c, db) })
	r.POST("/api/register", func(c *gin.Context) { registerHandler(c, userRepo) })
	r.POST("/api/login", func(c *gin.Context) { loginHandler(c, userRepo) })

	authorized := r.Group("/")
	authorized.Use(AuthMiddleware()) // jwt.go で定義されたミドルウェアを適用
	{
		authorized.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, todoRepo) })
		authorized.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, todoRepo) })
		authorized.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, todoRepo) })
		authorized.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, todoRepo) })
		authorized.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, todoRepo) })
		authorized.GET("/api/protected", ProtectedHandler) // テスト用
	}

	return r
}
