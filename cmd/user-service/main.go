package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = map[string]User{
	"1": {ID: "1", Name: "Alice Johnson", Email: "alice@example.com"},
	"2": {ID: "2", Name: "Bob Smith", Email: "bob@example.com"},
	"3": {ID: "3", Name: "Charlie Brown", Email: "charlie@example.com"},
}

func main() {
	r := chi.NewRouter()

	// GET /users — return all users
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "user-service")
		json.NewEncoder(w).Encode(users)
	})

	// GET /users/{id} — return one user
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		user, ok := users[id]
		if !ok {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "user-service")
		json.NewEncoder(w).Encode(user)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "user-service",
		})
	})

	log.Println("User service starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
