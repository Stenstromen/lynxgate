package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type User struct {
	AccountID string
	Token     string
	Quota     int
}

type Token struct {
	AccountID  string `json:"account_id"`
	Token      string `json:"token"`
	Quota      int    `json:"quota"`
	QuotaUsage int    `json:"quota_usage"`
}

type TokenRequest struct {
	AccountID string `json:"accountID"`
	Quota     int    `json:"quota"`
}

var db *sql.DB
var encryptionKey string

func main() {
	if err := initializeDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mux.HandleFunc("GET /validate", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header is required", http.StatusBadRequest)
			return
		}

		valid, err := validateToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if valid == 2 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if valid == 1 {
			http.Error(w, "Limit exceeded", http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /tokens", func(w http.ResponseWriter, r *http.Request) {
		tokens := getTokens()

		w.Header().Set("Content-Type", "application/json")
		response, err := json.Marshal(tokens)
		if err != nil {
			http.Error(w, "Failed to marshal tokens", http.StatusInternalServerError)
			return
		}

		w.Write(response)
	})

	mux.HandleFunc("/tokens/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		token := getToken(id)

		if token.AccountID == "" {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response, err := json.Marshal(token)
		if err != nil {
			http.Error(w, "Failed to marshal token", http.StatusInternalServerError)
			return
		}

		w.Write(response)
	})

	mux.HandleFunc("DELETE /tokens/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := deleteToken(id)
		if err != nil {
			http.Error(w, "Failed to delete token", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /tokens", func(w http.ResponseWriter, r *http.Request) {
		var req TokenRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := createToken(req.AccountID, req.Quota)
		if err != nil {
			fmt.Print(err)
			http.Error(w, "Failed to create token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{"account_id": "%s", "token": "%s", "quota": %d}`, user.AccountID, user.Token, user.Quota)
		w.Write([]byte(response))
	})

	http.ListenAndServe(":8080", mux)
}

func validateToken(providedToken string) (int, error) {
	selectTokenSQL := `
    SELECT AES_DECRYPT(token, ?), quota, quota_usage
    FROM token
    WHERE token = AES_ENCRYPT(?, ?);`

	var decryptedToken string
	var quota, quotaUsage int

	err := db.QueryRow(selectTokenSQL, encryptionKey, providedToken, encryptionKey).Scan(&decryptedToken, &quota, &quotaUsage)
	if err != nil {
		if err == sql.ErrNoRows {
			return 2, nil
		}

		return 2, fmt.Errorf("failed to query token: %v", err)
	}

	if quota == 0 {
		return 0, nil
	}

	if quotaUsage >= quota {
		return 1, nil
	}

	updateQuotaUsageSQL := `
    UPDATE token
    SET quota_usage = quota_usage + 1
    WHERE token = AES_ENCRYPT(?, ?);`

	if _, err := db.Exec(updateQuotaUsageSQL, providedToken, encryptionKey); err != nil {
		return 1, fmt.Errorf("failed to update quota usage: %v", err)
	}

	return 0, nil
}

func deleteToken(accountID string) error {
	deleteTokenSQL := `
	DELETE FROM token
	WHERE account_id = ?;`

	_, err := db.Exec(deleteTokenSQL, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %v", err)
	}

	return nil
}

func getTokens() []Token {
	selectTokensSQL := `
	SELECT account_id, AES_DECRYPT(token, ?), quota, quota_usage
	FROM token;`

	rows, err := db.Query(selectTokensSQL, encryptionKey)
	if err != nil {
		log.Fatalf("failed to query tokens: %v", err)
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var accountID, decryptedToken string
		var quota, quotaUsage int

		if err := rows.Scan(&accountID, &decryptedToken, &quota, &quotaUsage); err != nil {
			log.Fatalf("failed to scan token: %v", err)
		}

		tokens = append(tokens, Token{AccountID: accountID, Token: decryptedToken, Quota: quota, QuotaUsage: quotaUsage})
	}

	return tokens
}

func getToken(account_id string) Token {
	selectTokenSQL := `
	SELECT AES_DECRYPT(token, ?), quota, quota_usage
	FROM token
	WHERE account_id = ?;`

	var decryptedToken string
	var quota, quotaUsage int

	err := db.QueryRow(selectTokenSQL, encryptionKey, account_id).Scan(&decryptedToken, &quota, &quotaUsage)
	if err != nil {
		if err == sql.ErrNoRows {
			return Token{}
		}

		log.Fatalf("failed to query token: %v", err)
	}

	return Token{AccountID: account_id, Token: decryptedToken, Quota: quota, QuotaUsage: quotaUsage}
}

func generateToken() string {
	token := uuid.New().String()
	token = strings.Replace(token, "-", "", -1)
	return token
}

func createToken(accountID string, quota int) (User, error) {
	token := generateToken()
	encryptionKey := os.Getenv("MYSQL_ENCRYPTION_KEY")

	if encryptionKey == "" {
		log.Fatal("Encryption key is not set")
	}

	selectTokenSQL := `
	SELECT COUNT(*)
	FROM token
	WHERE account_id = ?;`

	var count int
	err := db.QueryRow(selectTokenSQL, accountID).Scan(&count)
	if err != nil {
		return User{}, fmt.Errorf("failed to query token: %v", err)
	}

	if count > 0 {
		return User{}, fmt.Errorf("AccountID '%v' already exists", accountID)
	}

	insertTokenSQL := `
    INSERT INTO token (account_id, token, quota, quota_usage)
    VALUES (?, AES_ENCRYPT(?, ?), ?, ?);`

	_, err = db.Exec(insertTokenSQL, accountID, token, encryptionKey, quota, 0)
	if err != nil {
		return User{}, fmt.Errorf("failed to insert token: %v", err)
	}

	return User{AccountID: accountID, Token: token, Quota: quota}, nil
}

func initializeDB() error {
	encryptionKey = os.Getenv("MYSQL_ENCRYPTION_KEY")
	if encryptionKey == "" {
		fmt.Errorf("encryption key not set")
	}
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return fmt.Errorf("MYSQL_DSN environment variable is not set")
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS token (
        id INT AUTO_INCREMENT PRIMARY KEY,
        account_id VARCHAR(255) NOT NULL,
        token VARBINARY(256) NOT NULL,
        quota INT NOT NULL,
        quota_usage INT NOT NULL
    );`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create token table: %v", err)
	}

	return nil
}
