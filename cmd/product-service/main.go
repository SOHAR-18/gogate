package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

var products = map[string]Product{
	"1": {ID: "1", Name: "Laptop Pro", Price: 1299.99, Category: "Electronics"},
	"2": {ID: "2", Name: "Wireless Mouse", Price: 29.99, Category: "Electronics"},
	"3": {ID: "3", Name: "Standing Desk", Price: 499.99, Category: "Furniture"},
}

func main() {
	r := chi.NewRouter()

	// GET /products — return all products
	r.Get("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "product-service")
		json.NewEncoder(w).Encode(products)
	})

	// GET /products/{id} — return one product
	r.Get("/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		product, ok := products[id]
		if !ok {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "product-service")
		json.NewEncoder(w).Encode(product)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "product-service",
		})
	})

	log.Println("Product service starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", r))
}
