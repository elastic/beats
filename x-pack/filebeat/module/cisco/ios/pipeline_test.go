// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ios_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/script/javascript"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/validator"

	// Register JS "require" modules.
	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module"
	// Register required processors.
	_ "github.com/elastic/beats/v7/libbeat/cmd/instance"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

var logInputHeaders = []string{
	"Feb  8 04:00:48 10.100.4.2 585917: Feb  8 04:00:47.272: ",
	"Jun 20 02:42:16 10.100.4.2 1663310: Jun 20 02:42:15.330: ",
}

type testCase struct {
	message   string
	validator validator.Validator
}

var testCases = []testCase{
	{
		"%SEC-6-IPACCESSLOGP: list 100 denied udp 198.51.100.1(55934) -> 198.51.100.255(15600), 1 packet",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.access_list": "100",
			"cisco.ios.facility":    "SEC",
			"destination.ip":        "198.51.100.255",
			"destination.port":      int64(15600),
			"event.category":        []string{"network", "network_traffic"},
			"event.code":            "IPACCESSLOGP",
			"event.original":        isdef.IsNonEmptyString,
			"event.outcome":         "deny",
			"event.severity":        int64(6),
			"event.type":            []string{"connection", "firewall"},
			"log.level":             "informational",
			"message":               "list 100 denied udp 198.51.100.1(55934) -> 198.51.100.255(15600), 1 packet",
			"network.community_id":  isdef.IsNonEmptyString,
			"network.packets":       int64(1),
			"network.transport":     "udp",
			"source.ip":             "198.51.100.1",
			"source.packets":        int64(1),
			"source.port":           int64(55934),
		}),
	},

	{
		"%SEC-6-IPACCESSLOGDP: list 100 denied icmp 198.51.100.1 -> 198.51.100.2 (3/5), 1 packet",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.access_list": "100",
			"cisco.ios.facility":    "SEC",
			"destination.ip":        "198.51.100.2",
			"event.category":        []string{"network", "network_traffic"},
			"event.code":            "IPACCESSLOGDP",
			"event.original":        isdef.IsNonEmptyString,
			"event.outcome":         "deny",
			"event.severity":        int64(6),
			"event.type":            []string{"connection", "firewall"},
			"icmp.code":             "5",
			"icmp.type":             "3",
			"log.level":             "informational",
			"message":               "list 100 denied icmp 198.51.100.1 -> 198.51.100.2 (3/5), 1 packet",
			"network.community_id":  isdef.IsNonEmptyString,
			"network.packets":       int64(1),
			"network.transport":     "icmp",
			"source.ip":             "198.51.100.1",
			"source.packets":        int64(1),
		}),
	},

	{
		"%SEC-6-IPACCESSLOGRP: list 170 denied igmp 198.51.100.1 -> 224.168.168.168, 1 packet",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.access_list": "170",
			"cisco.ios.facility":    "SEC",
			"destination.ip":        "224.168.168.168",
			"event.category":        []string{"network", "network_traffic"},
			"event.code":            "IPACCESSLOGRP",
			"event.original":        isdef.IsNonEmptyString,
			"event.outcome":         "deny",
			"event.severity":        int64(6),
			"event.type":            []string{"connection", "firewall"},
			"log.level":             "informational",
			"message":               "list 170 denied igmp 198.51.100.1 -> 224.168.168.168, 1 packet",
			"network.community_id":  isdef.IsNonEmptyString,
			"network.packets":       int64(1),
			"network.transport":     "igmp",
			"source.ip":             "198.51.100.1",
			"source.packets":        int64(1),
		}),
	},

	{
		"%SEC-6-IPACCESSLOGSP: list INBOUND-ON-AP denied igmp 198.51.100.1 -> 224.0.0.2 (20), 1 packet",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.access_list": "INBOUND-ON-AP",
			"cisco.ios.facility":    "SEC",
			"destination.ip":        "224.0.0.2",
			"event.category":        []string{"network", "network_traffic"},
			"event.code":            "IPACCESSLOGSP",
			"event.original":        isdef.IsNonEmptyString,
			"event.outcome":         "deny",
			"event.severity":        int64(6),
			"event.type":            []string{"connection", "firewall"},
			"igmp.type":             "20",
			"log.level":             "informational",
			"message":               "list INBOUND-ON-AP denied igmp 198.51.100.1 -> 224.0.0.2 (20), 1 packet",
			"network.community_id":  isdef.IsNonEmptyString,
			"network.packets":       int64(1),
			"network.transport":     "igmp",
			"source.ip":             "198.51.100.1",
			"source.packets":        int64(1),
		}),
	},

	{
		"%SEC-6-IPACCESSLOGNP: list 1 permitted 0 198.51.100.1 -> 239.10.10.10, 1 packet",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.access_list": "1",
			"cisco.ios.facility":    "SEC",
			"destination.ip":        "239.10.10.10",
			"event.category":        []string{"network", "network_traffic"},
			"event.code":            "IPACCESSLOGNP",
			"event.original":        isdef.IsNonEmptyString,
			"event.outcome":         "allow",
			"event.severity":        int64(6),
			"event.type":            []string{"connection", "firewall"},
			"log.level":             "informational",
			"message":               "list 1 permitted 0 198.51.100.1 -> 239.10.10.10, 1 packet",
			"network.community_id":  isdef.IsNonEmptyString,
			"network.packets":       int64(1),
			"network.iana_number":   "0",
			"source.ip":             "198.51.100.1",
			"source.packets":        int64(1),
		}),
	},

	{
		"%SEC-6-IPACCESSLOGRL: access-list logging rate-limited or missed 18 packets",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.facility": "SEC",
			"event.code":         "IPACCESSLOGRL",
			"event.original":     isdef.IsNonEmptyString,
			"event.severity":     int64(6),
			"log.level":          "informational",
			"message":            "access-list logging rate-limited or missed 18 packets",
		}),
	},

	{
		"%IPV6-6-ACCESSLOGP: list ACL-IPv6-E0/0-IN/10 permitted tcp 2001:DB8::3(1027) -> 2001:DB8:1000::1(22), 9 packets",
		lookslike.MustCompile(map[string]interface{}{
			"cisco.ios.facility": "IPV6",
			"event.code":         "ACCESSLOGP",
			"event.original":     isdef.IsNonEmptyString,
			"event.severity":     int64(6),
			"log.level":          "informational",
			"message":            "list ACL-IPv6-E0/0-IN/10 permitted tcp 2001:DB8::3(1027) -> 2001:DB8:1000::1(22), 9 packets",
		}),
	},
}

