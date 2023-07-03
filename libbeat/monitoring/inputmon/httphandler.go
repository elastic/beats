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

package inputmon

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	route           = "/inputs"
	contentType     = "Content-Type"
	applicationJSON = "application/json; charset=utf-8"
)

type handler struct {
	registry *monitoring.Registry
}

// AttachHandler attaches an HTTP handler to the given mux.Router to handle
// requests to /inputs.
func AttachHandler(r *mux.Router) error {
	return attachHandler(r, globalRegistry())
}

func attachHandler(r *mux.Router, registry *monitoring.Registry) error {
	h := &handler{registry: registry}
	r = r.PathPrefix(route).Subrouter()
	return r.StrictSlash(true).Handle("/", validationHandler("GET", []string{"pretty", "type"}, h.allInputs)).GetError()
}

func (h *handler) allInputs(w http.ResponseWriter, req *http.Request) {
	requestedPretty, err := getPretty(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestedType, err := getType(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filtered := filteredSnapshot(h.registry, requestedType)

	w.Header().Set(contentType, applicationJSON)
	serveJSON(w, filtered, requestedPretty)
}

func filteredSnapshot(r *monitoring.Registry, requestedType string) []map[string]any {
	metrics := monitoring.CollectStructSnapshot(r, monitoring.Full, false)

	filtered := make([]map[string]any, 0, len(metrics))
	for _, ifc := range metrics {
		m, ok := ifc.(map[string]any)
		if !ok {
			continue
		}

		// Require all entries to have an 'input' and 'id' to be accessed through this API.
		if id, ok := m["id"].(string); !ok || id == "" {
			continue
		}

		if inputType, ok := m["input"].(string); !ok || (requestedType != "" && !strings.EqualFold(inputType, requestedType)) {
			continue
		}

		filtered = append(filtered, m)
	}
	return filtered
}

func serveJSON(w http.ResponseWriter, value any, pretty bool) {
	w.Header().Set(contentType, applicationJSON)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	_ = enc.Encode(value)
}

func getPretty(req *http.Request) (bool, error) {
	if !req.URL.Query().Has("pretty") {
		return false, nil
	}

	switch req.URL.Query().Get("pretty") {
	case "", "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New(`invalid value for "pretty"`)
	}
}

func getType(req *http.Request) (string, error) {
	if !req.URL.Query().Has("type") {
		return "", nil
	}

	switch typ := req.URL.Query().Get("type"); typ {
	case "":
		return "", errors.New(`"type" requires a non-empty value`)
	default:
		return strings.ToLower(typ), nil
	}
}

type queryParamHandler struct {
	allowedParams map[string]struct{}
	next          http.Handler
}

func newQueryParamHandler(queryParams []string, h http.Handler) http.Handler {
	m := make(map[string]struct{}, len(queryParams))
	for _, q := range queryParams {
		m[q] = struct{}{}
	}
	return &queryParamHandler{allowedParams: m, next: h}
}

func (h queryParamHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for q := range req.URL.Query() {
		if _, found := h.allowedParams[q]; !found {
			http.Error(w, "Unknown query param "+q, http.StatusBadRequest)
			return
		}
	}
	h.next.ServeHTTP(w, req)
}

func validationHandler(method string, queryParams []string, h http.HandlerFunc) http.Handler {
	var next http.Handler = h
	next = handlers.CompressHandler(next)
	next = newQueryParamHandler(queryParams, next)
	next = handlers.MethodHandler{method: next}
	return next
}
