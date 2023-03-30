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

package pprof

import (
	"net/http"
	"net/http/pprof"
	"runtime"

	"go.uber.org/multierr"
)

type handlerAttacher interface {
	AttachHandler(route string, h http.Handler) (err error)
}

// Config holds config information about exposing pprof.
type Config struct {
	Enabled          *bool `config:"enabled"`
	BlockProfileRate int   `config:"block_profile_rate"`
	MemProfileRate   int   `config:"mem_profile_rate"`
	MutexProfileRate int   `config:"mutex_profile_rate"`
}

// IsEnabled returns true if the pprof config is non-nil and either 'enabled'
// is not set or it is set to true.
func (c *Config) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

// SetRuntimeProfilingParameters pushes the configuration parameters into the Go runtime.
// Profiling rates should be set once, early on in the program.
func SetRuntimeProfilingParameters(cfg *Config) {
	if !cfg.IsEnabled() {
		return
	}

	runtime.SetBlockProfileRate(cfg.BlockProfileRate)
	runtime.SetMutexProfileFraction(cfg.MutexProfileRate)
	if cfg.MemProfileRate > 0 {
		runtime.MemProfileRate = cfg.MemProfileRate
	}
}

// HttpAttach attaches the /debug/pprof HTTP handlers to the given mux. It
// returns an error if any handler is already registered at the standard paths.
func HttpAttach(cfg *Config, mux handlerAttacher) error {
	if !cfg.IsEnabled() {
		return nil
	}

	const path = "/debug/pprof"
	return multierr.Combine(
		mux.AttachHandler(path+"/", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/allocs", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/block", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/goroutine", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/heap", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/mutex", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/threadcreate", http.HandlerFunc(pprof.Index)),
		mux.AttachHandler(path+"/cmdline", http.HandlerFunc(pprof.Cmdline)),
		mux.AttachHandler(path+"/profile", http.HandlerFunc(pprof.Profile)),
		mux.AttachHandler(path+"/symbol", http.HandlerFunc(pprof.Symbol)),
		mux.AttachHandler(path+"/trace", http.HandlerFunc(pprof.Trace)),
	)
}
