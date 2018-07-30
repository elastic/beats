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
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

// Start starts the metrics api endpoint on the configured host and port
func Start(cfg *common.Config) {
	cfgwarn.Experimental("Metrics endpoint is enabled.")
	config := DefaultConfig
	cfg.Unpack(&config)

	logp.Info("Starting stats endpoint")
	go func() {
		mux := http.NewServeMux()

		// register handlers
		mux.HandleFunc("/", rootHandler())
		mux.HandleFunc("/state", stateHandler)
		mux.HandleFunc("/stats", statsHandler)
		mux.HandleFunc("/dataset", datasetHandler)

		url := config.Host + ":" + strconv.Itoa(config.Port)
		logp.Info("Metrics endpoint listening on: %s", url)
		endpoint := http.ListenAndServe(url, mux)
		logp.Info("finished starting stats endpoint: %v", endpoint)
	}()
}

func rootHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return error page
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("info").GetRegistry(), monitoring.Full, false)

		print(w, data, r.URL)
	}
}

// stateHandler reports state metrics
func stateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("state").GetRegistry(), monitoring.Full, false)

	print(w, data, r.URL)
}

// statsHandler report expvar and all libbeat/monitoring metrics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("stats").GetRegistry(), monitoring.Full, false)

	print(w, data, r.URL)
}

func datasetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("dataset").GetRegistry(), monitoring.Full, false)

	print(w, data, r.URL)
}

func print(w http.ResponseWriter, data common.MapStr, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprintf(w, data.StringToPrint())
	} else {
		fmt.Fprintf(w, data.String())
	}
}
