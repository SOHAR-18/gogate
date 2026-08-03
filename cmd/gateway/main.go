package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SOHAR-18/gogate/internal/config"
)

func main() {
	// Load config from .env
	cfg := config.Load()

	r := chi.NewRouter()

	// Built-in middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check for the gateway itself
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "gateway",
			"port":    cfg.GatewayPort,
		})
	})

	// Route: /users/* → user-service on :8081
	r.Mount("/users", proxyHandler("http://user-service:8081"))

	// Route: /products/* → product-service on :8082
	r.Mount("/products", proxyHandler("http://product-service:8082"))

	// Route: /orders/* → order-service on :8083
	r.Mount("/orders", proxyHandler("http://order-service:8083"))

	log.Printf("GoGate starting on :%s", cfg.GatewayPort)
	log.Fatal(http.ListenAndServe(":"+cfg.GatewayPort, r))
}

// proxyHandler creates a reverse proxy to the given target URL
func proxyHandler(target string) http.Handler {
	url, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Invalid proxy target: %s", target)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "upstream service unavailable",
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add gateway headers
		r.Header.Set("X-Forwarded-By", "gogate")
		proxy.ServeHTTP(w, r)
	})
}
