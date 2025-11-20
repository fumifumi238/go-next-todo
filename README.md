# how to use

## gitの設定

```
git init
```

```
touch .gitignore
```
## .gitignoreの設定(必要なものだけ抽出する)
```.gitignore
# --- OS files ---
.DS_Store
Thumbs.db

# --- Editor / IDE ---
.vscode/
.idea/
*.swp
*.swo

# --- Logs ---
logs/
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*

# --- Dependencies ---
node_modules/
vendor/
dist/
build/
out/

# --- Environment / Secrets ---
.env
.env.*.local
*.pem
*.key

# --- Cache / Temp ---
.tmp/
temp/
*.tmp
.cache/
coverage/

# --- Runtime files ---
*.pid
*.seed
*.pid.lock

# --- Compiled code ---
*.class
*.dll
*.exe
*.o
*.so

# --- Python ---
__pycache__/
*.pyc
*.pyo
*.pyd
*.egg-info/

# --- Go ---
bin/
*.exe
*.test

# --- Java ---
target/
*.iml

# --- Node / Next.js ---
.next/
.vercel/

# --- Docker ---
*.pid
*.log
docker-compose.override.yml

# --- Database ---
*.sqlite3
*.db

# --- Misc ---
*.orig

```
## backendの初期設定


```
mkdir backend
cd backend
```

```
go mod init go-next-to-do/backend

```

```
mkdir cmd internal pkg
mkdir cmd/api
touch main.go
```

main.goを記述
```go:main.go
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
		"message": "Hello from Go Backend (Gin)!",
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


```

パッケージを反映

```
go mod tidy
```

## frontendの初期設定

### frontendディレクトリを作成、typescript、eslint、npmを使用(他はご自由に)
```
cd ../
npx create-next-app@latest frontend --ts --eslint --use-npm
```
## Next13以上の場合以下のように設定
```Next.js:frontend/app/page.tsx
// app/page.tsx

// 環境変数 NEXT_PUBLIC_API_URL を利用
// サーバーコンポーネントで実行される場合は 'http://backend:8080' を使うのが確実
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function getData() {
  const res = await fetch(`${API_URL}/api/dbcheck`, {
    // サーバーコンポーネントでのデータキャッシュ設定 (必要に応じて)
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch data: ${res.statusText}`);
  }

  // GinのハンドラーはJSONを返しているため、.json()でパース
  return res.json();
}

export default async function Page() {
  const data = await getData();

  return (
    <div>
      <h1>CORS Test Page (Gin Backend)</h1>
      <p>Backend Response:</p>
      {/* 取得したJSONデータを表示 */}
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}

```

## Dockerfileの作成
```
touch backend/Dockerfile frontend/Dockerfile
```

```backend/Dockerfile
FROM golang:1.25.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
<!-- RUN ls -R /app -->
RUN go build -v -o server ./cmd/api
<!-- RUN ls -al /app  # 確認用 -->

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .
<!-- RUN ls -al /app  # 確認用 -->

RUN chmod +x server
CMD ["./server"]

```

```frontend/Dockerfile
# ---- Build Stage ----
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

# Next.js 本番ビルド
RUN npm run build

# ---- Run Stage ----
FROM node:20-alpine

WORKDIR /app

# 本番実行に必要なファイルのみコピー
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3000
CMD ["npm", "start"]

```

## docker-compose.ymlの記述
```
touch docker-compose.yml
```

```docker-compose.yml
version: '3.9'

services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: go-backend
    ports:
      - "8080:8080"
    environment:
        DB_HOST: ${DB_HOST}
        DB_PORT: ${DB_PORT}
        DB_USER: ${DB_USER}
        DB_PASS: ${DB_PASS}
        DB_NAME: ${DB_NAME}
    depends_on:
      - db
    # volumes:
    #   - ./backend:/app

    <!-- ./backend:/appをなくさないとserverが起動しなかったので -->

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: nextjs-frontend
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: "http://backend:8080"
    volumes:
      - ./frontend:/app
      - /app/node_modules
<!-- .nextが見つからなかったので追加 -->
      - /app/.next
    depends_on:
      - backend

```

## .envの記述(gitにpushしないように!!!!)

```
touch .env
```

```.env

<!-- dbはdocker-compose.ymlのdbのサービス名、mysqlとdbの名前は合わせておく -->
DB_HOST=db
DB_PORT=3306
DB_USER=your_user
DB_PASS=user_pass
DB_NAME=your_db

MYSQL_DATABASE=your_db
MYSQL_ROOT_PASSWORD=rootpass
MYSQL_USER=your_user
MYSQL_PASSWORD=user_pass
```
## 起動
```
docker-compose up --build
```
