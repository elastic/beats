// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
)

var (
	update = flag.Bool("update", false, "update golden data")

	sanitizer = strings.NewReplacer("-", "--", ":", "-", "/", "-", "+", "-", " ", "-", ",", "")
)

const (
	pcapDir     = "testdata/pcap"
	datDir      = "testdata/dat"
	goldenDir   = "testdata/golden"
	fieldsDir   = "testdata/fields"
	datSourceIP = "192.0.2.1"
)

// DatTests specifies the .dat files associated with test cases.
type DatTests struct {
	Tests map[string]TestCase `yaml:"tests"`
}

type TestCase struct {
	Files  []string `yaml:"files"`
	Fields []string `yaml:"custom_fields"`
}

// TestResult specifies the format of the result data that is written in a
// golden files.
type TestResult struct {
	Name  string       `json:"test_name"`
	Error string       `json:"error,omitempty"`
	Flows []beat.Event `json:"events,omitempty"`
}

func TestPCAPFiles(t *testing.T) {
	pcaps, err := filepath.Glob(filepath.Join(pcapDir, "*.pcap"))
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range pcaps {
		testName := strings.TrimSuffix(filepath.Base(file), ".pcap")

		t.Run(testName, func(t *testing.T) {
			goldenName := filepath.Join(goldenDir, testName+".pcap.golden.json")
			result := getFlowsFromPCAP(t, testName, file)

			if *update {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					t.Fatal(err)
				}

				if err = os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatal(err)
				}

				err = ioutil.WriteFile(goldenName, data, 0644)
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			goldenData := readGoldenFile(t, goldenName)
			assert.EqualValues(t, goldenData, normalize(t, result))
		})
	}
}

func TestDatFiles(t *testing.T) {
	tests := readDatTests(t)

	for name, testData := range tests.Tests {
		t.Run(name, func(t *testing.T) {
			goldenName := filepath.Join(goldenDir, sanitizer.Replace(name)+".golden.json")
			result := getFlowsFromDat(t, name, testData)

			if *update {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					t.Fatal(err)
				}

				if err = os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatal(err)
				}

				err = ioutil.WriteFile(goldenName, data, 0644)
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			goldenData := readGoldenFile(t, goldenName)
			jsonGolden, err := json.Marshal(goldenData)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			t.Logf("Golden data: %+v", string(jsonGolden))
			jsonResult, err := json.Marshal(result)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			t.Logf("Result data: %+v", string(jsonResult))
			assert.EqualValues(t, goldenData, normalize(t, result))
			assert.Equal(t, jsonGolden, jsonResult)
		})
	}
}

func readDatTests(t testing.TB) *DatTests {
	data, err := ioutil.ReadFile("testdata/dat_tests.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var tests DatTests
	if err := yaml.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}

	return &tests
}

func getFlowsFromDat(t testing.TB, name string, testCase TestCase) TestResult {
	t.Helper()

	config := decoder.NewConfig().
		WithProtocols(protocol.Registry.All()...).
		WithSequenceResetEnabled(false).
		WithExpiration(0).
		WithLogOutput(test.TestLogWriter{TB: t})

	for _, fieldFile := range testCase.Fields {
		fields, err := LoadFieldDefinitionsFromFile(filepath.Join(fieldsDir, fieldFile))
		if err != nil {
			t.Fatal(err, fieldFile)
		}
		config = config.WithCustomFields(fields)
	}

	decoder, err := decoder.NewDecoder(config)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}

	source := test.MakeAddress(t, datSourceIP+":4444")
	var events []beat.Event
	for _, f := range testCase.Files {
		dat, err := ioutil.ReadFile(filepath.Join(datDir, f))
		if err != nil {
			t.Fatal(err)
		}
		data := bytes.NewBuffer(dat)
		var packetCount int
		for packetCount = 0; data.Len() > 0; packetCount++ {
			startLen := data.Len()
			flows, err := decoder.Read(data, source)
			if err != nil {
				t.Logf("test %v: decode error: %v", name, err)
				break
			}
			if data.Len() == startLen {
				t.Log("Loop detected")
			}
			ev := make([]beat.Event, len(flows))
			for i := range flows {
				flow := toBeatEvent(flows[i], []string{"private"})
				flow.Fields.Delete("event.created")
				ev[i] = flow
			}
			// return TestResult{Name: name, Error: err.Error(), Events: flowsToEvents(flows)}
			events = append(events, ev...)
		}
	}

	return TestResult{Name: name, Flows: events}
}

