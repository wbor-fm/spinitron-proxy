package main

import (
	"io"
	"log"

	"net/http"
	"net/url"

	"github.com/WBOR-91-1-FM/spinitron-proxy/proxy"
)

const tokenEnvVarName = "SPINITRON_API_KEY"
const spinitronBaseURL = "https://spinitron.com"

func main() {
	// Parse the base URL for Spinitron using the net/url package.
	// Parse() returns a URL struct and an error if there is one, but here we're
	// ignoring the error with the underscore since we know the URL is valid (as
	// it is hardcoded).
	// If the URL were to be provided by the user, we would need to handle the
	// error and return a message to the user.
	parsedURL, _ := url.Parse(spinitronBaseURL)

	// Create a new reverse proxy that injects the API token.
	proxy := proxy.NewReverseProxy(tokenEnvVarName, parsedURL)

	// Normal proxy routes: /api/ and /images/
	// Register HTTP handlers so that any GET requests to /api/ or /images/ go 
	// through our custom reverse proxy (the proxy we created above).
	http.Handle("GET /api/", proxy)
	http.Handle("GET /images/", proxy)

	// SSE Endpoint
	http.HandleFunc("/spin-events", spinEventsHandler)

	// POST route to trigger an internal GET request for /api/spins and 
	// broadcast a message to all SSE clients.
	// (In the future: may consider adding authentication to this route
	// to prevent unauthorized access and abuse.)
	http.HandleFunc("/trigger/spins", func(w http.ResponseWriter, r *http.Request) {
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

		BroadcastSpinMessage("new spin data")
		log.Println("sse.broadcast", len(sseClients))

        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Forced refresh of /api/spins. Cache updated."))
    })

	log.Println("Spinitron-proxy application started on port 8080")

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