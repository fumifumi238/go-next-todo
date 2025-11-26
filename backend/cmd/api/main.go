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

	todoPkg "go-next-todo/backend/internal/todo" // パッケージエイリアスを使用
	userPkg "go-next-todo/backend/internal/user" // 追加: userパッケージをインポート
)

// DB接続をグローバル（または構造体）に保持するため、db変数とリポジトリ変数を定義
var db *sql.DB
var todoRepo *todoPkg.Repository // パッケージエイリアスを使用
var userRepo *userPkg.Repository // 追加: userリポジトリ変数を定義

// JWT署名のためのシークレットキー
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

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

	// DSN (Data Source Name) 形式に整形
	// parseTime=true: MySQLのDATETIME/TIMESTAMPをtime.Timeに自動変換
	// 例: user:pass@tcp(db:3306)/dbname?parseTime=true
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
}

// initDB はデータベース接続を初期化します。
func initDB() {
	dsn := getDSN()

	// DB接続を開く
	var err error
	// データベースドライバに "mysql" を指定
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Fatal: Failed to open database connection: %v", err)
	}

	// DBへの接続設定（プールサイズや接続時間を設定）
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 実際にDBに接続できているかPingで確認
	if err := db.Ping(); err != nil {
		log.Fatalf("Fatal: Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to MySQL database!")
}

// createTodoHandler は新しいToDoタスクを作成し、DBに保存します。
func createTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) {
	var newTodo todoPkg.Todo // パッケージエイリアスを使用

	// 1. リクエストボディのJSONを構造体にバインド（バリデーションも実行）
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// 2. リポジトリ層を呼び出してDBに挿入
	createdTodo, err := todoRepo.Create(&newTodo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save todo to database", "details": err.Error()})
		return
	}

	// 3. 201 Created ステータスと作成されたオブジェクトを返す
	c.JSON(http.StatusCreated, createdTodo)
}

// getTodoByIDHandler は指定されたIDのToDoタスクを取得します。
func getTodoByIDHandler(c *gin.Context, todoRepo *todoPkg.Repository) {
	// パラメータからIDを取得
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// リポジトリ層を呼び出してDBから取得
	foundTodo, err := todoRepo.FindByID(id)
	if err != nil {
		// TODOが見つからない場合
		if errors.Is(err, todoPkg.ErrTodoNotFound) { // パッケージエイリアスを使用
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo from database", "details": err.Error()})
		return
	}

	// 200 OK ステータスと取得したオブジェクトを返す
	c.JSON(http.StatusOK, foundTodo)
}

// updateTodoHandler は指定されたIDのToDoタスクを更新します。
func updateTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) {
	// パラメータからIDを取得
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// リクエストボディのJSONを構造体にバインド
	var updateTodo todoPkg.Todo // パッケージエイリアスを使用
	if err := c.ShouldBindJSON(&updateTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// リポジトリ層を呼び出してDBを更新
	updatedTodo, err := todoRepo.Update(id, &updateTodo)
	if err != nil {
		// TODOが見つからない場合
		if errors.Is(err, todoPkg.ErrTodoNotFound) { // パッケージエイリアスを使用
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update todo in database", "details": err.Error()})
		return
	}

	// 200 OK ステータスと更新されたオブジェクトを返す
	c.JSON(http.StatusOK, updatedTodo)
}

// deleteTodoHandler は指定されたIDのToDoタスクを削除します。
func deleteTodoHandler(c *gin.Context, todoRepo *todoPkg.Repository) {
	// パラメータからIDを取得
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// リポジトリ層を呼び出してDBから削除
	err = todoRepo.Delete(id)
	if err != nil {
		// TODOが見つからない場合
		if errors.Is(err, todoPkg.ErrTodoNotFound) { // パッケージエイリアスを使用
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo from database", "details": err.Error()})
		return
	}

	// 204 No Content ステータスを返す
	c.Status(http.StatusNoContent)
}

// getTodosHandler はすべてのToDoタスクを取得します。
func getTodosHandler(c *gin.Context, todoRepo *todoPkg.Repository) {
	// リポジトリ層を呼び出してDBから取得
	todos, err := todoRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos from database", "details": err.Error()})
		return
	}

	// 200 OK ステータスと取得したオブジェクトの配列を返す
	c.JSON(http.StatusOK, todos)
}

// registerHandler はユーザー登録ハンドラー
func registerHandler(c *gin.Context, userRepo *userPkg.Repository) {
	var req userPkg.UserRegisterRequest// userPkg.User を使用
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	// 💡 req.Username, req.Email, req.Password を使用
	// バリデーションは ShouldBindJSON の binding タグでカバーされるが、
	// 明示的なチェックも残しておく (特にパスワードのmin=8など)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email, and password are required"})
		return
	}


	// パスワードをハッシュ化
	hashedPassword, err := userPkg.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password", "details": err.Error()})
		return
	}
           // パスワードハッシュが格納されたので、元のパスワードはクリア
	// 新しいユーザーを作成
	newUser := userPkg.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "user", // デフォルトロール
	}

	// ユーザーをデータベースに保存
	createdUser, err := userRepo.Create(&newUser)
	if err != nil {
		// エラーの種類に応じて適切なステータスを返す
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, fmt.Errorf("could not insert user: Error 1062 (23000): Duplicate entry")) { // 重複エントリーのハンドリングを改善
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user", "details": err.Error()})
		return
	}

	// パスワードハッシュはレスポンスに含めない
	createdUser.PasswordHash = ""
	c.JSON(http.StatusCreated, createdUser)
}

