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
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type LookupFunc func(string) *monitoring.Registry

// RouteOption customizes the routes created by NewWithDefaultRoutes.
type RouteOption func(*routeOptions)

type routeOptions struct {
	statsExtra func() mapstr.M
}

// WithStatsExtra adds the metrics returned by extra to the /stats response,
// without overwriting metrics the stats registry already reports. It lets a
// process share metrics that are identical for every Beat in it, such as system
// metrics, instead of each Beat registering and collecting its own copy. extra
// is only called when /stats is requested, and may return nil.
func WithStatsExtra(extra func() mapstr.M) RouteOption {
	return func(o *routeOptions) { o.statsExtra = extra }
}

// NewWithDefaultRoutes creates a new server with default API routes.
func NewWithDefaultRoutes(log *logp.Logger, config *config.C, mon beatmonitoring.Monitoring, opts ...RouteOption) (*Server, error) {
	var options routeOptions
	for _, opt := range opts {
		opt(&options)
	}

	api, err := New(log, config)
	if err != nil {
		return nil, err
	}

	err = errors.Join(
		api.AttachHandler("/", makeRootAPIHandler(makeAPIHandler(mon.InfoRegistry()))),
		api.AttachHandler("/state", makeAPIHandler(mon.StateRegistry())),
		api.AttachHandler("/stats", makeStatsHandler(mon.StatsRegistry(), options.statsExtra)),
		api.AttachHandler("/dataset", makeAPIHandler(mon.InputsRegistry())),
	)
	if err != nil {
		return nil, err
	}

	return api, nil
}

func makeRootAPIHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}
}

func makeAPIHandler(registry *monitoring.Registry) http.HandlerFunc {
	return makeStatsHandler(registry, nil)
}

// makeStatsHandler serves a snapshot of registry, plus the metrics returned by
// extra if it is not nil.
func makeStatsHandler(registry *monitoring.Registry, extra func() mapstr.M) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(
			registry,
			monitoring.Full,
			false,
		)
		if extra != nil {
			mapstr.M(data).DeepUpdateNoOverwrite(extra())
		}

		prettyPrint(w, data, r.URL)
	}
}

func prettyPrint(w http.ResponseWriter, data mapstr.M, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprint(w, data.StringToPrint())
	} else {
		fmt.Fprint(w, data.String())
	}
}
