package common

import "net/http"

// CloseIdleConnections closes any idle connections if the transport allow it.
func CloseIdleConnections(http *http.Client) {
	type ci interface {
		CloseIdleConnections()
	}

	if t, ok := http.Transport.(ci); ok {
		t.CloseIdleConnections()
	}
}
