// backend/cmd/api/main.go
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"go-next-todo/backend/internal/todo"
	"go-next-todo/backend/internal/user"
)

    func main() {
    	err := godotenv.Load("../../../.env")
    	if err != nil {
    		log.Printf("Error loading .env file (this is fine if using explicit env vars): %v", err)
    	}
    	if os.Getenv("JWT_SECRET") == "" {
    		log.Fatal("Fatal: JWT_SECRET environment variable is not set. Please set it in your .env file.")
    	}

    	initJWTSecret() // jwt.go で定義された関数を呼び出す

    	db := initDB() // database.go で定義された関数を呼び出す
    	defer db.Close()

    	todoRepo := todo.NewRepository(db)
    	userRepo := user.NewRepository(db)

    	// router.go で定義された setupRouter を呼び出す
    	router := setupRouter(db, todoRepo, userRepo)

    	log.Println("Server listening on port 8080...")
    	if err := router.Run(":8080"); err != nil {
    		log.Fatal(err)
    	}
    }
