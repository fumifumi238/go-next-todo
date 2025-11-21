package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Ginルーターをセットアップするヘルパー関数
func setupRouter() *gin.Engine {
	// テスト中はリリースモードでログを最小限にする
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	// main.go のCORS設定はテストでは不要だが、念のためダミーで設定可能
	return r
}

// ----------------------------------------------------
// Step 1: ToDoタスクの追加 (POST /api/todos)
// ----------------------------------------------------

func TestCreateTodo_Success(t *testing.T) {
	// Arrange: ルーターとリクエストボディの準備
	r := setupRouter()

	// 💡 データベース接続部分をモック化または一時的な設定にする必要がありますが、
	// TDDの初期段階では、まずルーティングとJSON処理ができるかを確認します。
	// ここではDB接続は一時的にスキップし、ダミーの処理を呼び出します。

	// ダミーのPOSTルートを設定（まだ実装していない関数を呼ぶ）
	r.POST("/api/todos", func(c *gin.Context) {
		// 🚨 実際のアプリケーションコードで実装される部分
		// ここではテストを通すためのダミーコードを実装しません。
		// 本来のアプリケーションロジックを呼び出すことを想定します。
	})

	// ユーザーが送信するデータ
	newTodo := map[string]string{"title": "新しいタスクをテスト", "memo": "テストメモ"}
	jsonValue, _ := json.Marshal(newTodo)

	// HTTPリクエストの作成 (POST /api/todos)
	req, _ := http.NewRequest("POST", "/api/todos", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	// レコーダーの作成
	w := httptest.NewRecorder()

	// Act: リクエストを実行
	r.ServeHTTP(w, req)

	// Assert: 結果の検証
	// 期待値: ステータスコード 201 Created
	assert.Equal(t, http.StatusCreated, w.Code, "Expected HTTP Status Code 201 Created")

	// 期待値: レスポンスボディに、作成されたタスクの情報（IDなど）が含まれる
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be valid JSON")

	// 簡略化のため、ここでは title がレスポンスに含まれているかのみ確認
	assert.Contains(t, response, "title", "Response should contain the 'title' of the created todo")
	assert.Equal(t, newTodo["title"], response["title"], "Response title should match the request title")
}
