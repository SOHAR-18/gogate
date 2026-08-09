package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/SOHAR-18/gogate/internal/loadbalancer"
	"github.com/SOHAR-18/gogate/internal/middleware"
)

type ReverseProxy struct {
	routes     map[string]*Upstream
	balancers  map[string]*loadbalancer.RoundRobin
	middleware middleware.Middleware
}

func NewReverseProxy(routesConfig RoutesConfig) (*ReverseProxy, error) {
	routes := make(map[string]*Upstream)
	balancers := make(map[string]*loadbalancer.RoundRobin)

	for _, route := range routesConfig.Routes {
		upstream, err := NewUpstream(route)
		if err != nil {
			return nil, err
		}
		routes[route.Path] = upstream

		lb, err := loadbalancer.NewRoundRobin(route.Upstreams)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer for %s: %w", route.Path, err)
		}
		balancers[route.Path] = lb

		hc := loadbalancer.NewHealthChecker(lb, upstream.HealthPath)
		hc.Start()

		log.Printf("Registered route: %s -> %v", route.Path, route.Upstreams)
	}

	return &ReverseProxy{
		routes:     routes,
		balancers:  balancers,
		middleware: middleware.DefaultChain(),
	}, nil
}

func (rp *ReverseProxy) Handler(pathPrefix string) http.Handler {
	upstream, ok := rp.routes[pathPrefix]
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "no route found for "+pathPrefix)
		})
	}

	lb := rp.balancers[pathPrefix]

	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inst, err := lb.Next()
		if err != nil {
			log.Printf("[LB ERROR] No healthy instances for %s: %v", pathPrefix, err)
			writeError(w, http.StatusServiceUnavailable, "no healthy upstream instances")
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(inst.URL)

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Host = inst.URL.Host
			req.URL.Scheme = inst.URL.Scheme
			if upstream.StripPrefix {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefix)
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}
			req.Header.Set("X-Forwarded-By", "gogate")
			req.Header.Set("X-Original-Path", req.URL.Path)
			req.Header.Del("X-Forwarded-For")
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[PROXY ERROR] %s %s -> %s: %v",
				r.Method, r.URL.Path, inst.RawURL, err)
			inst.SetHealthy(false)
			writeError(w, http.StatusBadGateway, "upstream service unavailable")
		}

		requestID := middleware.GetRequestID(r)
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		ctx, cancel := context.WithTimeout(r.Context(), upstream.Timeout)
		defer cancel()
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Served-By", "gogate")
		w.Header().Set("X-Upstream", inst.RawURL)

		proxy.ServeHTTP(w, r)
	})

	return rp.middleware(coreHandler)
}

func (rp *ReverseProxy) GetRoutes() []string {
	paths := make([]string, 0, len(rp.routes))
	for path := range rp.routes {
		paths = append(paths, path)
	}
	return paths
}

func (rp *ReverseProxy) GetBalancer(path string) *loadbalancer.RoundRobin {
	return rp.balancers[path]
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": status,
	})
}

func parseURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
