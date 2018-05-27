package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServerClientPair(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	mux := http.NewServeMux()
	mux.Handle("/api/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name": "test", "version": {"number": "6.3.0", "build_snapshot": false}}`)
	}))
	mux.Handle("/", handler)

	server := httptest.NewServer(mux)

	config, err := ConfigFromURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return server, client
}
