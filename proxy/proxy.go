package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/WBOR-91-1-FM/spinitron-proxy/cache"
)

// Lingering questions: what is `io.NopCloser(bytes.NewReader(value)),`
// and why do we close resp.Body before reassigning it?
// What is reassignment?

// OnSpinsUpdate is a callback that, when set, is called with a message
// after /api/spins is updated.
var OnSpinsUpdate func(msg string)

// Custom transport that checks a local cache before making an external request.
// (Unless the request has ?forceRefresh=1, in which case it skips the cache.)
// It implements http.RoundTripper, which is the interface used by http.Client.
type TransportWithCache struct {
	Transport http.RoundTripper // Underlying transport for cache misses.
	Cache     *cache.Cache      // In-memory cache.
}

// RoundTrip checks the cache before making a network request. It caches fresh
// responses and broadcasts an SSE message if the request is for spins.
func (t *TransportWithCache) RoundTrip(req *http.Request) (*http.Response, error) {

	// Check if the request has ?forceRefresh=1 to skip cache retrieval
	forceRefresh := (req.URL.Query().Get("forceRefresh") == "1")

	// Generate a cache key based on the incoming HTTP request.
	key := t.Cache.MakeCacheKey(req)

	// If forceRefresh is NOT set, try retrieving from the cache as normal.
	if !forceRefresh {
		if value, found := t.Cache.Get(key); found {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(value)),
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil // `nil` means no error occurred.
		}
	} else {
		// If forceRefresh is set, log that we're skipping the cache.
		log.Println("cache.skip", key, "(forceRefresh)")
	}

	// If forceRefresh IS set, or cache was a miss, do the real network request.
	tick := time.Now()
	resp, err := t.Transport.RoundTrip(req) // Make the request, get response.
	if err != nil {
		// If there was an error making the request, return it immediately.
		return nil, err
	}

	// If the response status is not OK, return it directly without caching.
	if resp.StatusCode != http.StatusOK {
		return resp, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		// If there was an error reading the response body, return immediately.
		return nil, err
	}
	// The response body must be closed before reassigning.
	resp.Body.Close()

	// Wrap data in a new ReadCloser so the rest of the chain can read it.
	resp.Body = io.NopCloser(bytes.NewReader(data))
	log.Println("request.made", time.Since(tick), key)

	// Even if forceRefresh was set, we still store the new data in the cache,
	// so that subsequent requests without forceRefresh can use the updated
	// data.
	t.Cache.Set(key, data)

	// If the request is for the spins collection, broadcast an SSE message.
	if strings.HasPrefix(req.URL.Path, "/api/spins") {
		if OnSpinsUpdate != nil {
			OnSpinsUpdate("new spin data")
		}
	}

	return resp, err
}

// NewReverseProxy creates a reverse proxy client that forwards requests to the
// target (Spinitron API) URL. It also sets up authentication and caching.
func NewReverseProxy(tokenEnvVarName string, target *url.URL) *httputil.ReverseProxy {
	// Retrieve the Spinitron API token from the environment.
	t := os.Getenv(tokenEnvVarName)
	if t == "" {
		panic(tokenEnvVarName + " environment variable is empty")
	}

	pubDomain := os.Getenv("INSTALLATION_BASE_URL")
	if pubDomain == "" {
		panic("INSTALLATION_BASE_URL environment variable is empty")
	}

	// Create a single-host reverse proxy for the given target URL.
	// A single-host reverse proxy forwards requests to a single target URL.
	// It is a struct that implements the http.Handler interface.
	// The target URL is the destination to which the proxy forwards requests.
	rp := httputil.NewSingleHostReverseProxy(target)

	// Preserve the existing Director function (which rewrites request URLs)
	// then extend it to set our custom headers.
	// A director function is a function that modifies the request before it is
	// sent to the target server.
	d := rp.Director
	rp.Director = func(req *http.Request) {
		// Call the original director to set the X-Forwarded-* headers, etc.
		d(req)
		// Inject the "Authorization" header with our bearer token for API access.
		req.Header.Set("Authorization", "Bearer "+t)
		// Force the request to accept JSON.
		req.Header.Set("accept", "application/json")

		// Set the Host header to the target host
		req.Host = pubDomain
		req.Header.Set("X-Forwarded-Host", pubDomain)
	}

	// Initialize in-memory cache to store responses using the cache package we
	// defined in cache/cache.go.
	c := &cache.Cache{}
	c.Init()

	// Override the proxy's default (from httputil.ReverseProxy) transport with
	// our custom caching transport.
	rp.Transport = &TransportWithCache{
		Transport: http.DefaultTransport,
		Cache:     c,
	}

	return rp
}
