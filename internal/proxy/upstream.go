package proxy

import (
	"fmt"
	"net/url"
	"time"
)

type Route struct {
	Path        string `yaml:"path"` 
	Upstream    string `yaml:"upstream"` 
	StripPrefix bool   `yaml:"strip_prefix"` 
	Timeout     int    `yaml:"timeout"` 
}

type RoutesConfig struct {
	Routes []Route `yaml:"routes"` 
}

type Upstream struct {
	URL         *url.URL
	OriginalURL string
	Timeout     time.Duration
	StripPrefix bool
	PathPrefix  string
}

func NewUpstream(route Route) (*Upstream, error) {
	parsedURL, err := url.Parse(route.Upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL %s: %w", route.Upstream, err)
	}

	timeout := time.Duration(route.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Upstream{
		URL:         parsedURL,
		OriginalURL: route.Upstream,
		Timeout:     timeout,
		StripPrefix: route.StripPrefix,
		PathPrefix:  route.Path,
	}, nil
}
