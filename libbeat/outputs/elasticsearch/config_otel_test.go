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
	"strings"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
)

//go:embed testdata/filebeat.yml
var beatYAMLCfg string

//go:embed testdata/certs/client.crt
var clientCertPem string

//go:embed testdata/expectedCaPem.crt
var wantCAPem string

// Generating certs:
// Root CA:      openssl req -x509 -ca -sha256 -days 1825 -newkey rsa:2048 -keyout rootCA.key -out rootCA.crt -passout pass:changeme
// Server Cert:  openssl req -newkey rsa:2048 -keyout server.key -x509 -days 3650 -out server.crt -passout pass:changeme
// Client Cert:  openssl req -newkey rsa:2048 -keyout client.key -x509 -days 3650 -out client.crt -passout pass:changeme -extensions usr_cert
func TestToOtelConfig(t *testing.T) {
	beatCfg := config.MustNewConfigFrom(beatYAMLCfg)

	otelCfg, err := toOTelConfig(beatCfg)
	if err != nil {
		t.Fatalf("could not convert Beat config to OTel elasicsearch exporter: %s", err)
	}

	if otelCfg.Endpoint != "" {
		t.Errorf("OTel endpoint must be emtpy got %s", otelCfg.Endpoint)
	}

	expectedHost := "https://es-hostname.elastic.co:443"
	if len(otelCfg.Endpoints) != 1 || otelCfg.Endpoints[0] != expectedHost {
		t.Errorf("OTel endpoints must contain only %q, got %q", expectedHost, otelCfg.Endpoints)
	}

	if got, want := otelCfg.Authentication.User, "elastic-cloud"; got != want {
		t.Errorf("expecting User %q, got %q", want, got)
	}

	if got, want := string(otelCfg.Authentication.Password), "password"; got != want {
		t.Errorf("expecting password to be '%s', got '%s' instead", want, got)
	}

	// // The ES config from Beats does not allow api_key and username/password to
	// // be set at the same time, so I'm keeping this assertion commented out
	// // for now
	// if got, want := string(otelCfg.Authentication.APIKey), "secret key"; got != want {
	// 	t.Errorf("expecting api_key to be '%s', got '%s' instead", want, got)
	// }

	if got, want := otelCfg.LogsIndex, "some-index"; got != want {
		t.Errorf("expecting logs index to be '%s', got '%s' instead", want, got)
	}

	if got, want := otelCfg.Pipeline, "some-ingest-pipeline"; got != want {
		t.Errorf("expecting pipeline to be '%s', got '%s' instead", want, got)
	}

	if got, want := otelCfg.ClientConfig.ProxyURL, "https://proxy.url"; got != want {
		t.Errorf("expecting proxy URL to be '%s', got '%s' instead", want, got)
	}

	if got, want := string(otelCfg.ClientConfig.TLSSetting.CertPem), clientCertPem; got != want {
		t.Errorf("expecting client certificate %q got %q", want, got)
	}

	gotCAPem := strings.TrimSpace(string(otelCfg.ClientConfig.TLSSetting.CAPem))
	wantCAPem = strings.TrimSpace(wantCAPem)
	if gotCAPem != wantCAPem {
		t.Errorf("expecting CA PEM:\n%s\ngot:\n%s", wantCAPem, gotCAPem)
	}

	if !*otelCfg.Batcher.Enabled {
		t.Error("expecting batcher.enabled to be true")
	}

	if got, want := otelCfg.Batcher.MaxSizeItems, 42; got != want {
		t.Errorf("expecting batcher.max_size_items = %d got %d", want, got)
	}

	if !otelCfg.Retry.Enabled {
		t.Error("expecting retyr.enabled to be true")
	}

	if got, want := otelCfg.Retry.InitialInterval, time.Second*42; got != want {
		t.Errorf("expecting retry.initial_interval '%s', got '%s'", got, want)
	}

	if got, want := otelCfg.NumWorkers, 30; got != want {
		t.Errorf("expecting num_workers %d got %d", want, got)
	}

	headers := map[string]string{
		"X-Header-1":   "foo",
		"X-Bar-Header": "bar",
	}

	for k, v := range headers {
		gotV := string(otelCfg.Headers[k])
		if gotV != v {
			t.Errorf("expecting header[%s]='%s', got '%s", k, v, gotV)
		}
	}
}
