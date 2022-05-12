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

package inputest

import (
	"testing"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Loader wraps the input Loader in order to provide additional methods for reuse in tests.
type Loader struct {
	t testing.TB
	*v2.Loader
}

// MustNewTestLoader creates a new Loader. The test fails with fatal if the
// NewLoader constructor function returns an error.
func MustNewTestLoader(t testing.TB, plugins []v2.Plugin, typeField, defaultType string) *Loader {
	l, err := v2.NewLoader(logp.NewLogger("test"), plugins, typeField, defaultType)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}
	return &Loader{t: t, Loader: l}
}

// MustConfigure confiures a new input. The test fails with t.Fatal if the
// operation failed.
func (l *Loader) MustConfigure(cfg *conf.C) v2.Input {
	i, err := l.Configure(cfg)
	if err != nil {
		l.t.Fatalf("Failed to create the input: %v", err)
	}
	return i
}
