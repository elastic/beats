// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package testjamf provides httptest mocks and fixtures for Jamf entity-analytics tests.
package testjamf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	// For testdata/computers.json
	_ "embed"

	"github.com/gofrs/uuid/v5"
)

//go:embed testdata/computers.json
var ComputersJSON []byte

const (
	// Username is the Basic-auth username accepted by StartServer.
	Username = "testuser"
	// Password is the Basic-auth password accepted by StartServer.
	Password = "testpassword"
	// DeviceUDID is the UDID of the first computer in ComputersJSON.
	DeviceUDID = "5982CE36-4526-580B-B4B9-ECC6782535BC"
)

// StartServer starts a TLS httptest server that serves token + computers
// endpoints using ComputersJSON. Returns the tenant host (no scheme),
// credentials, and an HTTP client that trusts the test server certificate.
func StartServer(t *testing.T) (tenant, username, password string, client *http.Client) {
	t.Helper()

	username = Username
	password = Password

	var tokenMu sync.Mutex
	var currentToken string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		tokenMu.Lock()
		currentToken = uuid.Must(uuid.NewV4()).String()
		tok := currentToken
		tokenMu.Unlock()

		expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		fmt.Fprintf(w, `{"token":%q,"expires":%q}`, tok, expires)
	})
	mux.HandleFunc("/api/preview/computers", func(w http.ResponseWriter, r *http.Request) {
		tokenMu.Lock()
		tok := currentToken
		tokenMu.Unlock()

		if tok == "" || r.Header.Get("Authorization") != "Bearer "+tok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(ComputersJSON)
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host, username, password, srv.Client()
}
