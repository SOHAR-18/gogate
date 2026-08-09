package discovery

import (
	"context"
	"log"
	"net/url"

	"github.com/SOHAR-18/gogate/internal/loadbalancer"
)

type Registry struct {
	client    *Client
	balancers map[string]*loadbalancer.RoundRobin
}

func NewRegistry(client *Client) *Registry {
	return &Registry{
		client:    client,
		balancers: make(map[string]*loadbalancer.RoundRobin),
	}
}

func (r *Registry) Watch(ctx context.Context, serviceName string, lb *loadbalancer.RoundRobin) {
	r.balancers[serviceName] = lb

	urls, err := r.client.GetServices(ctx, serviceName)
	if err != nil {
		log.Printf("[REGISTRY] Failed to get initial services for %s: %v", serviceName, err)
	} else if len(urls) > 0 {
		r.addNewInstances(lb, urls)
	}

	r.client.Watch(ctx, serviceName, func(urls []string) {
		if len(urls) > 0 {
			r.addNewInstances(lb, urls)
		}
	})
}

func (r *Registry) addNewInstances(lb *loadbalancer.RoundRobin, urls []string) {
	existing := make(map[string]bool)
	for _, inst := range lb.GetAll() {
		existing[inst.RawURL] = true
	}

	for _, u := range urls {
		if !existing[u] {
			parsed, err := url.Parse(u)
			if err != nil {
				log.Printf("[REGISTRY] Invalid URL %s: %v", u, err)
				continue
			}
			lb.AddInstance(parsed, u)
			log.Printf("[REGISTRY] Added instance: %s", u)
		}
	}

	for _, inst := range lb.GetAll() {
		inst.SetHealthy(true)
		log.Printf("[REGISTRY] Marked healthy: %s", inst.RawURL)
	}
}