func TestFilebeatSyslogCisco(t *testing.T) {
	logp.TestingSetup()

	p, err := javascript.NewFromConfig(
		javascript.Config{File: "config/pipeline.js"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	testInput(t, "syslog", p)
	testInput(t, "log", p)
}

func testInput(t *testing.T, input string, p processors.Processor) {
	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%s/%d", input, i), func(t *testing.T) {
			if input == "log" {
				tc.message = logInputHeaders[i%len(logInputHeaders)] + tc.message
			}

			e := &beat.Event{
				Fields: common.MapStr{
					"message": tc.message,
					"input": common.MapStr{
						"type": input,
					},
				},
			}

			out, err := p.Run(e)
			if err != nil {
				t.Fatalf("%+v", err)
			}
			if out == nil {
				t.Fatal("event was dropped")
			}

			if testing.Verbose() {
				data, err := json.MarshalIndent(out.Fields, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				t.Log(string(data))
			}

			if results := tc.validator(e.Fields); !results.Valid {
				for _, err := range results.Errors() {
					t.Error(err)
				}
			}
		})
	}
}

func BenchmarkPipeline(b *testing.B) {
	p, err := javascript.NewFromConfig(
		javascript.Config{File: "config/pipeline.js"},
		nil,
	)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e := beat.Event{
			Fields: common.MapStr{
				"message": testCases[i%len(testCases)].message,
				"input": common.MapStr{
					"type": "syslog",
				},
			},
		}

		_, err := p.Run(&e)
		if err != nil {
			b.Fatal(err)
		}
	}
}
