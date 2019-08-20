// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"bufio"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

var updateGolden = flag.Bool("update", false, "update golden test files")

func TestProcessorRun(t *testing.T) {
	type testCase struct {
		config  func() config
		message string
		fields  common.MapStr
	}

	var testCases = map[string]testCase{
		"custom_target_root": {
			config: func() config {
				c := defaultConfig()
				c.TargetField = ""
				return c
			},
			message: "CEF:1|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
			fields: common.MapStr{
				"version":                   1,
				"device.event_class_id":     "600",
				"device.product":            "Deep Security Manager",
				"device.vendor":             "Trend Micro",
				"device.version":            "1.2.3",
				"name":                      "User Signed In",
				"severity":                  "3",
				"event.severity":            3,
				"extensions.message":        "User signed in from 2001:db8::5",
				"extensions.sourceAddress":  "10.52.116.160",
				"extensions.sourceUserName": "admin",
				"extensions.target":         "admin",
				"message":                   "User signed in from 2001:db8::5",
				"source.ip":                 "10.52.116.160",
				"source.user.name":          "admin",
			},
		},
		"parse_errors": {
			message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|Low|msg=User signed in with =xyz",
			fields: common.MapStr{
				"cef.version":               0,
				"cef.device.event_class_id": "600",
				"cef.device.product":        "Deep Security Manager",
				"cef.device.vendor":         "Trend Micro",
				"cef.device.version":        "1.2.3",
				"cef.name":                  "User Signed In",
				"cef.severity":              "Low",
				"event.severity":            0,
				"message":                   "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|Low|msg=User signed in with =xyz",
				"error.message": []string{
					"malformed value for msg at pos 94",
					"unexpected end of CEF event",
				},
			},
		},
		"ecs_disabled": {
			config: func() config {
				c := defaultConfig()
				c.ECS = false
				return c
			},
			message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
			fields: common.MapStr{
				"cef.version":                   0,
				"cef.device.event_class_id":     "600",
				"cef.device.product":            "Deep Security Manager",
				"cef.device.vendor":             "Trend Micro",
				"cef.device.version":            "1.2.3",
				"cef.name":                      "User Signed In",
				"cef.severity":                  "3",
				"cef.extensions.message":        "User signed in from 2001:db8::5",
				"cef.extensions.sourceAddress":  "10.52.116.160",
				"cef.extensions.sourceUserName": "admin",
				"cef.extensions.target":         "admin",
				"message":                       "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
			},
		},
	}

	dec, err := newDecodeCEF(defaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dec := dec
			if tc.config != nil {
				dec, err = newDecodeCEF(tc.config())
				if err != nil {
					t.Fatal(err)
				}
			}

			evt := &beat.Event{
				Fields: common.MapStr{
					"message": tc.message,
				},
			}

			evt, err = dec.Run(evt)
			if err != nil {
				t.Fatal(err)
			}

			assertEqual(t, tc.fields, evt.Fields.Flatten())
		})
	}

	t.Run("not_cef", func(t *testing.T) {
		evt := &beat.Event{
			Fields: common.MapStr{
				"message": "hello world!",
			},
		}

		evt, err = dec.Run(evt)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "does not contain a CEF header")
		}
	})

	t.Run("leading_garbage", func(t *testing.T) {
		tc := testCases["custom_target_root"]

		evt := &beat.Event{
			Fields: common.MapStr{
				"message": "leading garbage" + tc.message,
			},
		}

		evt, err = dec.Run(evt)
		if err != nil {
			t.Fatal(err)
		}

		version, _ := evt.GetValue("cef.version")
		assert.EqualValues(t, 1, version)
	})
}

func TestGolden(t *testing.T) {
	const source = "testdata/samples.log"

	events := readCEFSamples(t, source)

	if *updateGolden {
		writeGoldenJSON(t, source, events)
		return
	}

	expected := readGoldenJSON(t, source)
	if !assert.Len(t, events, len(expected)) {
		return
	}
	for i, e := range events {
		assertEqual(t, expected[i], normalize(t, e))
	}
}

func readCEFSamples(t testing.TB, source string) []common.MapStr {
	f, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	conf := defaultConfig()
	conf.Field = "log.original"
	dec, err := newDecodeCEF(conf)
	if err != nil {
		t.Fatal(err)
	}

	var samples []common.MapStr
	s := bufio.NewScanner(f)
	for s.Scan() {
		data := s.Bytes()
		if len(data) == 0 || data[0] == '#' {
			continue
		}

		evt := &beat.Event{
			Fields: common.MapStr{
				"log": common.MapStr{"original": string(data)},
			},
		}

		evt, err := dec.Run(evt)
		if err != nil {
			t.Fatalf("Error reading from %v: %v", source, err)
		}

		samples = append(samples, evt.Fields)
	}
	if err = s.Err(); err != nil {
		t.Fatal(err)
	}

	return samples
}

func readGoldenJSON(t testing.TB, source string) []common.MapStr {
	source = source + ".golden.json"

	f, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	dec := json.NewDecoder(bufio.NewReader(f))

	var events []common.MapStr
	if err = dec.Decode(&events); err != nil {
		t.Fatal(err)
	}

	return events
}

func writeGoldenJSON(t testing.TB, source string, events []common.MapStr) {
	dest := source + ".golden.json"

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(events); err != nil {
		t.Fatal(err)
	}
}

func normalize(t testing.TB, m common.MapStr) common.MapStr {
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var out common.MapStr
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

// assertEqual asserts that the two objects are deeply equal. If not it will
// error the test and output a diff of the two objects' JSON representation.
func assertEqual(t testing.TB, expected, actual interface{}) bool {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		return true
	}

	expJSON, _ := json.MarshalIndent(expected, "", "  ")
	actJSON, _ := json.MarshalIndent(actual, "", "  ")

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expJSON)),
		B:        difflib.SplitLines(string(actJSON)),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  1,
	})
	t.Errorf("Expected and actual are different:\n%s", diff)
	return false
}
