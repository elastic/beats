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

package api

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

type handlerFunc func(http.ResponseWriter, *http.Request)
type lookupFunc func(string) *monitoring.Namespace

// NewWithDefaultRoutes creates a new server with default API routes.
func NewWithDefaultRoutes(log *logp.Logger, config *common.Config, ns lookupFunc) (*Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", makeRootAPIHandler(makeAPIHandler(ns("info"))))
	mux.HandleFunc("/state", makeAPIHandler(ns("state")))
	mux.HandleFunc("/stats", makeAPIHandler(ns("stats")))
	mux.HandleFunc("/dataset", makeAPIHandler(ns("dataset")))
	return New(log, mux, config)
}

func makeRootAPIHandler(handler handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}
}

func makeAPIHandler(ns *monitoring.Namespace) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(
			ns.GetRegistry(),
			monitoring.Full,
			false,
		)

		prettyPrint(w, data, r.URL)
	}
}

func prettyPrint(w http.ResponseWriter, data common.MapStr, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprintf(w, data.StringToPrint())
	} else {
		fmt.Fprintf(w, data.String())
	}
}
