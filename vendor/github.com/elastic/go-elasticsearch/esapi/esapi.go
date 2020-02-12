package esapi // import "github.com/elastic/go-elasticsearch/esapi"

import (
	"net/http"
)

// Transport defines the interface for an API client.
//
type Transport interface {
	Perform(*http.Request) (*http.Response, error)
}

// BoolPtr returns a pointer to v.
//
// It is used as a convenience function for converting a bool value
// into a pointer when passing the value to a function or struct field
// which expects a pointer.
//
func BoolPtr(v bool) *bool { return &v }

// IntPtr returns a pointer to v.
//
// It is used as a convenience function for converting an int value
// into a pointer when passing the value to a function or struct field
// which expects a pointer.
//
func IntPtr(v int) *int { return &v }
