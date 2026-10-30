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

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
)

type wrappedRunner struct {
	cfgfile.Runner
}

func (w wrappedRunner) Unwrap() cfgfile.Runner { return w.Runner }

func TestUnwrapRunner(t *testing.T) {
	inner := &input.Runner{}

	tests := map[string]cfgfile.Runner{
		"not wrapped":   inner,
		"wrapped":       wrappedRunner{inner},
		"wrapped twice": wrappedRunner{wrappedRunner{inner}},
	}
	for name, runner := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Same(t, inner, unwrapRunner(runner), "expected the innermost runner")
		})
	}
}
