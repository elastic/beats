// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var configTests = []struct {
	name    string
	config  map[string]any
	wantErr error
}{
	{
		name: "invalid_url_scheme",
		config: map[string]any{
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
		config: map[string]any{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
		},
		wantErr: fmt.Errorf("missing required field accessing 'url'"),
	},
	{
		name: "invalid_program",
		config: map[string]any{
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
		config: map[string]any{
			"regexp": map[string]any{
				"products":  "(?i)(xq>)d+)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("failed to check regular expressions: error parsing regexp: unexpected ): `(?i)(xq>)d+)` accessing config"),
	},
	{
		name: "valid_regexps",
		config: map[string]any{
			"regexp": map[string]any{
				"products":  "(?i)(Elasticsearch|Beats|Logstash|Kibana)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_config",
		config: map[string]any{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"url": "wss://localhost:443/v1/stream",
			"regexp": map[string]any{
				"products":  "(?i)(Elasticsearch|Beats|Logstash|Kibana)",
				"solutions": "(?i)(Search|Observability|Security)",
			},
			"state": map[string]any{
				"cursor": map[string]int{
					"last_updated": 1502908200,
				},
			},
		},
	},
	{
		name: "invalid_retry_wait_min_greater_than_wait_max",
		config: map[string]any{
			"retry": map[string]any{
				"max_attempts": 3,
				"wait_min":     "3s",
				"wait_max":     "2s",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("wait_min must be less than or equal to wait_max accessing config"),
	},
	{
		name: "invalid_retry_max_attempts_eq_zero",
		config: map[string]any{
			"retry": map[string]any{
				"max_attempts": 0,
				"wait_min":     "1s",
				"wait_max":     "2s",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("max_attempts must be greater than zero accessing config"),
	},
	{
		name: "valid_retry",
		config: map[string]any{
			"retry": map[string]any{
				"max_attempts": 3,
				"wait_min":     "2s",
				"wait_max":     "5s",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_retry_with_infinite",
		config: map[string]any{
			"retry": map[string]any{
				"infinite_retries": true,
				"max_attempts":     0,
				"wait_min":         "1s",
				"wait_max":         "2s",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_authStyle_default",
		config: map[string]any{
			"auth": map[string]any{
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"token_url":     "https://localhost:443/token",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_authStyle_in_params",
		config: map[string]any{
			"auth": map[string]any{
				"auth_style":    "in_params",
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"token_url":     "https://localhost:443/token",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "valid_authStyle_in_header",
		config: map[string]any{
			"auth": map[string]any{
				"auth_style":    "in_header",
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"token_url":     "https://localhost:443/token",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "invalid_authStyle",
		config: map[string]any{
			"auth": map[string]any{
				"auth_style":    "in_query",
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"token_url":     "https://localhost:443/token",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("unsupported auth style: in_query accessing config"),
	},
	{
		name: "valid_tokenExpiryBuffer",
		config: map[string]any{
			"auth": map[string]any{
				"client_id":           "a_client_id",
				"client_secret":       "a_client_secret",
				"token_url":           "https://localhost:443/token",
				"token_expiry_buffer": "5m",
			},
			"url": "wss://localhost:443/v1/stream",
		},
	},
	{
		name: "invalid_tokenExpiryBuffer",
		config: map[string]any{
			"auth": map[string]any{
				"client_id":           "a_client_id",
				"client_secret":       "a_client_secret",
				"token_url":           "https://localhost:443/token",
				"token_expiry_buffer": "-1s",
			},
			"url": "wss://localhost:443/v1/stream",
		},
		wantErr: fmt.Errorf("requires duration >= 0 accessing 'auth.token_expiry_buffer'"),
	},
}

func TestConfig(t *testing.T) {
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

func TestConfiguredReadControlDeadline(t *testing.T) {
	for _, tc := range []struct {
		name          string
		writeDeadline string
		want          time.Duration
	}{
		{"default", "", 30 * time.Second},
		{"custom", "25s", 75 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keepAlive := map[string]any{"enable": true, "interval": "15s"}
			if tc.writeDeadline != "" {
				keepAlive["write_control_deadline"] = tc.writeDeadline
			}
			cfg := conf.MustNewConfigFrom(map[string]any{"url": "wss://example.org/stream", "keep_alive": keepAlive})
			sources, _, err := cursorConfigure(cfg, logptest.NewTestingLogger(t, "streaming"))
			if err != nil {
				t.Fatal(err)
			}
			if !assert.Len(t, sources, 1, "expected one configured source") {
				return
			}
			src, ok := sources[0].(*source)
			if !assert.True(t, ok, "configured source must be a streaming source") {
				return
			}
			assert.Equal(t, tc.want, src.cfg.KeepAlive.readControlDeadline, "read deadline must reflect the effective write deadline")
		})
	}
}

// TestKeepAliveConfiguration exercises the two unpack passes used by Create:
// a zero-valued config followed by cursorConfigure's defaulted config.
func TestKeepAliveConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		streamType string
		keepAlive  map[string]any
		wantErr    string
	}{
		{name: "omitted"},
		{name: "disabled", keepAlive: map[string]any{"enable": false, "write_control_deadline": "0s", "interval": "0s"}},
		{name: "default_deadline", keepAlive: map[string]any{"enable": true, "interval": "15s"}},
		{name: "default_interval", keepAlive: map[string]any{"enable": true, "write_control_deadline": "25s"}},
		{name: "default_boundary", keepAlive: map[string]any{"enable": true}, wantErr: "keep_alive.interval must be less than three times keep_alive.write_control_deadline"},
		{name: "zero_deadline", keepAlive: map[string]any{"enable": true, "write_control_deadline": "0s"}, wantErr: "keep_alive.write_control_deadline must be greater than zero"},
		{name: "negative_deadline", keepAlive: map[string]any{"enable": true, "write_control_deadline": "-1s"}, wantErr: "keep_alive.write_control_deadline must be greater than zero"},
		{name: "small_deadline", keepAlive: map[string]any{"enable": true, "write_control_deadline": "10ms"}, wantErr: "keep_alive.interval must be less than three times keep_alive.write_control_deadline"},
		{name: "equal_interval", keepAlive: map[string]any{"enable": true, "write_control_deadline": "25s", "interval": "75s"}, wantErr: "keep_alive.interval must be less than three times keep_alive.write_control_deadline"},
		{name: "long_interval", keepAlive: map[string]any{"enable": true, "write_control_deadline": "25s", "interval": "76s"}, wantErr: "keep_alive.interval must be less than three times keep_alive.write_control_deadline"},
		{name: "just_below_boundary", keepAlive: map[string]any{"enable": true, "write_control_deadline": "25s", "interval": "74.999999999s"}},
		{name: "small_valid_pair", keepAlive: map[string]any{"enable": true, "write_control_deadline": "10ms", "interval": "10ms"}},
		{name: "zero_interval", keepAlive: map[string]any{"enable": true, "interval": "0s"}, wantErr: "keep_alive.interval must be greater than zero"},
		{name: "negative_interval", keepAlive: map[string]any{"enable": true, "interval": "-1s"}, wantErr: "keep_alive.interval must be greater than zero"},
		{name: "overflow_deadline", keepAlive: map[string]any{"enable": true, "write_control_deadline": "3074457345618258603ns"}, wantErr: "keep_alive.write_control_deadline is too large: three times its value must fit in a duration"},
		{name: "largest_deadline", keepAlive: map[string]any{"enable": true, "write_control_deadline": "3074457345618258602ns"}},
		{name: "explicit_websocket", streamType: "websocket", keepAlive: map[string]any{"enable": true, "interval": "15s"}},
		{name: "crowdstrike_ignores_keep_alive", streamType: "crowdstrike", keepAlive: map[string]any{"enable": true, "write_control_deadline": "0s"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := "wss://example.org/stream"
			if tc.streamType == "crowdstrike" {
				url = "https://example.org/stream"
			}
			settings := map[string]any{"url": url, "stream_type": tc.streamType}
			if tc.keepAlive != nil {
				settings["keep_alive"] = tc.keepAlive
			}
			cfg := conf.MustNewConfigFrom(settings)
			var initial config
			if !assert.NoError(t, cfg.Unpack(&initial), "the initial unpack must allow unset durations") {
				return
			}
			sources, _, err := cursorConfigure(cfg, logptest.NewTestingLogger(t, "streaming"))
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr, "invalid effective keep-alive settings must be rejected")
				assert.Empty(t, sources, "invalid settings must not create a source")
				return
			}
			if assert.NoError(t, err, "valid effective settings must configure successfully") {
				assert.Len(t, sources, 1, "valid settings must create a source")
			}
		})
	}
}
