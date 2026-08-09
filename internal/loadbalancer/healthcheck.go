package loadbalancer

import (
	"log"
	"net/http"
	"time"
)

type HealthChecker struct {
	lb       *RoundRobin
	path     string
	interval time.Duration
	timeout  time.Duration
	failures map[string]int
	maxFails int
}

func NewHealthChecker(lb *RoundRobin, path string) *HealthChecker {
	return &HealthChecker{
		lb:       lb,
		path:     path,
		interval: 10 * time.Second,
		timeout:  3 * time.Second,
		failures: make(map[string]int),
		maxFails: 3,
	}
}

func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		for range ticker.C {
			hc.checkAll()
		}
	}()
	log.Printf("[HEALTHCHECK] Started for %d instances", len(hc.lb.GetAll()))
}

func (hc *HealthChecker) checkAll() {
	for _, inst := range hc.lb.GetAll() {
		go hc.checkOne(inst)
	}
}

func (hc *HealthChecker) checkOne(inst *Instance) {
	client := &http.Client{Timeout: hc.timeout}
	url := inst.RawURL + hc.path
	resp, err := client.Get(url)

	if err != nil || resp.StatusCode >= 500 {
		hc.failures[inst.RawURL]++
		if hc.failures[inst.RawURL] >= hc.maxFails {
			if inst.IsHealthy() {
				inst.SetHealthy(false)
				log.Printf("[HEALTHCHECK] Instance UNHEALTHY: %s (failed %d times)",
					inst.RawURL, hc.failures[inst.RawURL])
			}
		}
		return
	}

	if !inst.IsHealthy() {
		log.Printf("[HEALTHCHECK] Instance RECOVERED: %s", inst.RawURL)
	}
	hc.failures[inst.RawURL] = 0
	inst.SetHealthy(true)
}
