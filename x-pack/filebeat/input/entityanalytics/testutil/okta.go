// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package testutil provides shared httptest/LDAP mocks and fixtures for
// entity-analytics provider tests and OTel e2e tests.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ecokta "github.com/elastic/entcollect/provider/okta"
)

// StartOktaServer starts a TLS httptest server that serves the Okta list-users,
// groups, and devices fixtures used by provider equivalence and e2e tests.
func StartOktaServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(OktaUsersJSON())
	})

	mux.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(OktaGroupsJSON())
	})

	mux.HandleFunc("/api/v1/groups/g1/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"u1"},{"id":"u2"}]`))
	})

	mux.HandleFunc("/api/v1/groups/g2/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"u1"}]`))
	})

	mux.HandleFunc("/api/v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(OktaDevicesJSON())
	})

	mux.HandleFunc("/api/v1/devices/d1/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"created":"2026-01-01T12:00:00.000Z","managementStatus":"NOT_MANAGED","screenLockType":"NONE","user":{"id":"u1","status":"ACTIVE","created":"2026-01-01T12:00:00.000Z","lastUpdated":"2026-01-02T12:00:00.000Z","profile":{"login":"alice@example.com","email":"alice@example.com","firstName":"Alice","lastName":"Smith"}}}]`))
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// OktaUsersJSON returns the deterministic users fixture (ids u1, u2, u3).
func OktaUsersJSON() []byte {
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	return []byte(`[
		{
			"id": "u1",
			"status": "ACTIVE",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "alice@example.com", "email": "alice@example.com", "firstName": "Alice", "lastName": "Smith", "managerId": "u2"}
		},
		{
			"id": "u2",
			"status": "ACTIVE",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "bob@example.com", "email": "bob@example.com", "firstName": "Bob", "lastName": "Jones"}
		},
		{
			"id": "u3",
			"status": "DEPROVISIONED",
			"created": "2026-01-01T12:00:00.000Z",
			"activated": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "` + now.Format(ecokta.ISO8601) + `",
			"profile": {"login": "charlie@example.com", "email": "charlie@example.com", "firstName": "Charlie", "lastName": "Brown"}
		}
	]`)
}

// OktaGroupsJSON returns the deterministic groups fixture (ids g1, g2).
func OktaGroupsJSON() []byte {
	return []byte(`[
		{"id": "g1", "profile": {"name": "Staff", "description": "All staff"}},
		{"id": "g2", "profile": {"name": "Engineering", "description": "Engineering team"}}
	]`)
}

// OktaDevicesJSON returns the deterministic devices fixture (id d1).
func OktaDevicesJSON() []byte {
	return []byte(`[
		{
			"id": "d1",
			"created": "2026-01-01T12:00:00.000Z",
			"lastUpdated": "2026-01-02T12:00:00.000Z",
			"status": "ACTIVE",
			"profile": {"displayName": "Alice Laptop"},
			"resourceAlternateID": "",
			"resourceDisplayName": {"sensitive": false, "value": "Alice Laptop"},
			"resourceID": "d1",
			"resourceType": "UDDevice"
		}
	]`)
}
