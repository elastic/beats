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

package elasticsearch

import (
	_ "embed"
	"testing"

	"github.com/elastic/elastic-agent-libs/config"
)

//go:embed testdata/filebeat.yml
var beatYAMLCfg string

func TestToOtelConfig(t *testing.T) {
	beatCfg := config.MustNewConfigFrom(beatYAMLCfg)

	otelCfg, err := ToOtelConfig(beatCfg)
	if err != nil {
		t.Fatalf("could not convert Beat config to OTel elasicsearch exporter: %s", err)
	}

	got, want := string(otelCfg.Authentication.Password), "password"
	if got != want {
		t.Errorf("expecting password to be 'password', got '%s' instead", got)
	}

	got, want = otelCfg.Authentication.User, "elastic-cloud"
	if got != want {
		t.Errorf("expecting User %q, got %q", want, got)
	}
}
