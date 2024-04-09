package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/stenstromen/lynxgate/api"
	"github.com/stenstromen/lynxgate/db"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN environment variable is not set")
	}

	dbInstance, err := db.New(dsn)
	if err != nil {
		log.Fatalf("Failed to create DB instance: %v", err)
	}

	if err := dbInstance.InitializeDB(); err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}

	handler := api.NewHandler(dbInstance)
	port := ":8181"
	log.Printf("Server starting on port %s\n", port)

	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
