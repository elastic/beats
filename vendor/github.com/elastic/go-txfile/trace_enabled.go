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

// +build tracing

package txfile

import (
	"github.com/elastic/go-txfile/internal/tracelog"
)

var (
	tracers      []tracer
	activeTracer tracer
)

func init() {
	logTracer = tracelog.Get("txfile")
	activeTracer = logTracer
}

func pushTracer(t tracer) {
	tracers = append(tracers, activeTracer)
	activeTracer = t
}

func popTracer() {
	i := len(tracers) - 1
	activeTracer = tracers[i]
	tracers = tracers[:i]
}

func traceln(vs ...interface{}) {
	activeTracer.Println(vs...)
}

func tracef(s string, vs ...interface{}) {
	activeTracer.Printf(s, vs...)
}
