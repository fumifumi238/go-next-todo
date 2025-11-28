package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	todoPkg "go-next-todo/backend/internal/todo"
	userPkg "go-next-todo/backend/internal/user"
)

// DB接続をグローバル（または構造体）に保持するため、db変数とリポジトリ変数を定義
var db *sql.DB
var todoRepo *todoPkg.Repository
var userRepo *userPkg.Repository

// JWT署名のためのシークレットキー
var jwtSecret []byte

// getDSN は環境変数からMySQL接続文字列 (DSN) を構築します。
func getDSN() string {
	err := godotenv.Load("../../../.env") // ルート直下の .env を指定
	if err != nil {
		log.Printf("Error loading .env file (this is fine if using explicit env vars): %v", err)
	}
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
}

// initDB はデータベース接続を初期化します。
func initDB() {
	dsn := getDSN()
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Fatal: Failed to open database connection: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Fatalf("Fatal: Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to MySQL database!")
}

// createTodoHandler は新しいToDoタスクを作成し、DBに保存します。
func createTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) { // 💡 引数としてrepoを受け取る
	var newTodo todoPkg.Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

		// AuthMiddlewareでセットされたuser_idを取得
	// AuthMiddlewareはJWTトークンが有効な場合にのみc.Set("user_id", int(userID)) を実行するため、
	// ここではエラーチェックは不要だが、型アサーションの安全性を考慮する
	userIDVal, exists := c.Get("user_id")
	if !exists {
		// user_idがコンテキストに存在しない場合（ミドルウェアが正しく動作していないか、ルート保護がない場合）
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		// user_idがint型でない場合（型アサーション失敗）
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	// 取得したuser_idをnewTodoにセット
	newTodo.UserID = userID
	createdTodo, err := todoRepo.Create(&newTodo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save todo to database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, createdTodo)
}

// getTodoByIDHandler は指定されたIDのToDoタスクを取得します。
func getTodoByIDHandler(c *gin.Context, todoRepo *todoPkg.Repository) { // 💡 引数としてrepoを受け取る
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	foundTodo, err := todoRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, todoPkg.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo from database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, foundTodo)
}

// updateTodoHandler は指定されたIDのToDoタスクを更新します。
func updateTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) { // 💡 引数としてrepoを受け取る
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var updateTodo todoPkg.Todo
	if err := c.ShouldBindJSON(&updateTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	updatedTodo, err := todoRepo.Update(id, &updateTodo)
	if err != nil {
		if errors.Is(err, todoPkg.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update todo in database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedTodo)
}

// deleteTodoHandler は指定されたIDのToDoタスクを削除します。
func deleteTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) { // 💡 引数としてrepoを受け取る
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	err = todoRepo.Delete(id)
	if err != nil {
		if errors.Is(err, todoPkg.ErrTodoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo from database", "details": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// getTodosHandler はすべてのToDoタスクを取得します。
func getTodosHandler(c *gin.Context, todoRepo *todoPkg.Repository) { // 💡 引数としてrepoを受け取る
	todos, err := todoRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos from database", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, todos)
}

// registerHandler はユーザー登録ハンドラー
func registerHandler(c *gin.Context, userRepo *userPkg.Repository) { // 💡 引数としてrepoを受け取る
	var req userPkg.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email, and password are required"})
		return
	}
	hashedPassword, err := userPkg.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password", "details": err.Error()})
		return
	}
	newUser := userPkg.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
	}
	createdUser, err := userRepo.Create(&newUser)
	if err != nil {
		// userPkg.ErrDuplicateEmail をチェックする
		if errors.Is(err, userPkg.ErrDuplicateEmail) {
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
func loginHandler(c *gin.Context, userRepo *userPkg.Repository) { // 💡 引数としてrepoを受け取る
	var req userPkg.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	user, err := userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err := userPkg.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Failed to sign JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// helloHandler はシンプルなヘルスチェックエンドポイントです。
func helloHandler(c *gin.Context) { // 💡 引数はGin Contextのみ
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Go Backend!"})
}

// dbCheckHandler はデータベース接続の健全性を確認します。
func dbCheckHandler(c *gin.Context, db *sql.DB) { // 💡 引数としてdbを受け取る
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

// AuthMiddleware はJWTトークンを検証し、ユーザー情報をコンテキストに設定するミドルウェアです。
func AuthMiddleware() gin.HandlerFunc { // 💡 引数はGin Contextのみ
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		if len(tokenString) < 7 || tokenString[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}
		tokenString = tokenString[7:]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})
		if err != nil {
			log.Printf("JWT parse error: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user_id"].(float64)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in token claims"})
				c.Abort()
				return
			}
			userEmail, ok := claims["email"].(string)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found in token claims"})
				c.Abort()
				return
			}
			userRole, ok := claims["role"].(string)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in token claims"})
				c.Abort()
				return
			}
			c.Set("user_id", int(userID))
			c.Set("user_email", userEmail)
			c.Set("user_role", userRole)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

// ProtectedHandler は認証ミドルウェアのテストで使用されるダミーの保護されたエンドポイントです。
func ProtectedHandler(c *gin.Context) { // 💡 引数はGin Contextのみ
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

// InitJWTSecretForTest はテスト用にjwtSecretを初期化します。
// main_test.go から呼び出されることを想定しています。
func InitJWTSecretForTest() {
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
}


func main() {
	err := godotenv.Load("../../../.env")
	if err != nil {
		log.Printf("Error loading .env file (this is fine if using explicit env vars): %v", err)
	}
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("Fatal: JWT_SECRET environment variable is not set. Please set it in your .env file.")
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))

	initDB()
	todoRepo = todoPkg.NewRepository(db)
	userRepo = userPkg.NewRepository(db)

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
	authorized.Use(AuthMiddleware())
	{
		authorized.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, todoRepo) })
		authorized.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, todoRepo) })
		authorized.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, todoRepo) })
		authorized.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, todoRepo) })
		authorized.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, todoRepo) })
		authorized.GET("/api/protected", ProtectedHandler) // テスト用
	}

	log.Println("Server listening on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
