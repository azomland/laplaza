package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── Security Headers ──

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; style-src 'unsafe-inline'; script-src 'self'")
		}

		next.ServeHTTP(w, r)
	})
}

// ── Rate Limiter ──

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*rateEntry
	requests int
	window   time.Duration
}

type rateEntry struct {
	count    int
	resetAt  time.Time
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*rateEntry),
		requests: maxRequests,
		window:   window,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}

		rl.mu.Lock()
		entry, ok := rl.clients[ip]
		now := time.Now()

		if !ok || now.After(entry.resetAt) {
			entry = &rateEntry{count: 0, resetAt: now.Add(rl.window)}
			rl.clients[ip] = entry
		}

		entry.count++
		remaining := rl.requests - entry.count
		rl.mu.Unlock()

		w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", itoa(int(entry.resetAt.Unix())))

		if entry.count > rl.requests {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			log.Printf("rate limit exceeded for %s", ip)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// ── Logger ──

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &logWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, lw.status, time.Since(start))
	})
}

type logWriter struct {
	http.ResponseWriter
	status int
}

func (lw *logWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

// tiny itoa for headers (avoids strconv import)
func itoa(n int) string {
	if n == 0 { return "0" }
	s := ""
	neg := n < 0
	if neg { n = -n }
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg { s = "-" + s }
	return s
}
