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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
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
				"version":                   "1",
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
				// ECS
				"event.code":       "600",
				"message":          "User signed in from 2001:db8::5",
				"observer.product": "Deep Security Manager",
				"observer.vendor":  "Trend Micro",
				"observer.version": "1.2.3",
				"source.ip":        "10.52.116.160",
				"source.user.name": "admin",
			},
		},
		"parse_errors": {
			message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|Low|msg=User signed in with =xyz",
			fields: common.MapStr{
				"cef.version":               "0",
				"cef.device.event_class_id": "600",
				"cef.device.product":        "Deep Security Manager",
				"cef.device.vendor":         "Trend Micro",
				"cef.device.version":        "1.2.3",
				"cef.name":                  "User Signed In",
				"cef.severity":              "Low",
				// ECS
				"event.code":       "600",
				"event.severity":   0,
				"observer.product": "Deep Security Manager",
				"observer.vendor":  "Trend Micro",
				"observer.version": "1.2.3",
				"message":          "User Signed In",
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
				"cef.version":                   "0",
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
		"truncated_header": {
			config: func() config {
				c := defaultConfig()
				c.ECS = false
				return c
			},
			message: "CEF:0|SentinelOne|Mgmt|activityID=1111111111111111111 activityType=3505 siteId=None siteName=None accountId=1222222222222222222 accountName=foo-bar mdr notificationScope=ACCOUNT",
			fields: common.MapStr{
				"cef.version":                      "0",
				"cef.device.product":               "Mgmt",
				"cef.device.vendor":                "SentinelOne",
				"cef.extensions.accountId":         "1222222222222222222",
				"cef.extensions.accountName":       "foo-bar mdr",
				"cef.extensions.activityID":        "1111111111111111111",
				"cef.extensions.activityType":      "3505",
				"cef.extensions.notificationScope": "ACCOUNT",
				"cef.extensions.siteId":            "None",
				"cef.extensions.siteName":          "None",
				"message":                          "CEF:0|SentinelOne|Mgmt|activityID=1111111111111111111 activityType=3505 siteId=None siteName=None accountId=1222222222222222222 accountName=foo-bar mdr notificationScope=ACCOUNT",
				"error.message": []string{
					"unexpected end of CEF event",
					"incomplete CEF header",
				},
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

		_, err = dec.Run(evt)
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
		assert.EqualValues(t, "1", version)
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
	conf.Field = "event.original"
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
				"event": common.MapStr{"original": string(data)},
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
func assertEqual(t testing.TB, expected, actual interface{}) bool { //nolint:unparam // Bad linter!
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

func BenchmarkProcessorRun(b *testing.B) {
	dec, err := newDecodeCEF(defaultConfig())
	if err != nil {
		b.Fatal(err)
	}

	const msg = `CEF:1|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5`
	b.Run("short_msg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := dec.Run(&beat.Event{
				Fields: map[string]interface{}{
					"message": msg,
				},
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	const longMsg = `CEF:0|CISCO|ASA||305012|Teardown dynamic UDP translation|Low| eventId=56265798504 mrt=1484092683471 proto=UDP categorySignificance=/Informational categoryBehavior=/Access/Stop categoryDeviceGroup=/Firewall catdt=Firewall categoryOutcome=/Success categoryObject=/Host/Application/Service modelConfidence=0 severity=4 relevance=10 assetCriticality=0 priority=4 art=1484096108163 deviceSeverity=6 rt=1484096094000 src=1.2.3.4 sourceZoneID=GqtK3G9YBABCadQ465CqVeW\=\= sourceZoneURI=/All Zones/GTR/GTR/GTR/GTR sourceTranslatedAddress=4.3.2.1 sourceTranslatedZoneID=P84KXXTYDFYYFwwHq40BQcd\=\= sourceTranslatedZoneURI=/All Zones/GTR/GTR Internet Primary spt=5260 sourceTranslatedPort=5260 cs5=dynamic cs6=0:00:00 c6a4=ffff:0:0:0:222:5555:ffff:5555 locality=1 cs1Label=ACL cs2Label=Unit cs3Label=TCP Flags cs4Label=Order cs5Label=Connection Type cs6Label=Duration cn1Label=ICMP Type cn2Label=ICMP Code cn3Label=DurationInSeconds c6a4Label=Agent IPv6 Address ahost=host.gtr.gtr agt=100.222.333.55 av=7.1.7.7602.0 atz=LA/la aid=4p9IZi1kBABCq5RFPFdJWYUw\=\= at=agent_ac dvchost=super dvc=111.111.111.99 deviceZoneID=K-fU33AAOGVdfFpYAT3UdQ\=\= deviceZoneURI=/All Zones/ArcSight System/Private Address Space Zones/RFC1918: 192.168.0.0-192.168.255.255 deviceAssetId=5Wa8hHVSDFBCc-t56wI7mTw\=\= dtz=LA/LA deviceInboundInterface=eth0 deviceOutboundInterface=eth1 eventAnnotationStageUpdateTime=1484097686473 eventAnnotationModificationTime=1484097686475 eventAnnotationAuditTrail=1,1484012146095,root,Queued,,,,\\n eventAnnotationVersion=1 eventAnnotationFlags=0 eventAnnotationEndTime=1484096094000 eventAnnotationManagerReceiptTime=1484097686471 originalAgentHostName=host originalAgentAddress=10.2.88.3 originalAgentZoneURI=/All Zones/GR/GR/GR originalAgentVersion=7.3.0.7885.0 originalAgentId=6q0sfHVcBABCcSDFvMpvc1w\=\= originalAgentType=syslog_file _cefVer=0.1 ad.arcSightEventPath=7q0sfHVcBABCcMZVvMSDFc1w\=\=`
	b.Run("long_msg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := dec.Run(&beat.Event{
				Fields: map[string]interface{}{
					"message": longMsg,
				},
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
