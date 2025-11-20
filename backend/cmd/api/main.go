package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-contrib/cors" // Gin用のCORSライブラリ
	"github.com/gin-gonic/gin"
)

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

func dbCheckHandler(c *gin.Context) {
    dsn := getDSN()

    // 1. DBに接続
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Println("DB接続エラー:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to open DB connection", "error": err.Error()})
        return
    }
    defer db.Close()

    // 2. 接続を検証 (Ping)
    if err := db.Ping(); err != nil {
        log.Println("DB Pingエラー:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to connect to MySQL container", "error": err.Error()})
        return
    }

    // 3. シンプルなクエリを実行
    var result int
    err = db.QueryRow("SELECT 1").Scan(&result)
    if err != nil {
        log.Println("DBクエリ実行エラー:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to execute query", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Database connection successful", "result": result})
}

// Ginのハンドラー関数
func helloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "接続できました",
	})
}

func main() {
	r := gin.Default()

    // ------------------------------------
    // 💡 CORS設定をルーターに適用
    // ------------------------------------
	config := cors.Config{
        // Next.jsのオリジンを設定 (Dockerネットワーク内からのアクセスも考慮)
		AllowOrigins: []string{
            "http://localhost:3000", // ブラウザからのアクセス用
            // "http://frontend:3000", // (オプション) Dockerコンテナからのアクセス用
        },
        // 許可するHTTPメソッド
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        // 許可するヘッダー
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        // 認証情報（Cookieなど）の送信を許可
		AllowCredentials: true,
        // プリフライトリクエストの結果をキャッシュする時間
		MaxAge:           12 * time.Hour,
	}

	r.Use(cors.New(config))
    // ------------------------------------

	// ルーティングの設定
	r.GET("/api/hello", helloHandler)

    // db check
    r.GET("/api/dbcheck", dbCheckHandler)

	// サーバー起動
	log.Println("Server listening on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
