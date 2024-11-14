package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stenstromen/lynxgate/model"
)

func (db *DB) ValidateToken(providedToken string) (int, error) {
	if time.Now().Day() == 1 {
		resetQuotaUsageSQL := `
        UPDATE token
        SET quota_usage = 0
        WHERE token = AES_ENCRYPT(?, ?);`

		if _, err := db.Conn.Exec(resetQuotaUsageSQL, providedToken, encryptionKey); err != nil {
			return 1, fmt.Errorf("failed to reset quota usage: %v", err)
		}
	}

	selectTokenSQL := `
    SELECT AES_DECRYPT(token, ?), quota, quota_usage
    FROM token
    WHERE token = AES_ENCRYPT(?, ?);`

	var decryptedToken string
	var quota, quotaUsage int

	err := db.Conn.QueryRow(selectTokenSQL, encryptionKey, providedToken, encryptionKey).Scan(&decryptedToken, &quota, &quotaUsage)
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

	if _, err := db.Conn.Exec(updateQuotaUsageSQL, providedToken, encryptionKey); err != nil {
		return 1, fmt.Errorf("failed to update quota usage: %v", err)
	}

	return 0, nil
}

func (db *DB) DeleteToken(accountID string) error {
	deleteTokenSQL := `
	DELETE FROM token
	WHERE account_id = ?;`

	_, err := db.Conn.Exec(deleteTokenSQL, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %v", err)
	}

	return nil
}

func (db *DB) GetTokens() ([]model.Token, error) {
	selectTokensSQL := `
	SELECT account_id, COALESCE(AES_DECRYPT(token, ?), ''), quota, quota_usage
	FROM token;`

	rows, err := db.Conn.Query(selectTokensSQL, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query tokens: %v", err)
	}
	defer rows.Close()

	var tokens []model.Token
	for rows.Next() {
		var accountID, decryptedToken string
		var quota, quotaUsage int

		if err := rows.Scan(&accountID, &decryptedToken, &quota, &quotaUsage); err != nil {
			return nil, fmt.Errorf("failed to scan token: %v", err)
		}

		tokens = append(tokens, model.Token{
			AccountID:  accountID,
			Token:      decryptedToken,
			Quota:      quota,
			QuotaUsage: quotaUsage,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	return tokens, nil
}

func (db *DB) GetToken(account_id string) model.Token {
	selectTokenSQL := `
	SELECT AES_DECRYPT(token, ?), quota, quota_usage
	FROM token
	WHERE account_id = ?;`

	var decryptedToken string
	var quota, quotaUsage int

	err := db.Conn.QueryRow(selectTokenSQL, encryptionKey, account_id).Scan(&decryptedToken, &quota, &quotaUsage)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Token{}
		}

		log.Fatalf("failed to query token: %v", err)
	}

	return model.Token{AccountID: account_id, Token: decryptedToken, Quota: quota, QuotaUsage: quotaUsage}
}

func generateToken() string {
	token := uuid.New().String()
	token = strings.Replace(token, "-", "", -1)
	return token
}

func (db *DB) CreateToken(accountID string, quota int) (model.User, error) {
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
	err := db.Conn.QueryRow(selectTokenSQL, accountID).Scan(&count)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to query token: %v", err)
	}

	if count > 0 {
		return model.User{}, fmt.Errorf("AccountID '%v' already exists", accountID)
	}

	insertTokenSQL := `
    INSERT INTO token (account_id, token, quota, quota_usage)
    VALUES (?, AES_ENCRYPT(?, ?), ?, ?);`

	_, err = db.Conn.Exec(insertTokenSQL, accountID, token, encryptionKey, quota, 0)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to insert token: %v", err)
	}

	return model.User{AccountID: accountID, Token: token, Quota: quota}, nil
}
