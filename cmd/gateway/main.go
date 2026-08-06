package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"

	"github.com/SOHAR-18/gogate/internal/config"
	"github.com/SOHAR-18/gogate/internal/proxy"
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "gogate",
			"port":    cfg.GatewayPort,
			"routes":  rp.GetRoutes(),
		})
	})

	r.Mount("/users", rp.Handler("/users"))
	r.Mount("/products", rp.Handler("/products"))
	r.Mount("/orders", rp.Handler("/orders"))

	log.Printf("GoGate starting on :%s", cfg.GatewayPort)
	log.Printf("Loaded %d routes", len(rp.GetRoutes()))
	log.Fatal(http.ListenAndServe(":"+cfg.GatewayPort, r))
}
