package api

import (
	"net/url"
	"regexp"
	"strings"
)

// TODO: harden these and test edge cases

// Returns true if the path points to single resource rather than a collection.
// Example: `/api/shows/2` or `/images/Persona/16/65/166599-img_profile.jpg`
func IsResourcePath(path string) bool {
	// Matches any path starting with /api/ or /images/ followed by
	// a word and then some digits or additional path segments.
	// Would not match /api/shows or /images/Persona alone.
	// `re` is a regular expression object that can be used to match strings.
	// `_` is a blank identifier, used to ignore the error value returned by
	// `regexp.Compile` (since we know the regex is valid).
	re, _ := regexp.Compile(`\/(api|images)\/\w+\/\d+.*`)
	// MatchString returns true if the regex matches the input string.
	return re.MatchString(path)
}

// Returns true if the path is a collection path like /api/shows (as opposed to 
// /api/shows/2). It first checks if the path is a resource path to avoid 
// overlap.
func IsCollectionPath(path string) bool {
	if IsResourcePath(path) {
		return false
	}
	// Then check if it's an /api/ path with some word following it.
	// This would match /api/shows but not /api or /api/.
	re, _ := regexp.Compile(`\/api\/\w+.*`)
	return re.MatchString(path)
}

// Extracts the primary path segment right after "/api/".
// If not found, it just returns an empty string.
func GetCollectionName(s string) string {
	// Use a dummy base URL because url.Parse requires a full URL.
	dummy := "https://foo.com/"
	// `s` is the input string, which is a path.
	// Parse the path into a URL struct.
	u, _ := url.Parse(dummy + s)

	// Split the path by '/' and look for the segment that follows "api".
	segments := strings.Split(u.Path, "/")

	for i := range segments {
		// Skip the "api" segment (or empty strings, which may occur if the path
		// starts with a slash).
		if segments[i] == "api" || segments[i] == "" {
			continue
		}
		// Return the first relevant segment, as it is the collection name.
		return segments[i]
	}
	// If no relevant segment is found, return an empty string.
	return ""
}