// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

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
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("super-secret\n"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	refresh := time.Second
	cfg := &fileAuthConfig{Path: path, Prefix: "Bearer ", RefreshInterval: &refresh}

	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer super-secret" {
			t.Fatalf("unexpected authorization header: got %q want %q", auth, "Bearer super-secret")
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

func TestFileAuthTransportRefreshesValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("alpha"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	refresh := 50 * time.Millisecond
	cfg := &fileAuthConfig{Path: path, Prefix: "ApiKey ", RefreshInterval: &refresh}

	expectations := []string{"ApiKey alpha", "ApiKey beta"}
	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if len(expectations) == 0 {
			t.Fatalf("unexpected extra request")
		}
		expected := expectations[0]
		expectations = expectations[1:]
		if got := r.Header.Get("Authorization"); got != expected {
			t.Fatalf("unexpected authorization header: got %q want %q", got, expected)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	transport, err := newFileAuthTransport(cfg, base)
	if err != nil {
		t.Fatalf("unexpected error creating transport: %v", err)
	}

	current := transport.expires.Add(-refresh)
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
		t.Fatalf("failed to rotate secret file: %v", err)
	}

	current = current.Add(refresh + time.Millisecond)

	resp, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected round trip error after refresh: %v", err)
	}
	resp.Body.Close()

	if len(expectations) != 0 {
		t.Fatalf("not all expectations were consumed: %d remaining", len(expectations))
	}
}

func TestFileAuthTransportFailsWithMissingFile(t *testing.T) {
	cfg := &fileAuthConfig{Path: "/nonexistent/path/to/secret"}

	_, err := newFileAuthTransport(cfg, http.DefaultTransport)
	if err == nil {
		t.Fatal("expected error creating transport with missing file, got nil")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("expected file not found error, got: %v", err)
	}
}

func TestFileAuthTransportFailsWithInsecurePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	cfg := &fileAuthConfig{Path: path}

	_, err := newFileAuthTransport(cfg, http.DefaultTransport)
	if err == nil {
		t.Fatal("expected error creating transport with insecure permissions, got nil")
	}
	if !strings.Contains(err.Error(), "insecure permissions") {
		t.Fatalf("expected insecure permissions error, got: %v", err)
	}
}

func TestFileAuthTransportAllowsInsecurePermissionsWithFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	cfg := &fileAuthConfig{Path: path, RelaxedPermissions: true}

	transport, err := newFileAuthTransport(cfg, http.DefaultTransport)
	if err != nil {
		t.Fatalf("unexpected error with relaxed_permissions: %v", err)
	}
	if transport == nil {
		t.Fatal("expected transport to be created with relaxed_permissions")
	}
}
