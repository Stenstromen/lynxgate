package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stenstromen/lynxgate/db"
	"github.com/stenstromen/lynxgate/model"
)

func NewHandler(db *db.DB) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		err := db.ConnectionCheck()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /validate", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header is required", http.StatusBadRequest)
			return
		}

		valid, err := db.ValidateToken(token)
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
		tokens, err := db.GetTokens()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get tokens: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response, err := json.Marshal(tokens)
		if err != nil {
			http.Error(w, "Failed to marshal tokens", http.StatusInternalServerError)
			return
		}

		w.Write(response)
	})

	mux.HandleFunc("GET /tokens/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		token := db.GetToken(id)

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
		err := db.DeleteToken(id)
		if err != nil {
			http.Error(w, "Failed to delete token", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /tokens", func(w http.ResponseWriter, r *http.Request) {
		var req model.TokenRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := db.CreateToken(req.AccountID, req.Quota)
		if err != nil {
			fmt.Print(err)
			http.Error(w, "Failed to create token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := fmt.Sprintf(`{"account_id": "%s", "token": "%s", "quota": %d}`, user.AccountID, user.Token, user.Quota)
		w.Write([]byte(response))
	})

	return mux
}
