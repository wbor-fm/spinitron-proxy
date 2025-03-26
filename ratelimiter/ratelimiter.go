package ratelimiter

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	// The maximum number of requests allowed in the given duration.
	MaxRequests int
	// The duration in which the maximum number of requests is allowed.
	Duration time.Duration
	// The map to store the number of requests made by an IP address.
	VisitorMap map[string]int
	// The mutex to synchronize access to the VisitorMap.
	VisitorMapMux sync.Mutex
}

// NewRateLimiter creates a new RateLimiter with the given maximum number of
// requests and duration.
func NewRateLimiter(maxRequests int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		MaxRequests: maxRequests,
		Duration:    duration,
		VisitorMap:  make(map[string]int),
	}
}

// Allow checks if the given IP address is allowed to make a request based on
// the rate limiting rules (that is, the maximum number of requests allowed in
// the given duration under a given IP and path).
func (rl *RateLimiter) Allow(r *http.Request) bool {
	rl.VisitorMapMux.Lock()
	defer rl.VisitorMapMux.Unlock()

	// Generate the request key based on the IP address and the request path.
	key := rl.MakeRequestKey(r)

	// Get the current count of requests made by the IP address.
	count, ok := rl.VisitorMap[key]
	if !ok {
		// If the IP address is not in the map, initialize it with 1 request.
		rl.VisitorMap[key] = 1
		return true
	}

	// If the count exceeds the maximum number of requests, return false.
	if count >= rl.MaxRequests {
		return false
	}

	// Increment the count of requests made by the IP address.
	rl.VisitorMap[key]++

	// Launch a goroutine to decrement the count after the duration has passed.
	go func() {
		time.Sleep(rl.Duration)
		rl.Subtract(key)
	}()

	return true
}

// The Subtract function is used to decrement the count of requests made by an
// IP address, normally called after the request has been processed. It runs
// in a separate goroutine to avoid blocking the request processing.
func (rl *RateLimiter) Subtract(key string) {
	rl.VisitorMapMux.Lock()
	defer rl.VisitorMapMux.Unlock()

	// Decrement the count of requests made under the given key (IP + path).
	rl.VisitorMap[key]--
}

// Generates the request key based on the IP address and the request path.
func (rl *RateLimiter) MakeRequestKey(r *http.Request) string {
	// Strip the port number from the IP address.
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip + r.URL.Path
}

// The middleware function that wraps the handler and enforces rate limiting.
// If the request is denied, it returns a 429 Too Many Requests status code.
// Additionally, it logs the IP address and path of the request that was denied.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			// Write how much time the client has to wait before making another request.
			http.Header.Add(w.Header(), "Retry-After", rl.Duration.String())
			// Return a 429 Too Many Requests status code.
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)

			log.Printf("Rate limit exceeded for %s on %s\n", r.RemoteAddr, r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// This middleware function is similar to the previous one, but it's meant
// for use with the http.HandlerFunc type instead of http.Handler.
// It enforces rate limiting and returns a 429 Too Many Requests status code,
// along with logging the IP address and path of the request that was denied.
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			http.Header.Add(w.Header(), "Retry-After", rl.Duration.String())
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)

			log.Printf("Rate limit exceeded for %s on %s\n", r.RemoteAddr, r.URL.Path)
			return
		}
		next(w, r)
	}
}
