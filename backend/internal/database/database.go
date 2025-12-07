package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// GetDSN は環境変数からMySQL接続文字列 (DSN) を構築します。
func GetDSN() string {
	// main.go で godotenv.Load() が呼び出されるため、ここでは省略
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
}

// InitDB はデータベース接続を初期化します。
func InitDB() *sql.DB {
	dsn := GetDSN()
	db, err := sql.Open("mysql", dsn)
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
	return db
}
