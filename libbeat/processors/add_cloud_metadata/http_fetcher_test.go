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

package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestValidateResponse(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		expectErr   bool
	}{
		{
			name:        "valid plain text",
			contentType: "text/plain",
			body:        "i-0000ffac",
			expectErr:   false,
		},
		{
			name:        "valid JSON",
			contentType: "application/json",
			body:        `{"instance_id": "i-123"}`,
			expectErr:   false,
		},
		{
			name:        "no content type with valid body",
			contentType: "",
			body:        "us-east-1",
			expectErr:   false,
		},
		{
			name:        "HTML content type",
			contentType: "text/html",
			body:        "<html><body>Blocked</body></html>",
			expectErr:   true,
		},
		{
			name:        "HTML content type with charset",
			contentType: "text/html; charset=UTF-8",
			body:        "<html><body>Blocked</body></html>",
			expectErr:   true,
		},
		{
			name:        "HTML body with doctype but no content type",
			contentType: "",
			body:        `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN"><html><body>Firewall Notification</body></html>`,
			expectErr:   true,
		},
		{
			name:        "HTML body starting with html tag",
			contentType: "",
			body:        "<html><head></head><body>Blocked</body></html>",
			expectErr:   true,
		},
		{
			name:        "HTML body with leading whitespace",
			contentType: "",
			body:        "  \n  <!DOCTYPE html><html><body>Blocked</body></html>",
			expectErr:   true,
		},
		{
			name:        "HTML body case insensitive",
			contentType: "",
			body:        "<!DOCTYPE HTML><HTML><BODY>Blocked</BODY></HTML>",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp := &http.Response{
				Header: http.Header{},
			}
			if tt.contentType != "" {
				rsp.Header.Set("Content-Type", tt.contentType)
			}

			err := validateResponse(rsp, []byte(tt.body))
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

const firewallHTML = `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN">
<html><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
<style type="text/css">body {font-family: arial}</style></head>
<body><div><h1>Firewall Notification</h1>
<h2>Your access has been blocked by firewall policy.</h2></div></body></html>`

func TestRejectHTMLFromFirewall_PlainTextProvider(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(firewallHTML))
	}))
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"alibaba"},
		"host":      server.Listener.Addr().String(),
	})
	require.NoError(t, err)

	p, err := New(config, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	require.NoError(t, err)

	_, err = actual.Fields.GetValue("cloud")
	assert.Error(t, err, "cloud metadata should not be present when firewall returns HTML")
}

func TestRejectHTMLFromFirewall_JSONProvider(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(firewallHTML))
	}))
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"digitalocean"},
		"host":      server.Listener.Addr().String(),
	})
	require.NoError(t, err)

	p, err := New(config, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	require.NoError(t, err)

	_, err = actual.Fields.GetValue("cloud")
	assert.Error(t, err, "cloud metadata should not be present when firewall returns HTML")
}

func TestRejectHTMLBodyWithoutContentType(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(firewallHTML))
	}))
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"alibaba"},
		"host":      server.Listener.Addr().String(),
	})
	require.NoError(t, err)

	p, err := New(config, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	require.NoError(t, err)

	_, err = actual.Fields.GetValue("cloud")
	assert.Error(t, err, "cloud metadata should not be present when response body is HTML")
}
