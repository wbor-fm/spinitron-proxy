package api

import (
	"net/url"
	"regexp"
	"strings"
)

var (
    // For /api/: e.g. /api/shows/123 plus optional ?query
    apiResourceRegex = regexp.MustCompile(`^/api/\w+/\d+(?:\?.*)?$`)

    // For /images/: allow multiple subdirectories, final segment must contain at least one digit
    // e.g. /images/Persona/16/65/166599-img_profile.225x225.jpg?v=123
    // It matches zero or more non-slash chars, at least one digit, then zero or more non-slash chars.
    imagesResourceRegex = regexp.MustCompile(`^/images/(?:[^/]+/)*[^/]*[0-9]+[^/]*(?:\?.*)?$`)

    // For /api/: e.g. /api/shows plus optional ?query
    apiCollectionRegex = regexp.MustCompile(`^/api/\w+(?:\?.*)?$`)
)

// IsResourcePath returns true if it matches either the /api/ resource pattern or the /images/ resource pattern.
func IsResourcePath(path string) bool {
    return apiResourceRegex.MatchString(path) || imagesResourceRegex.MatchString(path)
}

// IsCollectionPath returns true if it’s a valid /api/ collection path, excluding anything recognized as a resource path.
func IsCollectionPath(path string) bool {
	if IsResourcePath(path) {
		return false
	}
	return apiCollectionRegex.MatchString(path)
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
