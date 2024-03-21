// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"

	"github.com/stretchr/testify/assert"

	confpkg "github.com/elastic/elastic-agent-libs/config"
)

func Test_validateConfig(t *testing.T) {
	testCases := []struct {
		name      string // Sub-test name.
		config    config // Load config parameters.
		wantError string // Expected error
	}{
		{
			name: "valid config",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				SessionName:     "MySession-DNSServer",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
				MatchAllKeyword: "0x0000000000000000",
			},
		},
		{
			name: "minimal config",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
		},
		{
			name: "missing source config",
			config: config{
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "provider, existing logfile or running session must be set",
		},
		{
			name: "invalid trace level",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				TraceLevel:      "failed",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "invalid Trace Level value 'failed'",
		},
		{
			name: "conflict provider GUID and name",
			config: config{
				ProviderGUID:    "{eb79061a-a566-4698-1234-3ed2807033a0}",
				ProviderName:    "Microsoft-Windows-DNSServer",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: provider GUID and provider name cannot be defined together",
		},
		{
			name: "conflict provider GUID and logfile",
			config: config{
				ProviderGUID:    "{eb79061a-a566-4698-1234-3ed2807033a0}",
				Logfile:         "C:\\Windows\\System32\\winevt\\File.etl",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: provider GUID and file cannot be defined together",
		},
		{
			name: "conflict provider GUID and session",
			config: config{
				ProviderGUID:    "{eb79061a-a566-4698-1234-3ed2807033a0}",
				Session:         "EventLog-Application",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: provider GUID and existing session cannot be defined together",
		},
		{
			name: "conflict provider name and logfile",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				Logfile:         "C:\\Windows\\System32\\winevt\\File.etl",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: provider name and file cannot be defined together",
		},
		{
			name: "conflict provider name and session",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				Session:         "EventLog-Application",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: provider name and existing session cannot be defined together",
		},
		{
			name: "conflict logfile and session",
			config: config{
				Logfile:         "C:\\Windows\\System32\\winevt\\File.etl",
				Session:         "EventLog-Application",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xffffffffffffffff",
			},
			wantError: "configuration constraint error: file and existing session cannot be defined together",
		},
		{
			name: "invalid match any keyword",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				TraceLevel:      "verbose",
				MatchAnyKeyword: "0xfffffffffffffffff",
			},
			wantError: "invalid match_any_keyword value '0xfffffffffffffffff'",
		},
		{
			name: "invalid match all keyword",
			config: config{
				ProviderName:    "Microsoft-Windows-DNSServer",
				TraceLevel:      "verbose",
				MatchAllKeyword: "0xffff",
			},
			wantError: "invalid match_any_keyword value '0xffff'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := confpkg.MustNewConfigFrom(tc.config)
			config := defaultConfig()
			err := c.Unpack(&config)

			// Validate responses
			if tc.wantError != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.wantError)
				} else {
					t.Fatalf("Configuration validation failed. No returned error while expecting '%s'", tc.wantError)
				}
			} else {
				if err != nil {
					t.Fatalf("Configuration validation failed. No error expected but got '%v'", err)
				}
			}
		})
	}
}
