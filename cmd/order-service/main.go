package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/SOHAR-18/gogate/internal/discovery"
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
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceURL := os.Getenv("SERVICE_URL")

	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}
	if serviceName == "" {
		serviceName = "order-service"
	}
	if serviceURL == "" {
		serviceURL = "http://localhost:8083"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	etcdClient, err := discovery.NewClient(strings.Split(etcdEndpoints, ","))
	if err != nil {
		log.Printf("[WARNING] Could not connect to etcd: %v", err)
	} else {
		if err := etcdClient.Register(ctx, serviceName, serviceURL); err != nil {
			log.Printf("[WARNING] Could not register with etcd: %v", err)
		}
		defer etcdClient.Deregister(context.Background(), serviceName, serviceURL)
	}

	r := chi.NewRouter()

	r.Get("/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "order-service")
		json.NewEncoder(w).Encode(orders)
	})

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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "order-service",
		})
	})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		log.Println("Shutting down order-service...")
		cancel()
	}()

	log.Println("Order service starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", r))
}
