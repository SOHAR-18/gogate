package circuitbreaker

import (
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Breaker struct {
	cb   *gobreaker.CircuitBreaker
	name string
}

type Settings struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	FailThreshold uint32
}

func DefaultSettings() Settings {
	return Settings{
		MaxRequests:   3,
		Interval:      10 * time.Second,
		Timeout:       30 * time.Second,
		FailThreshold: 5,
	}
}

func NewBreaker(name string, settings Settings) *Breaker {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: settings.MaxRequests,
		Interval:    settings.Interval,
		Timeout:     settings.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= settings.FailThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("[CIRCUIT BREAKER] %s: %s -> %s",
				name, stateToString(from), stateToString(to))
		},
	}
	return &Breaker{
		cb:   gobreaker.NewCircuitBreaker(st),
		name: name,
	}
}

func (b *Breaker) Execute(fn func() error) error {
	_, err := b.cb.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

func (b *Breaker) State() State {
	return stateToString(b.cb.State())
}

func (b *Breaker) Counts() gobreaker.Counts {
	return b.cb.Counts()
}

func (b *Breaker) Name() string {
	return b.name
}

func (b *Breaker) IsOpen() bool {
	return b.cb.State() == gobreaker.StateOpen
}

func stateToString(s gobreaker.State) State {
	switch s {
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	default:
		return StateClosed
	}
}

func ErrOpen(name string) error {
	return fmt.Errorf("circuit breaker '%s' is open", name)
}
