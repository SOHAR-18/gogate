package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Order struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Status    string  `json:"status"`
	Total     float64 `json:"total"`
}

var orders = map[string]Order{
	"1": {ID: "1", UserID: "1", ProductID: "1", Status: "delivered", Total: 1299.99},
	"2": {ID: "2", UserID: "2", ProductID: "2", Status: "pending", Total: 29.99},
	"3": {ID: "3", UserID: "1", ProductID: "3", Status: "processing", Total: 499.99},
}

func main() {
	r := chi.NewRouter()

	// GET /orders — return all orders
	r.Get("/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "order-service")
		json.NewEncoder(w).Encode(orders)
	})

	// GET /orders/{id} — return one order
	r.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		order, ok := orders[id]
		if !ok {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "order-service")
		json.NewEncoder(w).Encode(order)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "order-service",
		})
	})

	log.Println("Order service starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", r))
}