func getFlowsFromPCAP(t testing.TB, name, pcapFile string) TestResult {
	t.Helper()

	r, err := pcap.OpenOffline(pcapFile)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	config := decoder.NewConfig().
		WithProtocols(protocol.Registry.All()...).
		WithSequenceResetEnabled(false).
		WithExpiration(0).
		WithLogOutput(test.TestLogWriter{TB: t})

	decoder, err := decoder.NewDecoder(config)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	packetSource := gopacket.NewPacketSource(r, r.LinkType())
	var events []beat.Event

	// Process packets in PCAP and get flow records.
	for packet := range packetSource.Packets() {
		remoteAddr := &net.UDPAddr{
			IP:   net.ParseIP(packet.NetworkLayer().NetworkFlow().Src().String()),
			Port: int(binary.BigEndian.Uint16(packet.TransportLayer().TransportFlow().Src().Raw())),
		}
		payloadData := packet.TransportLayer().LayerPayload()
		flows, err := decoder.Read(bytes.NewBuffer(payloadData), remoteAddr)
		if err != nil {
			return TestResult{Name: name, Error: err.Error(), Flows: events}
		}
		ev := make([]beat.Event, len(flows))
		for i := range flows {
			flow := toBeatEvent(flows[i], []string{"private"})
			flow.Fields.Delete("event.created")
			ev[i] = flow
		}
		events = append(events, ev...)
	}

	return TestResult{Name: name, Flows: events}
}

func normalize(t testing.TB, result TestResult) TestResult {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var tr TestResult
	if err = json.Unmarshal(data, &tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

func readGoldenFile(t testing.TB, file string) TestResult {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	var tr TestResult
	if err = json.Unmarshal(data, &tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

// This test converts a flow and its reverse flow to a Beat event
// to check that they have the same flow.id, locality and community-id.
func TestReverseFlows(t *testing.T) {
	parseMAC := func(s string) net.HardwareAddr {
		addr, err := net.ParseMAC(s)
		if err != nil {
			t.Fatal(err)
		}
		return addr
	}
	flows := []record.Record{
		{
			Type: record.Flow,
			Fields: record.Map{
				"ingressInterface":         uint64(2),
				"destinationTransportPort": uint64(50285),
				"sourceTransportPort":      uint64(993),
				"packetDeltaCount":         uint64(26),
				"ipVersion":                uint64(4),
				"sourceIPv4Address":        net.ParseIP("203.0.113.123").To4(),
				"deltaFlowCount":           uint64(0),
				"sourceMacAddress":         parseMAC("10:00:00:00:00:02"),
				"flowDirection":            uint64(0),
				"flowEndSysUpTime":         uint64(64526131),
				"vlanId":                   uint64(0),
				"ipClassOfService":         uint64(0),
				"mplsLabelStackLength":     uint64(3),
				"tcpControlBits":           uint64(27),
				"egressInterface":          uint64(3),
				"destinationIPv4Address":   net.ParseIP("10.111.111.96").To4(),
				"protocolIdentifier":       uint64(6),
				"flowStartSysUpTime":       uint64(64523806),
				"destinationMacAddress":    parseMAC("10:00:00:00:00:03"),
				"octetDeltaCount":          uint64(12852),
			},
		},
		{
			Type: record.Flow,
			Fields: record.Map{
				"ingressInterface":          uint64(3),
				"destinationTransportPort":  uint64(993),
				"sourceTransportPort":       uint64(50285),
				"packetDeltaCount":          uint64(26),
				"ipVersion":                 uint64(4),
				"destinationIPv4Address":    net.ParseIP("203.0.113.123").To4(),
				"deltaFlowCount":            uint64(0),
				"postDestinationMacAddress": parseMAC("10:00:00:00:00:03"),
				"flowDirection":             uint64(1),
				"flowEndSysUpTime":          uint64(64526131),
				"vlanId":                    uint64(0),
				"ipClassOfService":          uint64(0),
				"mplsLabelStackLength":      uint64(3),
				"tcpControlBits":            uint64(27),
				"egressInterface":           uint64(3),
				"sourceIPv4Address":         net.ParseIP("10.111.111.96").To4(),
				"protocolIdentifier":        uint64(6),
				"flowStartSysUpTime":        uint64(64523806),
				"postSourceMacAddress":      parseMAC("10:00:00:00:00:02"),
				"octetDeltaCount":           uint64(12852),
			},
		},
	}

	var evs []beat.Event
	for _, f := range flows {
		evs = append(evs, toBeatEvent(f, []string{"private"}))
	}
	if !assert.Len(t, evs, 2) {
		t.Fatal()
	}
	for _, key := range []string{"flow.id", "flow.locality", "network.community_id"} {
		var keys [2]interface{}
		for i := range keys {
			var err error
			if keys[i], err = evs[i].Fields.GetValue(key); err != nil {
				t.Fatal(err, "event num=", i, "key=", key)
			}
		}
		assert.Equal(t, keys[0], keys[1], key)
	}
}
