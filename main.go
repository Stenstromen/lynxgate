package main

import (
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/stenstromen/lynxgate/api"
	"github.com/stenstromen/lynxgate/db"
)

func scheduleQuotaReset(db *db.DB) {
	go func() {
		for {
			now := time.Now()
			nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			timeUntilReset := nextMonth.Sub(now)

			time.Sleep(timeUntilReset)

			resetSQL := `UPDATE token SET quota_usage = 0;`
			if _, err := db.Conn.Exec(resetSQL); err != nil {
				log.Printf("Failed to reset quotas: %v", err)
			}
		}
	}()
}

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN environment variable is not set")
	}

	dbInstance, err := db.New(dsn)
	if err != nil {
		log.Fatalf("Failed to create DB instance: %v", err)
	}

	scheduleQuotaReset(dbInstance)

	if err := dbInstance.InitializeDB(); err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}

	handler := api.NewHandler(dbInstance)
	port := ":8080"
	log.Printf("Server starting on port %s\n", port)

	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
