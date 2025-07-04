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

// InputNested is a sentinel value that can be set on the "input" (type) field
// of nested inputs, to indicate that the registry contains additional inputs
// that should be reported at the top level.
const InputNested = "__NESTED__"

const (
	route           = "/inputs"
	contentType     = "Content-Type"
	applicationJSON = "application/json; charset=utf-8"
)

type handler struct {
	globalReg *monitoring.Registry
	localReg  *monitoring.Registry
}

// AttachHandler attaches an HTTP handler to the given mux.Router to handle
// requests to /inputs. It will publish the metrics registered in the global
// 'dataset' metrics namespace and on reg.
func AttachHandler(
	r *mux.Router,
	reg *monitoring.Registry,
) error {
	return attachHandler(r, globalRegistry(), reg)
}

func attachHandler(r *mux.Router, global *monitoring.Registry, local *monitoring.Registry) error {
	h := &handler{globalReg: global, localReg: local}
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

	filtered := filteredSnapshot(h.globalReg, h.localReg, requestedType)

	w.Header().Set(contentType, applicationJSON)
	serveJSON(w, filtered, requestedPretty)
}

func filteredSnapshot(
	global *monitoring.Registry,
	local *monitoring.Registry,
	requestedType string) []map[string]any {
	// 1st collect all input metrics.
	inputs := inputMetricsFromRegistry(global)
	// If there is an id collision, metrics in the local registry take precedence
	for id, value := range inputMetricsFromRegistry(local) {
		inputs[id] = value
	}

	// Now collect all that match the requested type
	selected := make([]map[string]any, 0)
	for _, table := range inputs {
		if requestedType == "" || strings.EqualFold(table.input, requestedType) {
			selected = append(selected, table.data)
		}
	}
	return selected
}

type inputMetricsTable struct {
	id    string
	input string
	path  string // The path of this input's registry under its root metrics registry
	data  map[string]any
}

// Finds all valid input sub-registries within the given registry (including
// nested ones) and returns them as a map keyed by input id, with all inputs
// at the top level.
func inputMetricsFromRegistry(registry *monitoring.Registry) map[string]inputMetricsTable {
	metrics := monitoring.CollectStructSnapshot(registry, monitoring.Full, false)
	result := map[string]inputMetricsTable{}
	addInputMetrics(result, metrics, nil)
	return result
}

// A helper that iterates over the entries in "from" looking for
// valid input metrics tables, adding them to "to", and recurses on
// any that are tagged with InputNested.
func addInputMetrics(to map[string]inputMetricsTable, from map[string]any, pathPrefix []string) {
	for key, value := range from {
		// A valid input metrics table must be a string-keyed map with string
		// values for the "id" and "input" keys. An "input" value of InputNested
		// indicates that this is a container input and only its child registries
		// should be included.
		data, ok := value.(map[string]any)
		if !ok {
			continue
		}

		id, ok := data["id"].(string)
		if !ok || id == "" {
			continue
		}

		input, ok := data["input"].(string)
		if !ok || input == "" {
			continue
		}

		inputPath := append(pathPrefix, key)
		if input == InputNested {
			// Add the contents of this entry recursively
			addInputMetrics(to, data, inputPath)
		} else {
			to[id] = inputMetricsTable{
				id:    id,
				input: input,
				path:  strings.Join(inputPath, "."),
				data:  data,
			}
		}
	}
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
