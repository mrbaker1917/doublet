package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultCreateRateLimit  = 20
	defaultMoveRateLimit    = 120
	defaultReadRateLimit    = 180
	defaultAPIRateWindow    = time.Minute
	defaultMaxConcurrentBFS = 4
	defaultBFSWait          = 5 * time.Second
	defaultPathCacheSize    = 4096
)

type bfsGate struct {
	sem chan struct{}
}

func newBFSGate(maxConcurrent int) *bfsGate {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrentBFS
	}
	return &bfsGate{sem: make(chan struct{}, maxConcurrent)}
}

func (g *bfsGate) acquire(wait time.Duration) bool {
	select {
	case g.sem <- struct{}{}:
		return true
	case <-time.After(wait):
		return false
	}
}

func (g *bfsGate) release() {
	<-g.sem
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
	limit   int
	window  time.Duration
}

type rateEntry struct {
	count int
	reset time.Time
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	if limit <= 0 {
		limit = defaultReadRateLimit
	}
	if window <= 0 {
		window = defaultAPIRateWindow
	}
	return &ipRateLimiter{
		entries: make(map[string]*rateEntry),
		limit:   limit,
		window:  window,
	}
}

func (l *ipRateLimiter) allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.After(entry.reset) {
		l.entries[key] = &rateEntry{count: 1, reset: now.Add(l.window)}
		l.cleanupLocked(now)
		return true
	}

	if entry.count >= l.limit {
		return false
	}

	entry.count++
	return true
}

func (l *ipRateLimiter) cleanupLocked(now time.Time) {
	for key, entry := range l.entries {
		if now.After(entry.reset) {
			delete(l.entries, key)
		}
	}
}

type pathCache struct {
	mu      sync.RWMutex
	entries map[string]pathCacheEntry
	maxSize int
	order   []string
}

type pathCacheEntry struct {
	path  []string
	found bool
}

func newPathCache(maxSize int) *pathCache {
	if maxSize <= 0 {
		maxSize = defaultPathCacheSize
	}
	return &pathCache{
		entries: make(map[string]pathCacheEntry),
		maxSize: maxSize,
		order:   make([]string, 0, maxSize),
	}
}

func pathCacheKey(start, end string, expert bool) string {
	mode := "common"
	if expert {
		mode = "expert"
	}
	return mode + "\x00" + start + "\x00" + end
}

func (c *pathCache) get(start, end string, expert bool) ([]string, bool, bool) {
	key := pathCacheKey(start, end, expert)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false, false
	}

	path := append([]string(nil), entry.path...)
	return path, entry.found, true
}

func (c *pathCache) put(start, end string, expert bool, path []string, found bool) {
	key := pathCacheKey(start, end, expert)
	stored := append([]string(nil), path...)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists {
		c.order = append(c.order, key)
		for len(c.order) > c.maxSize {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.entries, oldest)
		}
	}

	c.entries[key] = pathCacheEntry{path: stored, found: found}
}

func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("Fly-Client-IP")); ip != "" {
		return ip
	}
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
