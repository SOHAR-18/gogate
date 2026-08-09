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

type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

var products = map[string]Product{
	"1": {ID: "1", Name: "Laptop", Price: 1299.99},
	"2": {ID: "2", Name: "Mouse", Price: 29.99},
	"3": {ID: "3", Name: "Keyboard", Price: 99.99},
}

func main() {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceURL := os.Getenv("SERVICE_URL")

	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}

	if serviceName == "" {
		serviceName = "product-service"
	}

	if serviceURL == "" {
		serviceURL = "http://localhost:8082"
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

		defer etcdClient.Deregister(
			context.Background(),
			serviceName,
			serviceURL,
		)
	}

	r := chi.NewRouter()

	r.Get("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "product-service")
		json.NewEncoder(w).Encode(products)
	})

	r.Get("/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		product, ok := products[id]
		if !ok {
			http.Error(
				w,
				`{"error":"product not found"}`,
				http.StatusNotFound,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "product-service")
		json.NewEncoder(w).Encode(product)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "product-service",
		})
	})

	go func() {
		sig := make(chan os.Signal, 1)

		signal.Notify(
			sig,
			syscall.SIGTERM,
			syscall.SIGINT,
		)

		<-sig

		log.Println("Shutting down product-service...")
		cancel()
	}()

	log.Println("Product service starting on :8082")

	log.Fatal(
		http.ListenAndServe(":8082", r),
	)
}
