// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServerClientPair(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	mux := http.NewServeMux()
	mux.Handle("/api/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", 401)
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
