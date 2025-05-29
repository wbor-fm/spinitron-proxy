package main

import (
	"io"
	"log"
	"time"

	"net/http"
	"net/url"

	"github.com/WBOR-91-1-FM/spinitron-proxy/proxy"
	"github.com/WBOR-91-1-FM/spinitron-proxy/ratelimiter"
)

const tokenEnvVarName = "SPINITRON_API_KEY"
const spinitronBaseURL = "https://spinitron.com"

// healthzHandler responds with a simple OK for health checks.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	// Parse the base URL for Spinitron using the net/url package.
	// Parse() returns a URL struct and an error if there is one, but here we're
	// ignoring the error with the underscore since we know the URL is valid (as
	// it is hardcoded).
	// If the URL were to be provided by the user, we would need to handle the
	// error and return a message to the user.
	parsedURL, _ := url.Parse(spinitronBaseURL)

	// Create a new reverse proxy that injects the API token.
	revProxy := proxy.NewReverseProxy(tokenEnvVarName, parsedURL)
	proxy.OnSpinsUpdate = BroadcastSpinMessage

	// Create a new rate limiter with a maximum of 60 requests per minute.
	rateLimiter := ratelimiter.NewRateLimiter(60, time.Minute)

	// Register the health check handler for the /healthz endpoint, not rate-limited.
	http.HandleFunc("/healthz", healthzHandler)

	// Normal proxy routes: /api/ and /images/
	// Register HTTP handlers so that any GET requests to /api/ or /images/ go
	// through our custom reverse proxy (the proxy we created above).
	http.Handle("GET /api/", rateLimiter.Middleware(revProxy))
	http.Handle("GET /images/", rateLimiter.Middleware(revProxy))

	// SSE Endpoint.
	http.HandleFunc("/spin-events", rateLimiter.MiddlewareFunc(spinEventsHandler))

	// POST route to trigger an internal GET request for /api/spins to force a
	// refresh of the cache. This is used by Spinitron to trigger a refresh of
	// the cache when new spins POSTed by a DJ or Automation.
	// (In the future: may consider adding authentication to this route
	// to prevent unauthorized access and abuse.)
	http.HandleFunc("/trigger/spins", rateLimiter.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Println("trigger.spins")

		// This request goes back into our own server, ensuring the proxy logic
		// is used. The key part is `?forceRefresh=1`.
		resp, err := http.Get("http://localhost:8080/api/spins?forceRefresh=1")
		if err != nil {
			http.Error(w, "Failed to fetch spins: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		// Read entire body so that the data flows through to our caching
		// mechanism. TL;DR: this line is important because without it, the
		// response body is not read and the cache is not updated! Reading the
		// body is necessary to update the cache since the cache is updated in
		// the proxy logic via `t.Cache.Set(key, data)` which is called when the
		// response status is OK (handled in proxy.go).
		_, _ = io.ReadAll(resp.Body)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Forced refresh of /api/spins. Cache updated."))
	}))

	log.Println("spinitron-proxy started on port 8080, health check available at /healthz")

	// Listen on port 8080 for incoming HTTP requests. If there's an error, it
	// returns a non-nil error (nil means no error).
	// The `:=` operator is shorthand for declaring and initializing a variable
	// in one line. It is equivalent to:
	//   var err error
	//   err = http.ListenAndServe(":8080", nil)
	err := http.ListenAndServe(":8080", nil)

	// If ListenAndServe returns an error, panic is called to log it and exit
	// the program.
	panic(err)
}
