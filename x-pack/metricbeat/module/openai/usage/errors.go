package usage

import "errors"

var (
	// ErrNoState indicates no previous state exists for the given API key
	ErrNoState = errors.New("no previous state found")

	// ErrHTTPClientTimeout indicates request timeout
	ErrHTTPClientTimeout = errors.New("http client request timeout")
)
