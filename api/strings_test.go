package api

import (
	"testing"
)

// Checks known resource paths and expects them to match.
func TestIsResourcePath(t *testing.T) {
	known := []string{
		"/api/personas/1",
		"/api/shows/2",
		"/api/playlists/3",
		"/api/spins/4",
		"/images/Persona/16/65/166599-img_profile.225x225.jpg?v=123",
	}

	for _, name := range known {
		result := IsResourcePath(name)
		if !result {
			t.Errorf("IsResourcePath(%s) = %t; want true", name, result)
		}
	}
}

func TestIsResourcePathEdgeCases(t *testing.T) {
	paths := []string{
		"",
		"/",
		"/api",
		"/api/",
		"/api/shows/",
		"/api/shows//2",
		"/images/Persona/noDigits",
		"/api/spins/abc",
		"api/shows/2",
		"/api/spins/4?someparam=123",
		"/api/spins/4/edit",
	}

	for _, p := range paths {
		got := IsResourcePath(p)
		// We only expect certain ones to be true.
		switch p {
		case "/api/spins/4?someparam=123":
			if !got {
				t.Errorf("IsResourcePath(%q) returned false; want true", p)
			}
		default:
			if got {
				t.Errorf("IsResourcePath(%q) returned true; want false", p)
			}
		}
	}
}

// Checks known collection paths and expects them to be identified correctly.
func TestIsCollectionPath(t *testing.T) {
	known := []string{
		"/api/personas",
		"/api/shows",
		"/api/playlists",
		"/api/spins",
	}

	for _, name := range known {
		result := IsCollectionPath(name)
		if !result {
			t.Errorf("IsCollectionPath(%s) = %t; want true", name, result)
		}
	}
}

func TestIsCollectionPathEdgeCases(t *testing.T) {
	paths := []string{
		"",
		"/",
		"/api",
		"/api/",
		"/api//",
		"/images/Persona",
		"api/shows",
		"/api/shows//",
	}

	for _, p := range paths {
		got := IsCollectionPath(p)
		if got {
			t.Errorf("IsCollectionPath(%q) returned true; want false", p)
		}
	}
}

// Ensures that resource-like paths aren't identified as collection paths.
func TestIsNotCollectionPath(t *testing.T) {
	known := []string{
		"/api/personas/1",
		"/images/Persona/16/65/166599-img_profile.225x225.jpg?v=123",
	}

	for _, name := range known {
		result := !IsCollectionPath(name)
		if !result {
			t.Errorf("IsCollectionPath(%s) = %t; want true", name, result)
		}
	}
}

// Checks various path formats to verify correct extraction of the collection
// name.
func TestGetCollectionName(t *testing.T) {
	s := []string{
		"api/foo",
		"foo",
		"/api/foo",
		"/foo",
		"/foo/",
		"/api/foo/",
		"/foo/",
		"/api/foo/?bar=baz",
		"/api/foo?bar=baz",
	}

	for _, name := range s {
		result := GetCollectionName(name)
		if result != "foo" {
			t.Errorf("GetCollectionName(%s) = %s; want foo", name, result)
		}
	}
}

func TestGetCollectionNameEdgeCases(t *testing.T) {
	tests := map[string]string{
		"":            "",
		"/":           "",
		"/api/":       "",
		"/api//":      "",
		"api/shows":   "shows",
		"/api/shows//2": "shows",
	}

	for input, want := range tests {
		got := GetCollectionName(input)
		if got != want {
			t.Errorf("GetCollectionName(%q) = %q; want %q", input, got, want)
		}
	}
}
