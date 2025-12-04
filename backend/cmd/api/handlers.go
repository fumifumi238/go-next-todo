// backend/cmd/api/handlers.go
package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-next-todo/backend/internal/todo"
	"go-next-todo/backend/internal/user"
)

// helloHandler はシンプルなヘルスチェックエンドポイントです。
func helloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Go Backend!"})
}

// dbCheckHandler はデータベース接続の健全性を確認します。
func dbCheckHandler(c *gin.Context, db *sql.DB) {
	if err := db.Ping(); err != nil {
		log.Printf("DB Ping failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Database connection is healthy"})
}

// registerHandler はユーザー登録ハンドラー
func registerHandler(c *gin.Context, userRepo *user.Repository) {
	var req user.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email, and password are required"})
		return
	}
	hashedPassword, err := user.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password", "details": err.Error()})
		return
	}
	newUser := user.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
	}
	createdUser, err := userRepo.Create(&newUser)
	if err != nil {
		if errors.Is(err, user.ErrDuplicateEmail) {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user", "details": err.Error()})
		return
	}
	createdUser.PasswordHash = ""
	c.JSON(http.StatusCreated, createdUser)
}

// loginHandler はユーザーログインを処理し、成功した場合はJWTを返します。
func loginHandler(c *gin.Context, userRepo *user.Repository) {
	var req user.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	foundUser, err := userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err := user.VerifyPassword(foundUser.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// JWTトークンを生成
	tokenString, err := GenerateToken(uint(foundUser.ID), foundUser.Email, foundUser.Role)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
c.JSON(http.StatusOK, gin.H{"token": tokenString, "user_id": foundUser.ID, "role": foundUser.Role})
}

// createTodoHandler は新しいToDoタスクを作成し、DBに保存します。
func createTodoHandler(c *gin.Context, todoRepo *todo.Repository) {
	var newTodo todo.Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	newTodo.UserID = userID
	createdTodo, err := todoRepo.Create(&newTodo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save todo to database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, createdTodo)
}


// updateTodoHandler は指定されたIDのToDoタスクを更新します。
func updateTodoHandler(c *gin.Context, todoRepo *todo.Repository) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	// まずTODOを取得し、所有者を確認
	existingTodo, err := todoRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, todo.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		log.Printf("Failed to fetch todo for update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo for update", "details": err.Error()})
		return
	}

	// 認可チェック: 自分のTODOであるか、管理者であるか
	if existingTodo.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var updateTodo todo.Todo
	if err := c.ShouldBindJSON(&updateTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// 💡 ユーザーが他のユーザーのTODOを更新しようとしても、UserIDはコンテキストから取得したものに上書きする
	updateTodo.UserID = existingTodo.UserID // 元の所有者を保持するか、あるいは更新させない
	// あるいは、updateTodo.UserID = userID とすることで、更新者が所有者になるようにすることも可能。今回は既存の所有者を保持する方針。

	updatedTodo, err := todoRepo.Update(id, &updateTodo)
	if err != nil {
		if errors.Is(err, todo.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		log.Printf("Failed to update todo in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update todo in database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedTodo)
}

// deleteTodoHandler は指定されたIDのToDoタスクを削除します。
func deleteTodoHandler(c *gin.Context, todoRepo *todo.Repository) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	// まずTODOを取得し、所有者を確認
	existingTodo, err := todoRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, todo.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		log.Printf("Failed to fetch todo for deletion: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo for deletion", "details": err.Error()})
		return
	}

	// 認可チェック: 自分のTODOであるか、管理者であるか
	if existingTodo.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = todoRepo.Delete(id)
	if err != nil {
		if errors.Is(err, todo.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		log.Printf("Failed to delete todo from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo from database", "details": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}


// getTodosHandler は認証されたユーザーのToDoタスク、または管理者であればすべてのToDoタスクを取得します。
func getTodosHandler(c *gin.Context, todoRepo *todo.Repository) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	var todos []*todo.Todo
	var err error

	if userRole == "admin" {
		// 管理者はすべてのTODOを取得
		todos, err = todoRepo.FindAll()
	} else {
		// 一般ユーザーは自分のTODOのみ取得
		todos, err = todoRepo.FindByUserID(userID)
	}

	if err != nil {
		log.Printf("Failed to fetch todos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos from database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, todos)
}

// getTodoByIDHandler は指定されたIDのToDoタスクを取得します。
func getTodoByIDHandler(c *gin.Context, todoRepo *todo.Repository) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	foundTodo, err := todoRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, todo.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		log.Printf("Failed to fetch todo from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo from database", "details": err.Error()})
		return
	}

	// 認可チェック: 自分のTODOであるか、管理者であるか
	if foundTodo.UserID != userID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, foundTodo)
}

// ProtectedHandler は認証ミドルウェアのテストで使用されるダミーの保護されたエンドポイントです。
func ProtectedHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found in token claims"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in token claims"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Access granted",
		"user_id": userID,
		"email":   userEmail,
		"role":    userRole,
	})
}
