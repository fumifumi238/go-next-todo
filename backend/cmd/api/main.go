// backend/cmd/api/main.go
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"go-next-todo/backend/internal/database"
	"go-next-todo/backend/internal/routes"
)

func main() {
	err := godotenv.Load("../../../.env")
	if err != nil {
		log.Printf("Error loading .env file (this is fine if using explicit env vars): %v", err)
	}
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("Fatal: JWT_SECRET environment variable is not set. Please set it in your .env file.")
	}

	db := database.InitDB()
	defer db.Close()

	router := routes.SetupRouter(db)

	log.Println("Server listening on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
