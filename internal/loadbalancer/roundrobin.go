package loadbalancer

import (
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
)

type Instance struct {
	URL     *url.URL
	RawURL  string
	Healthy bool
	Weight  int
	mu      sync.RWMutex
}

func (i *Instance) SetHealthy(healthy bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Healthy = healthy
}

func (i *Instance) IsHealthy() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Healthy
}

type RoundRobin struct {
	instances []*Instance
	counter   atomic.Uint64
	mu        sync.RWMutex
}

func NewRoundRobin(rawURLs []string) (*RoundRobin, error) {
	instances := make([]*Instance, 0, len(rawURLs))
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		instances = append(instances, &Instance{
			URL:     u,
			RawURL:  raw,
			Healthy: true,
			Weight:  1,
		})
	}
	return &RoundRobin{instances: instances}, nil
}

func (rr *RoundRobin) Next() (*Instance, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	healthy := make([]*Instance, 0)
	for _, inst := range rr.instances {
		if inst.IsHealthy() {
			healthy = append(healthy, inst)
		}
	}

	if len(healthy) == 0 {
		return nil, errors.New("no healthy instances available")
	}

	idx := rr.counter.Add(1) % uint64(len(healthy))
	return healthy[idx], nil
}

func (rr *RoundRobin) GetAll() []*Instance {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.instances
}

func (rr *RoundRobin) SetHealthy(rawURL string, healthy bool) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	for _, inst := range rr.instances {
		if inst.RawURL == rawURL {
			inst.SetHealthy(healthy)
			return
		}
	}
}
