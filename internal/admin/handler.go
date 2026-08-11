package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SOHAR-18/gogate/internal/circuitbreaker"
	"github.com/SOHAR-18/gogate/internal/loadbalancer"
	"github.com/SOHAR-18/gogate/internal/proxy"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	routesConfig *proxy.RoutesConfig
	balancers    map[string]*loadbalancer.RoundRobin
	cbManager    *circuitbreaker.Manager
	apiKey       string
	rp           *proxy.ReverseProxy
}

func NewHandler(
	routesConfig *proxy.RoutesConfig,
	balancers map[string]*loadbalancer.RoundRobin,
	cbManager *circuitbreaker.Manager,
	apiKey string,
	rp *proxy.ReverseProxy,
) *Handler {
	return &Handler{
		routesConfig: routesConfig,
		balancers:    balancers,
		cbManager:    cbManager,
		apiKey:       apiKey,
		rp:           rp,
	}
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Admin-Key")
		if key != h.apiKey {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid admin API key",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) GetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := []map[string]interface{}{}
	for _, route := range h.routesConfig.Routes {
		lb := h.balancers[route.Path]
		instances := []map[string]interface{}{}
		if lb != nil {
			for _, inst := range lb.GetAll() {
				instances = append(instances, map[string]interface{}{
					"url":     inst.RawURL,
					"healthy": inst.IsHealthy(),
				})
			}
		}
		cb := h.cbManager.GetOrCreate(route.Path)
		counts := cb.Counts()
		routes = append(routes, map[string]interface{}{
			"path":        route.Path,
			"service":     route.ServiceName,
			"protected":   route.Protected,
			"rate_limit":  route.RateLimit,
			"rate_window": route.RateWindow,
			"instances":   instances,
			"circuit_breaker": map[string]interface{}{
				"state":                string(cb.State()),
				"consecutive_failures": counts.ConsecutiveFailures,
				"total_requests":       counts.TotalSuccesses + counts.TotalFailures,
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":  len(routes),
		"routes": routes,
	})
}

func (h *Handler) DrainInstance(w http.ResponseWriter, r *http.Request) {
	path := "/" + chi.URLParam(r, "path")
	lb := h.balancers[path]
	if lb == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "route not found: " + path,
		})
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "url is required in request body",
		})
		return
	}

	lb.SetHealthy(body.URL, false)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "instance drained: " + body.URL,
		"path":    path,
	})
}
func (h *Handler) GetDiscovery(w http.ResponseWriter, r *http.Request) {
	result := map[string][]string{}

	for _, route := range h.routesConfig.Routes {
		lb := h.balancers[route.Path]
		if lb == nil {
			result[route.ServiceName] = []string{}
			continue
		}

		instances := []string{}

		for _, inst := range lb.GetAll() {
			instances = append(instances, inst.RawURL)
		}

		result[route.ServiceName] = instances
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) EnableInstance(w http.ResponseWriter, r *http.Request) {
	path := "/" + chi.URLParam(r, "path")
	lb := h.balancers[path]
	if lb == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "route not found: " + path,
		})
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "url is required in request body",
		})
		return
	}

	lb.SetHealthy(body.URL, true)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "instance enabled: " + body.URL,
		"path":    path,
	})
}

/*func (h *Handler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	path := "/" + chi.URLParam(r, "path")
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "circuit breaker noted for: " + path,
		"note":    "breaker will auto-reset after timeout period",
	})
}
*/

func (h *Handler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	path := "/" + chi.URLParam(r, "path")
	newBreaker := h.cbManager.GetOrCreate(path)
	_ = newBreaker
	h.cbManager.Reset(path)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "circuit breaker reset: " + path,
		"state":   "closed",
	})
}

func (h *Handler) AddRoute(w http.ResponseWriter, r *http.Request) {
	var route proxy.Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}
	if route.Path == "" || len(route.Upstreams) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "path and upstreams are required",
		})
		return
	}
	for _, existing := range h.routesConfig.Routes {
		if existing.Path == route.Path {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "route already exists: " + route.Path,
			})
			return
		}
	}
	if err := h.rp.AddRoute(route); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to add route: " + err.Error(),
		})
		return
	}
	h.routesConfig.Routes = append(h.routesConfig.Routes, route)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "route added: " + route.Path,
		"route":   route,
	})
}

func (h *Handler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	path := "/" + chi.URLParam(r, "path")

	// Check whether route exists in configuration.
	found := false
	for _, route := range h.routesConfig.Routes {
		if route.Path == path {
			found = true
			break
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "route not found: " + path,
		})
		return
	}

	// Remove from the actual reverse proxy.
	if err := h.rp.RemoveRoute(path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to remove route: " + err.Error(),
		})
		return
	}

	// Remove from configuration.
	newRoutes := []proxy.Route{}

	for _, route := range h.routesConfig.Routes {
		if route.Path != path {
			newRoutes = append(newRoutes, route)
		}
	}

	h.routesConfig.Routes = newRoutes

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "route deleted: " + path,
	})
}

func (h *Handler) GetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{}
	for name, breaker := range h.cbManager.GetAll() {
		counts := breaker.Counts()
		result[name] = map[string]interface{}{
			"state":                string(breaker.State()),
			"total_requests":       counts.TotalSuccesses + counts.TotalFailures,
			"total_successes":      counts.TotalSuccesses,
			"total_failures":       counts.TotalFailures,
			"consecutive_failures": counts.ConsecutiveFailures,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetInstances(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{}
	for _, route := range h.routesConfig.Routes {
		lb := h.balancers[route.Path]
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
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	healthy := 0
	total := 0
	for _, route := range h.routesConfig.Routes {
		lb := h.balancers[route.Path]
		if lb == nil {
			continue
		}
		for _, inst := range lb.GetAll() {
			total++
			if inst.IsHealthy() {
				healthy++
			}
		}
	}
	status := "healthy"
	if healthy < total {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            status,
		"healthy_instances": healthy,
		"total_instances":   total,
		"routes":            len(h.routesConfig.Routes),
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
