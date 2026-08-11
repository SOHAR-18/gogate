package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"

	"github.com/SOHAR-18/gogate/internal/admin"
	"github.com/SOHAR-18/gogate/internal/auth"
	"github.com/SOHAR-18/gogate/internal/config"
	"github.com/SOHAR-18/gogate/internal/discovery"
	"github.com/SOHAR-18/gogate/internal/proxy"
	"github.com/SOHAR-18/gogate/internal/ratelimit"
	"github.com/SOHAR-18/gogate/pkg/metrics"
	"github.com/SOHAR-18/gogate/pkg/tracing"
)

func main() {
	// Load application configuration
	cfg := config.Load()

	// Initialize tracing
	shutdown, err := tracing.Init("gogate", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("[WARNING] Tracing unavailable: %v", err)
	} else {
		defer shutdown()
	}

	// Load routes configuration
	routesData, err := os.ReadFile("configs/routes.yaml")
	if err != nil {
		log.Fatalf("Failed to read routes config: %v", err)
	}

	var routesConfig proxy.RoutesConfig
	if err := yaml.Unmarshal(routesData, &routesConfig); err != nil {
		log.Fatalf("Failed to parse routes config: %v", err)
	}

	// Create reverse proxy
	rp, err := proxy.NewReverseProxy(routesConfig)
	if err != nil {
		log.Fatalf("Failed to create reverse proxy: %v", err)
	}

	// Connect to etcd and watch services
	ctx := context.Background()

	etcdEndpoints := strings.Split(cfg.EtcdEndpoints, ",")

	etcdClient, err := discovery.NewClient(etcdEndpoints)
	if err != nil {
		log.Printf("[WARNING] etcd unavailable: %v", err)
	} else {
		registry := discovery.NewRegistry(etcdClient)

		for _, route := range routesConfig.Routes {
			lb := rp.GetBalancer(route.Path)

			if lb != nil {
				registry.Watch(ctx, route.ServiceName, lb)
				log.Printf("[DISCOVERY] Watching service: %s", route.ServiceName)
			}
		}
	}

	// Authentication middleware
	authMiddleware := auth.NewAuthMiddleware(
		cfg.JWTSecret,
		cfg.RedisHost,
		cfg.RedisPort,
		cfg.RedisPassword,
	)

	// Rate limiter
	limiter := ratelimit.NewLimiter(
		cfg.RedisHost,
		cfg.RedisPort,
		cfg.RedisPassword,
	)

	// Admin handler
	adminHandler := admin.NewHandler(
		&routesConfig,
		rp.GetAllBalancers(),
		rp.GetCircuitBreakerManager(),
		cfg.AdminAPIKey,
		rp,
	)

	// Create router
	r := chi.NewRouter()

	// Common middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)
	r.Use(metrics.Middleware)
	r.Use(tracing.Middleware)

	// Prometheus metrics
	r.Get("/metrics", metrics.Handler().ServeHTTP)

	// Gateway health endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "gogate",
			"port":    cfg.GatewayPort,
			"routes":  rp.GetRoutes(),
		})
	})

	// Generate JWT token
	r.Post("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GenerateToken(
			"user-1",
			"test@example.com",
			"user",
			cfg.JWTSecret,
		)

		if err != nil {
			http.Error(
				w,
				"failed to generate token",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"type":  "Bearer",
		})
	})

	// Admin endpoints
	r.Route("/admin", func(r chi.Router) {

		// Dashboard
		r.Get("/", adminHandler.Dashboard)
		r.Get("/dashboard", adminHandler.Dashboard)

		// Admin information APIs
		r.Get("/health", adminHandler.GetHealth)
		r.Get("/routes", adminHandler.GetRoutes)
		r.Get("/instances", adminHandler.GetInstances)
		r.Get("/discovery", adminHandler.GetDiscovery)
		r.Get("/circuit-breakers", adminHandler.GetCircuitBreakers)

		// Protected admin operations
		r.With(adminHandler.AuthMiddleware).
			Post("/drain/{path}", adminHandler.DrainInstance)

		r.With(adminHandler.AuthMiddleware).
			Post("/enable/{path}", adminHandler.EnableInstance)

		r.With(adminHandler.AuthMiddleware).
			Post("/reset/{path}", adminHandler.ResetCircuitBreaker)
		r.With(adminHandler.AuthMiddleware).Post("/routes", adminHandler.AddRoute)
		r.With(adminHandler.AuthMiddleware).Delete("/routes/{path}", adminHandler.DeleteRoute)
	})

	// Mount all configured routes
	for _, route := range routesConfig.Routes {

		path := route.Path

		// Rate limiting
		rl := ratelimit.NewMiddleware(
			limiter,
			route.RateLimit,
			route.RateWindow,
		)

		handler := rl.Limit(rp.Handler(path))

		// JWT protection for protected routes
		if route.Protected {
			handler = authMiddleware.Protect(handler)
		}

		r.Mount(path, handler)

		log.Printf(
			"Mounted route: %s (protected=%v, limit=%d/%ds)",
			path,
			route.Protected,
			route.RateLimit,
			route.RateWindow,
		)
	}

	// Start gateway
	log.Printf("GoGate starting on :%s", cfg.GatewayPort)

	log.Fatal(
		http.ListenAndServe(
			":"+cfg.GatewayPort,
			r,
		),
	)
}
