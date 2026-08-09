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
	// Service discovery configuration
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceURL := os.Getenv("SERVICE_URL")

	// Defaults for local development
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}

	if serviceName == "" {
		serviceName = "user-service"
	}

	if serviceURL == "" {
		serviceURL = "http://localhost:8081"
	}

	// Create context for service lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to etcd and register service
	etcdClient, err := discovery.NewClient(strings.Split(etcdEndpoints, ","))
	if err != nil {
		log.Printf("[WARNING] Could not connect to etcd: %v", err)
	} else {
		if err := etcdClient.Register(ctx, serviceName, serviceURL); err != nil {
			log.Printf("[WARNING] Could not register with etcd: %v", err)
		} else {
			log.Printf("[DISCOVERY] Registered %s -> %s", serviceName, serviceURL)
		}

		defer etcdClient.Deregister(
			context.Background(),
			serviceName,
			serviceURL,
		)
	}

	// Create router
	r := chi.NewRouter()

	// GET /users - return all users
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "user-service")

		json.NewEncoder(w).Encode(users)
	})

	// GET /users/{id} - return one user
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		user, ok := users[id]
		if !ok {
			http.Error(
				w,
				`{"error":"user not found"}`,
				http.StatusNotFound,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "user-service")

		json.NewEncoder(w).Encode(user)
	})

	// GET /health - health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "user-service",
		})
	})

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)

		signal.Notify(
			sig,
			syscall.SIGTERM,
			syscall.SIGINT,
		)

		<-sig

		log.Println("Shutting down user-service...")
		cancel()
	}()

	// Start HTTP server
	log.Println("User service starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
