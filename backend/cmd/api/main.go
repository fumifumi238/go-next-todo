package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-next-to-do/backend/internal/todo"
)

// DB接続をグローバル（または構造体）に保持するため、db変数を定義
var db *sql.DB
var todoRepo *todo.Repository

// getDSN は環境変数からMySQL接続文字列 (DSN) を構築します。
func getDSN() string {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	// DSN (Data Source Name) 形式に整形
	// 例: user:pass@tcp(db:3306)/dbname
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, name)
}

// ------------------------------------
// 💡 DB接続初期化関数
// ------------------------------------
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
func createTodoHandler(c *gin.Context) {
	var newTodo todo.Todo

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

// ------------------------------------
// 💡 追加: ヘルスチェック用ハンドラー
// ------------------------------------

// helloHandler はシンプルなヘルスチェックエンドポイントです。
func helloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello from Go Backend!"})
}

// dbCheckHandler はデータベース接続の健全性を確認します。
func dbCheckHandler(c *gin.Context) {
	// PingでDB接続をチェック
	if err := db.Ping(); err != nil {
		log.Printf("DB Ping failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"message": "Database connection failed",
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Database connection is healthy"})
}


func main() {
	// 1. データベース接続の初期化
	initDB()

	// 2. リポジトリの初期化
	todoRepo = todo.NewRepository(db)

	r := gin.Default()

	// ------------------------------------
	// 💡 CORS設定をルーターに適用
	// ------------------------------------
	config := cors.DefaultConfig()
	// Next.js (http://localhost:3000) からのアクセスを許可
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}

	r.Use(cors.New(config))

	// ルーティングの設定
	r.GET("/api/hello", helloHandler)
	r.GET("/api/dbcheck", dbCheckHandler)
	r.POST("/api/todos", createTodoHandler) // タスク作成

	// サーバー起動
	log.Println("Server listening on port 8080...")
	// main関数を抜ける際にDB接続を閉じる (但し、ウェブサーバーなので通常は閉じない)
	// defer db.Close()
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
