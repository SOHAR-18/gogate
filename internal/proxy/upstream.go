package proxy

import (
	"fmt"
	"time"
)

type Route struct {
	Path        string   `yaml:"path"`
	ServiceName string   `yaml:"service_name"`
	Upstreams   []string `yaml:"upstreams"`
	StripPrefix bool     `yaml:"strip_prefix"`
	Timeout     int      `yaml:"timeout"`
	Protected   bool     `yaml:"protected"`
	RateLimit   int      `yaml:"rate_limit"`
	RateWindow  int      `yaml:"rate_window"`
	HealthPath  string   `yaml:"health_path"`
}

type RoutesConfig struct {
	Routes []Route `yaml:"routes"`
}

type Upstream struct {
	URLs        []string
	ServiceName string
	Timeout     time.Duration
	StripPrefix bool
	PathPrefix  string
	Protected   bool
	RateLimit   int
	RateWindow  int
	HealthPath  string
}

func NewUpstream(route Route) (*Upstream, error) {
	if len(route.Upstreams) == 0 {
		return nil, fmt.Errorf("no upstreams defined for route %s", route.Path)
	}

	timeout := time.Duration(route.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	healthPath := route.HealthPath
	if healthPath == "" {
		healthPath = "/health"
	}

	serviceName := route.ServiceName
	if serviceName == "" {
		serviceName = route.Path[1:]
	}

	return &Upstream{
		URLs:        route.Upstreams,
		ServiceName: serviceName,
		Timeout:     timeout,
		StripPrefix: route.StripPrefix,
		PathPrefix:  route.Path,
		Protected:   route.Protected,
		RateLimit:   route.RateLimit,
		RateWindow:  route.RateWindow,
		HealthPath:  healthPath,
	}, nil
}
