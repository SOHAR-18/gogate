package circuitbreaker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/sony/gobreaker"
)

type Manager struct {
	breakers map[string]*Breaker
	settings Settings
	mu       sync.RWMutex
}

func NewManager(settings Settings) *Manager {
	return &Manager{
		breakers: make(map[string]*Breaker),
		settings: settings,
	}
}

func (m *Manager) GetOrCreate(name string) *Breaker {
	m.mu.RLock()
	b, ok := m.breakers[name]
	m.mu.RUnlock()
	if ok {
		return b
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.breakers[name]; ok {
		return b
	}
	b = NewBreaker(name, m.settings)
	m.breakers[name] = b
	log.Printf("[CIRCUIT BREAKER] Created breaker for: %s", name)
	return b
}

func (m *Manager) Reset(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, name)
	log.Printf("[CIRCUIT BREAKER] Reset: %s", name)
}

func (m *Manager) GetAll() map[string]*Breaker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*Breaker, len(m.breakers))
	for k, v := range m.breakers {
		result[k] = v
	}
	return result
}

func (m *Manager) Wrap(name string, next http.Handler) http.Handler {
	breaker := m.GetOrCreate(name)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if breaker.IsOpen() {
			counts := breaker.Counts()
			log.Printf("[CIRCUIT BREAKER] OPEN - rejecting request to %s (failures: %d)",
				name, counts.TotalFailures)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Circuit-Breaker", "open")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "service temporarily unavailable",
				"reason":  "circuit breaker open",
				"service": name,
				"status":  503,
			})
			return
		}

		rw := &responseCapture{ResponseWriter: w, statusCode: 200}

		err := breaker.Execute(func() error {
			next.ServeHTTP(rw, r)
			if rw.statusCode >= 500 {
				return fmt.Errorf("upstream returned %d", rw.statusCode)
			}
			return nil
		})

		if err != nil {
			if err == gobreaker.ErrOpenState {
				w.Header().Set("X-Circuit-Breaker", "open")
			}
		}

		w.Header().Set("X-Circuit-Breaker-State", string(breaker.State()))
	})
}

type responseCapture struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.wroteHeader = true
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if !rc.wroteHeader {
		rc.statusCode = 200
	}
	return rc.ResponseWriter.Write(b)
}
