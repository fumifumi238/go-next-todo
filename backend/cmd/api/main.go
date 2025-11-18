package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Response構造体
type HelloResponse struct {
	Message string `json:"message"`
	Service string `json:"service"`
}

func main() {
	// Ginのデフォルトルーターを作成
	r := gin.Default()

	// CORS設定
	// 開発環境では全てのOriginからのアクセスを一時的に許可します
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Preflightリクエスト(OPTIONSメソッド)への対応
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 接続確認用エンドポイント
	r.GET("/api/hello", func(c *gin.Context) {
		// Go APIが正常に動作していることを示すメッセージ
		response := HelloResponse{
			Message: "Hello from Go API using Gin!",
			Service: "Golang Gin Backend Service (Port 8080)",
		}
		// JSONを返す
		c.JSON(http.StatusOK, response)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Docker Composeのポート
	}

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Go Gin API starting on port %s...\n", port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}

// 💡 必要なパッケージのインストール (backendディレクトリで実行):
// go mod tidy
// go get github.com/gin-gonic/gin
