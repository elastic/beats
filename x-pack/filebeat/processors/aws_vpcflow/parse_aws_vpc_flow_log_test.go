// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
)

var updateGolden = flag.Bool("update", false, "Update golden test data.")

const (
	formatV5       = `version account-id interface-id srcaddr dstaddr srcport dstport protocol packets bytes start end action log-status vpc-id subnet-id instance-id tcp-flags type pkt-srcaddr pkt-dstaddr region az-id sublocation-type sublocation-id pkt-src-aws-service pkt-dst-aws-service flow-direction traffic-path`
	formatV5Sample = `5 64111117617 eni-069xxxxxb7a490 89.160.20.156 10.200.0.0 50041 33004 17 52 1 1616729292 1616729349 REJECT OK vpc-09676f97xxxxxb8a7 subnet-02d645xxxxxxxdbc0 i-0axxxxxx1ad77 1 IPv4 89.160.20.156 10.200.0.80 us-east-1 use1-az5 wavelength fake-id AMAZON CLOUDFRONT ingress 1`
)

func TestProcessorRun(t *testing.T) {
	t.Run("ecs_and_original-mode-v5-message", func(t *testing.T) {
		c := defaultConfig()
		c.Format = []string{
			"version account-id", // Not a match.
			formatV5,
		}
		c.Mode = ecsAndOriginalMode

		p, err := newParseAWSVPCFlowLog(c)
		require.NoError(t, err)

		assert.Contains(t, p.String(), procName+"=")
		assert.Contains(t, p.String(), formatV5)

		evt := beat.Event{
			Timestamp: time.Now().UTC(),
			Fields: map[string]interface{}{
				"message": formatV5Sample,
			},
		}

		out, err := p.Run(&evt)
		require.NoError(t, err)

		start := time.Unix(1616729292, 0).UTC()
		end := time.Unix(1616729349, 0).UTC()
		expected := mapstr.M{
			"aws": mapstr.M{
				"vpcflow": mapstr.M{
					"account_id":          "64111117617",
					"action":              "REJECT",
					"az_id":               "use1-az5",
					"bytes":               int64(1),
					"dstaddr":             "10.200.0.0",
					"dstport":             int32(33004),
					"end":                 end,
					"flow_direction":      "ingress",
					"instance_id":         "i-0axxxxxx1ad77",
					"interface_id":        "eni-069xxxxxb7a490",
					"log_status":          "OK",
					"packets":             int64(52),
					"pkt_dst_aws_service": "CLOUDFRONT",
					"pkt_dstaddr":         "10.200.0.80",
					"pkt_src_aws_service": "AMAZON",
					"pkt_srcaddr":         "89.160.20.156",
					"protocol":            int32(17),
					"region":              "us-east-1",
					"srcaddr":             "89.160.20.156",
					"srcport":             int32(50041),
					"start":               start,
					"sublocation_id":      "fake-id",
					"sublocation_type":    "wavelength",
					"subnet_id":           "subnet-02d645xxxxxxxdbc0",
					"tcp_flags":           int32(1),
					"tcp_flags_array": []string{
						"fin",
					},
					"traffic_path": int32(1),
					"type":         "IPv4",
					"version":      int32(5),
					"vpc_id":       "vpc-09676f97xxxxxb8a7",
				},
			},
			"cloud": mapstr.M{
				"account": mapstr.M{
					"id": "64111117617",
				},
				"availability_zone": "use1-az5",
				"instance": mapstr.M{
					"id": "i-0axxxxxx1ad77",
				},
				"region": "us-east-1",
			},
			"destination": mapstr.M{
				"address": "10.200.0.0",
				"ip":      "10.200.0.0",
				"port":    int32(33004),
			},
			"event": mapstr.M{
				"action":  "reject",
				"end":     end,
				"outcome": "failure",
				"start":   start,
				"type":    []string{"connection", "denied"},
			},
			"message": formatV5Sample,
			"network": mapstr.M{
				"bytes":       int64(1),
				"direction":   "ingress",
				"iana_number": "17",
				"packets":     int64(52),
				"transport":   "udp",
				"type":        "ipv4",
			},
			"related": mapstr.M{
				"ip": []string{"89.160.20.156", "10.200.0.0", "10.200.0.80"},
			},
			"source": mapstr.M{
				"address": "89.160.20.156",
				"bytes":   int64(1),
				"ip":      "89.160.20.156",
				"packets": int64(52),
				"port":    int32(50041),
			},
		}

		assert.Equal(t, end, out.Timestamp)
		if diff := cmp.Diff(expected, out.Fields); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestGoldenFile(t *testing.T) {
	testCases := readGoldenTestCase(t)

	if *updateGolden {
		// Delete existing golden files.
		goldens, _ := filepath.Glob("testdata/*.golden.*")
		for _, golden := range goldens {
			os.Remove(golden)
		}
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			c := defaultConfig()
			c.Format = []string{tc.Format}
			if tc.Mode != nil {
				c.Mode = *tc.Mode
			}

			p, err := newParseAWSVPCFlowLog(c)
			require.NoError(t, err)

			observed := make([]mapstr.M, 0, len(tc.Samples))
			for _, sample := range tc.Samples {
				evt := &beat.Event{Fields: mapstr.M{"message": sample}}
				out, err := p.Run(evt)
				require.NoError(t, err)

				if !out.Timestamp.IsZero() {
					out.Fields["@timestamp"] = out.Timestamp
				}
				observed = append(observed, out.Fields)
			}

			goldenFile := filepath.Join("testdata", tc.Name+".golden.json")
			if *updateGolden {
				writeGolden(t, goldenFile, observed)
			} else {
				expectedJSON := readGolden(t, goldenFile)

				observedJSON, err := json.Marshal(observed)
				require.NoError(t, err)

				assert.JSONEq(t, expectedJSON, string(observedJSON))
			}
		})
	}
}

type goldenTestCase struct {
	Name    string   `yaml:"-"` // Name of test.
	Mode    *mode    // Processing mode (what fields to generate).
	Format  string   // Flow log format.
	Samples []string // List of sample logs to parse.
}

func readGoldenTestCase(t *testing.T) []goldenTestCase {
	t.Helper()

	f, err := os.Open("testdata/aws-vpc-flow-logs.yml")
	if err != nil {
		t.Fatal(err)
	}

	dec := yaml.NewDecoder(f)

	var testCases map[string]goldenTestCase
	if err = dec.Decode(&testCases); err != nil {
		t.Fatal(err)
	}

	testCasesList := make([]goldenTestCase, 0, len(testCases))
	for k, v := range testCases {
		v.Name = k
		testCasesList = append(testCasesList, v)
	}

	return testCasesList
}

func writeGolden(t *testing.T, path string, events []mapstr.M) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	if err = enc.Encode(events); err != nil {
		t.Fatal()
	}
}

func readGolden(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func BenchmarkProcessorRun(b *testing.B) {
	benchmarks := []struct {
		name    string
		mode    mode
		format  string
		message string
	}{
		{"original-mode-v5-message", originalMode, formatV5, formatV5Sample},
		{"ecs-mode-v5-message", ecsMode, formatV5, formatV5Sample},
		{"ecs_and_original-mode-v5-message", ecsAndOriginalMode, formatV5, formatV5Sample},
	}

	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			c := defaultConfig()
			c.Format = []string{benchmark.format}
			c.Mode = benchmark.mode

			p, err := newParseAWSVPCFlowLog(c)
			require.NoError(b, err)

			evt := beat.Event{
				Timestamp: time.Now().UTC(),
				Fields: map[string]interface{}{
					"message": benchmark.message,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err = p.Run(&evt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
