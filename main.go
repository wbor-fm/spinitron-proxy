package main

import (
	"net/http"
	"net/url"

	"github.com/wcbn/spinitron-proxy/proxy"
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
	url, _ := url.Parse(spinitronBaseURL)

	// Create a new reverse proxy that injects the API token.
	proxy := proxy.NewReverseProxy(tokenEnvVarName, url)

	// Register HTTP handlers so that any GET requests to /api/ or /images/ go 
	// through our custom reverse proxy (the proxy we created above).
	http.Handle("GET /api/", proxy)
	http.Handle("GET /images/", proxy)

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