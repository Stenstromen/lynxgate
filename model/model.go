package model

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
