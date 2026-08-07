package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/SOHAR-18/gogate/internal/middleware"
)

type ReverseProxy struct {
	routes     map[string]*Upstream
	middleware middleware.Middleware
}

func NewReverseProxy(routesConfig RoutesConfig) (*ReverseProxy, error) {
	routes := make(map[string]*Upstream)
	for _, route := range routesConfig.Routes {
		upstream, err := NewUpstream(route)
		if err != nil {
			return nil, err
		}
		routes[route.Path] = upstream
		log.Printf("Registered route: %s -> %s", route.Path, route.Upstream)
	}
	return &ReverseProxy{
		routes:     routes,
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

	proxy := httputil.NewSingleHostReverseProxy(upstream.URL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
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
			r.Method, r.URL.Path, upstream.OriginalURL, err)
		writeError(w, http.StatusBadGateway, "upstream service unavailable")
	}

	coreHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r)
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		ctx, cancel := context.WithTimeout(r.Context(), upstream.Timeout)
		defer cancel()
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Served-By", "gogate")
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

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": status,
	})
}
