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

//This file was contributed to by generative AI

//go:build integration

package integration

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

// NewDisabledProxy returns an enabled `DisablingProxy` that proxies
// requests to `target`.
func NewDisabledProxy(target *url.URL) *DisablingProxy {
	return &DisablingProxy{
		target:  target,
		enabled: true,
	}
}

// DisablingProxy is a HTTP proxy that can be disabled/enabled at runtime
type DisablingProxy struct {
	mu      sync.RWMutex
	enabled bool
	target  *url.URL
}

// ServeHTTP handles incoming requests and forwards them to the target if
// the proxy is enabled.
func (d *DisablingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.enabled {
		http.Error(w, "Proxy is disabled", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(d.target)
	proxy.ServeHTTP(w, r)
}

// Enable enables the proxy.
func (d *DisablingProxy) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

// Disable disables the proxy.
func (d *DisablingProxy) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
}
