package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"

	"github.com/SOHAR-18/gogate/internal/auth"
	"github.com/SOHAR-18/gogate/internal/config"
	"github.com/SOHAR-18/gogate/internal/proxy"
	"github.com/SOHAR-18/gogate/internal/ratelimit"
)

func main() {
	cfg := config.Load()

	routesData, err := os.ReadFile("configs/routes.yaml")
	if err != nil {
		log.Fatalf("Failed to read routes config: %v", err)
	}

	var routesConfig proxy.RoutesConfig
	if err := yaml.Unmarshal(routesData, &routesConfig); err != nil {
		log.Fatalf("Failed to parse routes config: %v", err)
	}

	rp, err := proxy.NewReverseProxy(routesConfig)
	if err != nil {
		log.Fatalf("Failed to create reverse proxy: %v", err)
	}

	authMiddleware := auth.NewAuthMiddleware(
		cfg.JWTSecret,
		cfg.RedisHost,
		cfg.RedisPort,
		cfg.RedisPassword,
	)

	limiter := ratelimit.NewLimiter(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "gogate",
			"port":    cfg.GatewayPort,
			"routes":  rp.GetRoutes(),
		})
	})

	r.Post("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GenerateToken(
			"user-1", "test@example.com", "user", cfg.JWTSecret)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"type":  "Bearer",
		})
	})

	r.Get("/admin/instances", func(w http.ResponseWriter, r *http.Request) {
		result := map[string]interface{}{}
		for _, route := range routesConfig.Routes {
			lb := rp.GetBalancer(route.Path)
			if lb == nil {
				continue
			}
			instances := []map[string]interface{}{}
			for _, inst := range lb.GetAll() {
				instances = append(instances, map[string]interface{}{
					"url":     inst.RawURL,
					"healthy": inst.IsHealthy(),
				})
			}
			result[route.Path] = instances
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	for _, route := range routesConfig.Routes {
		path := route.Path
		rl := ratelimit.NewMiddleware(limiter, route.RateLimit, route.RateWindow)
		handler := rl.Limit(rp.Handler(path))

		if route.Protected {
			handler = authMiddleware.Protect(handler)
		}

		r.Mount(path, handler)
		log.Printf("Mounted route: %s (protected=%v, limit=%d/%ds, upstreams=%d)",
			path, route.Protected, route.RateLimit, route.RateWindow, len(route.Upstreams))
	}

	log.Printf("GoGate starting on :%s", cfg.GatewayPort)
	log.Fatal(http.ListenAndServe(":"+cfg.GatewayPort, r))
}
