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

package compose

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

type TestRunner struct {
	Service  string
	Options  map[string][]string
	Parallel bool
}

func (r *TestRunner) scenarios() []map[string]string {
	n := 1
	for _, values := range r.Options {
		n *= len(values)
	}

	scenarios := make([]map[string]string, n)
	for variable, values := range r.Options {
		v := 0
		for i, s := range scenarios {
			if s == nil {
				s = make(map[string]string)
				scenarios[i] = s
			}
			s[variable] = values[v]
			v = (v + 1) % len(values)
		}
	}

	return scenarios
}

func (r *TestRunner) Run(t *testing.T, tests ...func(t *testing.T)) {
	for _, s := range r.scenarios() {
		var vars []string
		for k, v := range s {
			os.Setenv(k, v)
			vars = append(vars, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(vars)
		desc := strings.Join(vars, ",")

		t.Run(desc, func(t *testing.T) {
			EnsureUp(t, r.Service)
			for _, test := range tests {
				t.Run(desc, test)
			}
			// Down(t, r.Service)
		})

	}
}
