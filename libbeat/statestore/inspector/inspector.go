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
	"fmt"
	"html"
	"net/http"
	"sort"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
)

// Handler serves a web-interface for inspecting and manipulating a state
// store. It is safe for concurrent use. The handler starts with no registry;
// GET endpoints return empty results and mutating endpoints return 503
// until SetRegistry is called.
//
// Each HTTP request obtains its own short-lived Store instance via
// Registry.Get, following the Store contract that forbids sharing a
// single Store across goroutines.
type Handler struct {
	mu        sync.RWMutex
	registry  *statestore.Registry
	storeName string
	mux       *http.ServeMux
}

// New creates a Handler with no backing registry. Call SetRegistry to
// provide one. The returned handler expects to receive requests with
// paths relative to its mount point (i.e. after prefix stripping).
func New() *Handler {
	h := &Handler{
		mux: http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /states", h.handleGetStates)
	h.mux.HandleFunc("GET /states.html", h.handleGetStatesHTML)
	h.mux.HandleFunc("DELETE /states/{key...}", h.handleDeleteState)
	return h
}

// SetRegistry provides the registry and store name used by all endpoints.
// Each request will call registry.Get(name) to obtain its own Store instance.
func (h *Handler) SetRegistry(registry *statestore.Registry, name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.registry = registry
	h.storeName = name
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type stateEntry struct {
	Key   string `json:"key"`
	State any    `json:"state"`
}

func (h *Handler) openStore() (*statestore.Store, error) {
	h.mu.RLock()
	reg := h.registry
	name := h.storeName
	h.mu.RUnlock()

	if reg == nil {
		return nil, nil //nolint:nilnil // nil store signals "no registry configured"
	}
	return reg.Get(name)
}

func (h *Handler) collectStates() ([]stateEntry, error) {
	store, err := h.openStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}
	if store == nil {
		return []stateEntry{}, nil
	}
	defer store.Close()

	entries := make([]stateEntry, 0)
	err = store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		var val any
		if err := dec.Decode(&val); err != nil {
			return false, fmt.Errorf("failed to decode value for key %q: %w", key, err)
		}
		entries = append(entries, stateEntry{Key: key, State: val})
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries, nil
}

func (h *Handler) handleGetStates(w http.ResponseWriter, r *http.Request) {
	entries, err := h.collectStates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	if _, ok := r.URL.Query()["pretty"]; ok {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) handleGetStatesHTML(w http.ResponseWriter, r *http.Request) {
	entries, err := h.collectStates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Active States</title>
<style>
  body { font-family: monospace; margin: 2em; background: #fafafa; }
  h1 { font-size: 1.4em; }
  table { border-collapse: collapse; width: 100%; }
  th, td { border: 1px solid #ccc; padding: 0.5em 0.75em; text-align: left; vertical-align: top; }
  th { background: #eee; }
  pre { margin: 0; white-space: pre-wrap; word-break: break-all; }
  .btn-delete { background: #c0392b; color: #fff; border: none; padding: 0.4em 1em; cursor: pointer; }
  .btn-delete:hover { background: #e74c3c; }
</style>
<script>function del(k){fetch('states/'+encodeURIComponent(k),{method:'DELETE'}).then(()=>location.reload())}</script>
</head>
<body>
<h1>Active States</h1>
<table>
<tr><th>Key</th><th>State</th><th>Delete</th></tr>
`)
	for _, e := range entries {
		stateJSON, _ := json.MarshalIndent(e.State, "", "  ")
		escapedKey := html.EscapeString(e.Key)
		escapedState := html.EscapeString(string(stateJSON))
		jsKeyJSON, _ := json.Marshal(e.Key)
		jsKey := html.EscapeString(string(jsKeyJSON))
		fmt.Fprintf(w, "<tr><td><pre>%s</pre></td><td><pre>%s</pre></td><td><button class=\"btn-delete\" onclick=\"del(%s)\">Delete</button></td></tr>\n",
			escapedKey, escapedState, jsKey)
	}
	fmt.Fprint(w, `</table>
</body>
</html>
`)
}

func (h *Handler) handleDeleteState(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	store, err := h.openStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if store == nil {
		http.Error(w, "no store configured", http.StatusServiceUnavailable)
		return
	}
	defer store.Close()

	if err := store.Remove(key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
