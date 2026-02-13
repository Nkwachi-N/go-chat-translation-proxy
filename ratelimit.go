package main

import (
	"sync"
	"time"
)

type RateLimiter struct {
	limits      map[string][]time.Time
	mu          sync.Mutex
	maxMessages int
	window      time.Duration
}

func NewRateLimiter(maxMessages int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limits:      make(map[string][]time.Time),
		maxMessages: maxMessages,
		window:      window,
	}
}

func (r *RateLimiter) Allow(token string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.window)
	var recent []time.Time
	for _, t := range r.limits[token] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= r.maxMessages {
		r.limits[token] = recent
		return false
	}

	r.limits[token] = append(recent, time.Now())
	return true
}
