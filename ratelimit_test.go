package main

import (
	"testing"
	"time"
)

func TestAllowUnderLimit(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)

	for i := 0; i < 5; i++ {
		if !limiter.Allow("token1") {
			t.Fatalf("expected message %d to be allowed", i+1)
		}
	}
}

func TestAllowOverLimit(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		limiter.Allow("token1")
	}

	if limiter.Allow("token1") {
		t.Fatal("expected 4th message to be rejected")
	}
}

func TestAllowSeparateClients(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	limiter.Allow("token1")
	limiter.Allow("token1")

	if limiter.Allow("token1") {
		t.Fatal("expected token1 to be rate limited")
	}
	if !limiter.Allow("token2") {
		t.Fatal("expected token2 to be allowed (separate client)")
	}
}

func TestAllowWindowExpiry(t *testing.T) {
	limiter := NewRateLimiter(2, 50*time.Millisecond)

	limiter.Allow("token1")
	limiter.Allow("token1")

	if limiter.Allow("token1") {
		t.Fatal("expected to be rate limited")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("token1") {
		t.Fatal("expected to be allowed after window expired")
	}
}
