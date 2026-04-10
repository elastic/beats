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

package dissect

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestPrefixWithIndirectField verifies that dynamically-created keys
// from indirect fields (%{?name}=%{&name}) are still prefixed correctly.
func TestPrefixWithIndirectField(t *testing.T) {
	conf := map[string]interface{}{
		"tokenizer":     `%{?k1}=%{&k1} msg="%{message}"`,
		"field":         "message",
		"target_prefix": "dissect",
	}
	c, _ := cfg.NewConfigFrom(conf)
	p, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	event := &beat.Event{
		Fields: mapstr.M{
			"message": `id=7736 msg="hello"`,
		},
	}

	result, err := p.Run(event)
	if err != nil {
		t.Fatal(err)
	}

	// The indirect field creates a dynamic key "id" with value "7736".
	// With target_prefix="dissect", it should become "dissect.id".
	val, err := result.GetValue("dissect.id")
	if err != nil {
		t.Fatalf("expected dissect.id to exist: %v", err)
	}
	if val != "7736" {
		t.Fatalf("expected dissect.id=7736, got %v", val)
	}

	// Also verify the static field
	val, err = result.GetValue("dissect.message")
	if err != nil {
		t.Fatalf("expected dissect.message to exist: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected dissect.message=hello, got %v", val)
	}
}

// BenchmarkDissectProcessor benchmarks the full processor Run path
// with the dissector already constructed (the real hot path).
func BenchmarkDissectProcessor(b *testing.B) {
	tests := []struct {
		name    string
		tok     string
		msg     string
		prefix  string
	}{
		{
			name:   "6_fields_default_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect",
		},
		{
			name:   "6_fields_with_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect",
		},
		{
			name:   "6_fields_nested_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect.parsed",
		},
		{
			// Envoyproxy-style access log with default prefix "dissect"
			// 10 extracted fields — realistic complex pattern
			name:   "envoy_access_log_default_prefix",
			tok:    `%{log_type} [%{timestamp}] "%{method} %{path} %{proto}" %{response_code} %{response_flags} %{bytes_received} %{bytes_sent} %{duration} %{upstream_service_time}`,
			msg:    `ACCESS [2026-04-08T12:00:00.000Z] "GET /api/v1/users HTTP/1.1" 200 - 0 1234 42 38`,
			prefix: "dissect",
		},
		{
			// Cisco ASA 106001 pattern — real ECS dotted field names, no prefix
			name:   "cisco_asa_ecs_no_prefix",
			tok:    `%{network.direction} %{network.transport} connection %{event.outcome} from %{source.address}/%{source.port} to %{destination.address}/%{destination.port} flags %{} on interface %{observer.ingress.interface.name}`,
			msg:    `Inbound TCP connection permitted from 192.168.1.100/44523 to 10.0.0.1/443 flags SYN on interface outside`,
			prefix: "",
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			conf := map[string]interface{}{
				"tokenizer":     tc.tok,
				"field":         "message",
				"target_prefix": tc.prefix,
			}
			c, _ := cfg.NewConfigFrom(conf)
			p, err := NewProcessor(c, logptest.NewTestingLogger(b, ""))
			if err != nil {
				b.Fatal(err)
			}

			event := &beat.Event{
				Fields: mapstr.M{
					"message": tc.msg,
				},
			}

			// Warm up
			if _, err := p.Run(event); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				// Reset the event for each iteration
				event.Fields = mapstr.M{
					"message": tc.msg,
				}
				_, _ = p.Run(event)
			}
		})
	}
}
