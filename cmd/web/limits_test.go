package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := newIPRateLimiter(2, time.Minute)

	if !limiter.allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.allow("127.0.0.1") {
		t.Fatal("second request should be allowed")
	}
	if limiter.allow("127.0.0.1") {
		t.Fatal("third request should be blocked")
	}
}

func TestIPRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := newIPRateLimiter(1, 10*time.Millisecond)

	if !limiter.allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}
	if limiter.allow("127.0.0.1") {
		t.Fatal("second request should be blocked")
	}

	time.Sleep(15 * time.Millisecond)

	if !limiter.allow("127.0.0.1") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestIPRateLimiterTracksIPsSeparately(t *testing.T) {
	limiter := newIPRateLimiter(1, time.Minute)

	if !limiter.allow("1.1.1.1") {
		t.Fatal("first IP should be allowed")
	}
	if !limiter.allow("2.2.2.2") {
		t.Fatal("second IP should be allowed")
	}
}

func TestBFSGateLimitsConcurrency(t *testing.T) {
	gate := newBFSGate(1)

	if !gate.acquire(time.Second) {
		t.Fatal("first acquire should succeed")
	}

	done := make(chan bool, 1)
	go func() {
		done <- gate.acquire(50 * time.Millisecond)
	}()

	select {
	case ok := <-done:
		if ok {
			t.Fatal("second acquire should time out while slot held")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second acquire attempt")
	}

	gate.release()

	if !gate.acquire(time.Second) {
		t.Fatal("acquire should succeed after release")
	}
	gate.release()
}

func TestPathCacheHitAvoidsRepeatLookup(t *testing.T) {
	cache := newPathCache(10)

	cache.put("cat", "dog", false, []string{"cat", "cot", "dog"}, true)

	path, found, ok := cache.get("cat", "dog", false)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !found || len(path) != 3 {
		t.Fatalf("unexpected cached path: found=%v path=%v", found, path)
	}

	path[0] = "mutated"
	pathAgain, _, ok := cache.get("cat", "dog", false)
	if !ok || pathAgain[0] != "cat" {
		t.Fatalf("cache entry should not be mutated by caller: %v", pathAgain)
	}
}

func TestPathCacheEvictsOldestWhenFull(t *testing.T) {
	cache := newPathCache(2)

	cache.put("a", "b", false, []string{"a", "b"}, true)
	cache.put("c", "d", false, []string{"c", "d"}, true)
	cache.put("e", "f", false, []string{"e", "f"}, true)

	if _, _, ok := cache.get("a", "b", false); ok {
		t.Fatal("oldest cache entry should be evicted")
	}
	if _, _, ok := cache.get("c", "d", false); !ok {
		t.Fatal("middle cache entry should remain")
	}
	if _, _, ok := cache.get("e", "f", false); !ok {
		t.Fatal("newest cache entry should remain")
	}
}

func TestClientIPUsesFlyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Fly-Client-IP", "203.0.113.10")
	req.RemoteAddr = "127.0.0.1:1234"

	if got := clientIP(req); got != "203.0.113.10" {
		t.Fatalf("clientIP() = %q, want Fly-Client-IP", got)
	}
}

func TestClientIPUsesForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.20, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:1234"

	if got := clientIP(req); got != "203.0.113.20" {
		t.Fatalf("clientIP() = %q, want first X-Forwarded-For hop", got)
	}
}
