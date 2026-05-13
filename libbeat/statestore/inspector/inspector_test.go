// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package inspector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
)

type testRegistry struct {
	registry *statestore.Registry
	name     string
}

func newTestRegistry(t *testing.T) testRegistry {
	t.Helper()
	reg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
	t.Cleanup(func() { _ = reg.Close() })
	return testRegistry{registry: reg, name: "test"}
}

// store returns a short-lived Store for setting up test data.
// The caller must close it when done.
func (tr testRegistry) store(t *testing.T) *statestore.Store {
	t.Helper()
	store, err := tr.registry.Get(tr.name)
	require.NoError(t, err, "failed to get test store")
	return store
}

func TestGetStates_NoStore(t *testing.T) {
	handler := New()

	req := httptest.NewRequest(http.MethodGet, "/states", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "unexpected status code")
	assert.JSONEq(t, `[]`, rec.Body.String(), "expected empty JSON array when no store is set")
}

func TestGetStates_Empty(t *testing.T) {
	tr := newTestRegistry(t)
	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodGet, "/states", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "unexpected status code")
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "unexpected content type")
	assert.JSONEq(t, `[]`, rec.Body.String(), "expected empty JSON array for an empty store")
}

func TestGetStates_WithEntries(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	require.NoError(t, store.Set("key-b", map[string]interface{}{"offset": 100}), "failed to set key-b")
	require.NoError(t, store.Set("key-a", map[string]interface{}{"offset": 200}), "failed to set key-a")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodGet, "/states", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "unexpected status code")

	var entries []stateEntry
	err := json.Unmarshal(rec.Body.Bytes(), &entries)
	require.NoError(t, err, "failed to unmarshal response body")
	require.Len(t, entries, 2, "expected exactly 2 entries")

	assert.Equal(t, "key-a", entries[0].Key, "entries should be sorted alphabetically")
	assert.Equal(t, "key-b", entries[1].Key, "entries should be sorted alphabetically")
}

func TestGetStates_Pretty(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	require.NoError(t, store.Set("k", map[string]interface{}{"v": 1}), "failed to set key")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	compact := httptest.NewRequest(http.MethodGet, "/states", nil)
	compactRec := httptest.NewRecorder()
	handler.ServeHTTP(compactRec, compact)

	pretty := httptest.NewRequest(http.MethodGet, "/states?pretty", nil)
	prettyRec := httptest.NewRecorder()
	handler.ServeHTTP(prettyRec, pretty)

	assert.JSONEq(t, compactRec.Body.String(), prettyRec.Body.String(), "both should decode to the same JSON")
	assert.NotContains(t, compactRec.Body.String(), "\n  ", "compact output should not be indented")
	assert.Contains(t, prettyRec.Body.String(), "\n  ", "pretty output should be indented")
}

func TestDeleteState(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	require.NoError(t, store.Set("to-delete", map[string]interface{}{"val": 1}), "failed to set key")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodDelete, "/states/to-delete", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "unexpected status code for DELETE")

	verifyStore := tr.store(t)
	defer verifyStore.Close()
	has, err := verifyStore.Has("to-delete")
	require.NoError(t, err, "failed to check key existence after delete")
	assert.False(t, has, "key should have been removed from the store")
}

func TestDeleteState_NoStore(t *testing.T) {
	handler := New()

	req := httptest.NewRequest(http.MethodDelete, "/states/some-key", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "expected 503 when no store is set")
}

func TestDeleteState_MissingKey(t *testing.T) {
	tr := newTestRegistry(t)
	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodDelete, "/states/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "expected bad request for empty key")
}

func TestDeleteState_KeyWithSlashes(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	key := "filestream::input-id::/var/log/syslog"
	require.NoError(t, store.Set(key, map[string]interface{}{"offset": 42}), "failed to set key with slashes")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodDelete, "/states/"+key, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "unexpected status code for DELETE with slashes in key")

	verifyStore := tr.store(t)
	defer verifyStore.Close()
	has, err := verifyStore.Has(key)
	require.NoError(t, err, "failed to check key existence after delete")
	assert.False(t, has, "key with slashes should have been removed from the store")
}

func TestDeleteState_PercentEncodedKey(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	key := "filestream::my-id::/var/log/syslog"
	require.NoError(t, store.Set(key, map[string]interface{}{"offset": 99}), "failed to set key")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	// Build the request path the same way the browser does:
	// encodeURIComponent produces percent-encoded segments.
	encodedPath := "/states/" + url.PathEscape(key)
	require.NotEqual(t, "/states/"+key, encodedPath, "encoded path should differ from the raw key (slashes must be escaped)")

	req := httptest.NewRequest(http.MethodDelete, encodedPath, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "unexpected status code for DELETE with percent-encoded key")

	verifyStore := tr.store(t)
	defer verifyStore.Close()
	has, err := verifyStore.Has(key)
	require.NoError(t, err, "failed to check key existence after delete")
	assert.False(t, has, "key should have been removed via percent-encoded DELETE request")
}

func TestGetStatesHTML(t *testing.T) {
	tr := newTestRegistry(t)
	store := tr.store(t)
	require.NoError(t, store.Set("html-key", map[string]interface{}{"offset": 10}), "failed to set key")
	require.NoError(t, store.Close())

	handler := New()
	handler.SetRegistry(tr.registry, tr.name)

	req := httptest.NewRequest(http.MethodGet, "/states.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "unexpected status code")
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"), "unexpected content type")
	assert.Contains(t, rec.Body.String(), "Active States", "page should contain the title")
	assert.Contains(t, rec.Body.String(), "html-key", "page should contain the key")
}
