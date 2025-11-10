// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestFileAuthTransportSetsHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("secret\n"), 0o600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	refresh := time.Second
	cfg := &fileAuthConfig{Path: path, Prefix: "Bearer ", RefreshInterval: &refresh}

	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: got %q want %q", got, "Bearer secret")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	transport, err := newFileAuthTransport(cfg, base)
	if err != nil {
		t.Fatalf("unexpected error creating transport: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected round trip error: %v", err)
	}
	resp.Body.Close()
}

func TestFileAuthTransportRefreshesToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("alpha"), 0o600); err != nil {
		t.Fatalf("failed to write token: %v", err)
	}

	refresh := 50 * time.Millisecond
	cfg := &fileAuthConfig{Path: path, Prefix: "Token ", RefreshInterval: &refresh}

	expect := []string{"Token alpha", "Token beta"}
	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if len(expect) == 0 {
			t.Fatalf("unexpected request beyond expectations")
		}
		want := expect[0]
		expect = expect[1:]
		if got := r.Header.Get("Authorization"); got != want {
			t.Fatalf("unexpected authorization header: got %q want %q", got, want)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	transport, err := newFileAuthTransport(cfg, base)
	if err != nil {
		t.Fatalf("unexpected error creating transport: %v", err)
	}

	current := transport.loadedAt
	transport.clock = func() time.Time { return current }

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected round trip error: %v", err)
	}
	resp.Body.Close()

	if err := os.WriteFile(path, []byte("beta"), 0o600); err != nil {
		t.Fatalf("failed to rotate token: %v", err)
	}

	current = current.Add(refresh + time.Millisecond)

	resp, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected round trip error after refresh: %v", err)
	}
	resp.Body.Close()

	if len(expect) != 0 {
		t.Fatalf("not all expectations consumed: %d remaining", len(expect))
	}
}

func TestFileAuthTransportFailsWithMissingFile(t *testing.T) {
	cfg := &fileAuthConfig{Path: "/nonexistent/path/to/token"}

	_, err := newFileAuthTransport(cfg, http.DefaultTransport)
	if err == nil {
		t.Fatal("expected error creating transport with missing file, got nil")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("expected file not found error, got: %v", err)
	}
}
