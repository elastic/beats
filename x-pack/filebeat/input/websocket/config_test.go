// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var configTests = []struct {
	name    string
	config  map[string]interface{}
	wantErr error
}{
	{
		name: "invalid_url_scheme",
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"url": "http://localhost:8080",
		},
		wantErr: fmt.Errorf("unsupported scheme: http accessing config"),
	},
	{
		name: "missing_url",
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
		},
		wantErr: fmt.Errorf("missing required field accessing 'url'"),
	},
	{
		name: "invalid_program",
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": has(state.cursor) && inner_body.ts > state.cursor.last_updated ? 
					[inner_body] 
					: 
					null,
				})`,
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("failed to check program: failed compilation: ERROR: <input>:3:79: found no matching overload for '_?_:_' applied to '(bool, list(dyn), null)'\n |      \"events\": has(state.cursor) && inner_body.ts > state.cursor.last_updated ? \n | ..............................................................................^ accessing config"),
	},
	{
		name: "invalid_regexps",
		config: map[string]interface{}{
			"regexp": map[string]interface{}{
				"products":  "(?i)(xq>)d+)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("failed to check regular expressions: error parsing regexp: unexpected ): `(?i)(xq>)d+)` accessing config"),
	},
	{
		name: "valid_regexps",
		config: map[string]interface{}{
			"regexp": map[string]interface{}{
				"products":  "(?i)(Elasticsearch|Beats|Logstash|Kibana)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_config",
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"url": "wss://localhost:443/v1/stream",
			"regexp": map[string]interface{}{
				"products":  "(?i)(Elasticsearch|Beats|Logstash|Kibana)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"state": map[string]interface{}{
				"cursor": map[string]int{
					"last_updated": 1502908200,
				},
			},
		},
	},
}

func TestConfig(t *testing.T) {
	logp.TestingSetup()
	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(test.config)
			conf := config{}
			// Make sure we pass the redact requirement.
			conf.Redact = &redact{}
			err := cfg.Unpack(&conf)

			switch {
			case err == nil && test.wantErr != nil:
				t.Fatalf("expected error unpacking config: %v", test.wantErr)
			case err != nil && test.wantErr == nil:
				t.Fatalf("unexpected error unpacking config: %v", err)
			case err != nil && test.wantErr != nil:
				assert.EqualError(t, err, test.wantErr.Error())
			default:
				// no error
			}
		})
	}
}

func TestRegexpConfig(t *testing.T) {
	cfg := config{
		Program: `{}`,
		URL:     &urlConfig{URL: &url.URL{Scheme: "ws"}},
		Regexps: map[string]string{"regex_cve": `[Cc][Vv][Ee]-[0-9]{4}-[0-9]{4,7}`},
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("failed to validate config with regexps: %v", err)
	}
}
