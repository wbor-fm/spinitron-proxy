package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/wcbn/spinitron-proxy/cache"
)

// Lingering questions: what is `io.NopCloser(bytes.NewReader(value)),`
// and why do we close resp.Body before reassigning it?
// What is reassignment?


// Custom transport that checks a local cache before making an external request.
// It implements http.RoundTripper, which is the interface used by http.Client.
type TransportWithCache struct {
	Transport http.RoundTripper // Underlying transport used if a cache miss.
	Cache     *cache.Cache      // Reference to our cache instance.
}

// RoundTrip is the core method of http.RoundTripper. It is called for each
// request to check the cache first, then fall back to a *real* network request.
func (t *TransportWithCache) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate a cache key based on the incoming HTTP request.
	key := t.Cache.MakeCacheKey(req)

	// Attempt to retrieve the response from the cache (e.g. if it was cached 
	// before). The second return value is a boolean indicating whether the key
	// was found in the cache.
	value, found := t.Cache.Get(key)

	if found {
		// Construct a new response with the cached data.
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(value)),
			// Wrap cached byte slice as a readable stream.
		}
		// Set the appropriate Content-Type header for JSON responses.
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil // `nil` means no error occurred.
	}

	// If not found in cache, make the actual request via underlying transport.
	tick := time.Now()  // Record the time before making the request.
	resp, err := t.Transport.RoundTrip(req) // Make the request, get response.
	if err != nil {
		// If there was an error making the request, return it immediately.
		return nil, err
	}

	// If the response status is not OK, return it directly without caching.
	if resp.StatusCode != http.StatusOK {
		return resp, err
	}

	// Else: the response status is OK, so we can cache the response body.
	// Read the entire response body into memory so we can store it in cache.
	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		// If there was an error reading the response body, return immediately.
		return nil, err
	}

	// The response body must be closed before reassigning. We then wrap the
	// data in a new read buffer so it can be read again later.
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(data))

	log.Println("request.made", time.Since(tick), key)

	// Store the response body data in the cache at the given key.
	t.Cache.Set(key, data)

	// Return the real response to the client who made the request.
	return resp, err
}

// NewReverseProxy creates a reverse proxy client to forward requests to the 
// target (Spinitron API) URL.
// It also sets up headers for authentication and configures caching.
func NewReverseProxy(tokenEnvVarName string, target *url.URL) *httputil.ReverseProxy {
	// Retrieve the Spinitron API token from the environment.
	t := os.Getenv(tokenEnvVarName)
	if t == "" {
		// If the environment variable is missing or empty, panic & crash early.
		panic(tokenEnvVarName + " environment variable is empty")
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