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
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

type handlerFunc func(http.ResponseWriter, *http.Request)
type lookupFunc func(string) *monitoring.Namespace

var (
	routes = &Routes{
		routes: make(map[string]handlerFunc),
		smux:   http.NewServeMux(),
	}
)

func init() {
	routes.smux.HandleFunc("/", routes.handle)
}

// NewWithDefaultRoutes creates a new server with default API routes.
func NewWithDefaultRoutes(log *logp.Logger, config *common.Config, ns lookupFunc) (*Server, error) {
	defaultRoutes := map[string]handlerFunc{
		"/":        makeRootAPIHandler(makeAPIHandler(ns("info"))),
		"/state":   makeAPIHandler(ns("state")),
		"/stats":   makeAPIHandler(ns("stats")),
		"/dataset": makeAPIHandler(ns("dataset")),
	}
	for api, h := range defaultRoutes {
		if err := routes.register(api, h); err != nil {
			return nil, err
		}
	}
	if log == nil {
		log = logp.NewLogger("")
	}
	if routes.log == nil {
		routes.log = log
	}
	return New(log, routes.smux, config)
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

type Routes struct {
	routes map[string]handlerFunc
	log    *logp.Logger
	smux   *http.ServeMux
	mux    sync.RWMutex
}

func (d *Routes) handle(w http.ResponseWriter, r *http.Request) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	if h, exist := d.routes[r.URL.Path]; exist {
		h(w, r)
		return
	}

	http.NotFound(w, r)
}

func (d *Routes) register(api string, h handlerFunc) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	if !strings.HasPrefix(api, "/") {
		return fmt.Errorf("route should starts with /")
	}
	if _, exist := d.routes[api]; exist {
		err := fmt.Errorf("route %s is already in use", api)
		d.log.Error(err.Error())
		return err
	}
	d.routes[api] = h
	return nil
}

func (d *Routes) deregister(api string) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	if _, exist := d.routes[api]; !exist {
		return fmt.Errorf("route %s is not registered", api)
	}
	delete(d.routes, api)
	return nil
}

// Register registers an API and its http handler
func Register(api string, h handlerFunc) error {
	return routes.register(api, h)
}

// Deregister deregisters an API
func Deregister(api string) error {
	return routes.deregister(api)
}