// loginHandler はユーザーログインを処理し、成功した場合はJWTを返します。
func loginHandler(c *gin.Context, userRepo *userPkg.Repository) {
	var req userPkg.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// 1. メールアドレスでユーザーを検索
	user, err := userRepo.FindByEmail(req.Email)
	if err != nil {
		// ユーザーが見つからない場合や、その他のDBエラー
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 2. パスワードの検証
	if err := userPkg.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// パスワードが一致しない場合
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 3. JWTの生成
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24時間有効期限
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Failed to sign JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 4. トークンをクライアントに返す
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// helloHandler はシンプルなヘルスチェックエンドポイントです。
func helloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Go Backend!"})
}

// dbCheckHandler はデータベース接続の健全性を確認します。
func dbCheckHandler(c *gin.Context, db *sql.DB) {
	// PingでDB接続をチェック
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

func main() {
	// .env ファイルの読み込み (initDB() の中にもあるが、ここでも安全策として読み込む)
	err := godotenv.Load("../../../.env") // ルート直下の .env を指定
	if err != nil {
		log.Printf("Error loading .env file (this is fine if using explicit env vars): %v", err)
	}

	// JWT_SECRET が設定されているか確認
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("Fatal: JWT_SECRET environment variable is not set. Please set it in your .env file.")
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET")) // 環境変数からシークレットキーをロード

	// 1. データベース接続の初期化
	initDB()

	// 2. リポジトリの初期化
	todoRepo = todoPkg.NewRepository(db) // パッケージエイリアスを使用
	userRepo = userPkg.NewRepository(db) // 追加: userリポジトリの初期化

	r := gin.Default()

	// ------------------------------------
	// 💡 CORS設定をルーターに適用
	// ------------------------------------
	config := cors.DefaultConfig()
	// Next.js (http://localhost:3000) からのアクセスを許可
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	// 認証情報を伴うリクエストのために'Authorization'ヘッダーを許可リストに追加
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"} // 'Authorization'を追加

	r.Use(cors.New(config))

	// ルーティングの設定 (クロージャを使用してリポジトリをハンドラーに注入)
	r.GET("/api/hello", helloHandler)
	r.GET("/api/dbcheck", func(c *gin.Context) { dbCheckHandler(c, db) })
	r.GET("/api/todos", func(c *gin.Context) { getTodosHandler(c, todoRepo) })        // タスク一覧取得
	r.GET("/api/todos/:id", func(c *gin.Context) { getTodoByIDHandler(c, todoRepo) }) // タスク取得（ID指定）
	r.POST("/api/todos", func(c *gin.Context) { createTodoHandler(c, todoRepo) })     // タスク作成
	r.PUT("/api/todos/:id", func(c *gin.Context) { updateTodoHandler(c, todoRepo) })  // タスク更新
	r.DELETE("/api/todos/:id", func(c *gin.Context) { deleteTodoHandler(c, todoRepo) }) // タスク削除

	// 💡 追加: ユーザー関連エンドポイント
	r.POST("/api/register", func(c *gin.Context) { registerHandler(c, userRepo) }) // ユーザー登録
	r.POST("/api/login", func(c *gin.Context) { loginHandler(c, userRepo) })       // ユーザーログイン

	// サーバー起動
	log.Println("Server listening on port 8080...")
	// main関数を抜ける際にDB接続を閉じる (但し、ウェブサーバーなので通常は閉じない)
	// defer db.Close()
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
