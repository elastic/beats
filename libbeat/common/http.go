package common

import "net/http"

func CloseIdleConnections(http *http.Client) {
	type ci interface {
		CloseIdleConnections()
	}

	if t, ok := http.Transport.(ci); ok {
		t.CloseIdleConnections()
	}
}
