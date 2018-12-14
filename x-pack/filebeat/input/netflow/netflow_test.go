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

	"github.com/stretchr/testify/assert"
	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/pcap"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/test"
)

var (
	update = flag.Bool("update", false, "update golden data")

	sanitizer = strings.NewReplacer("-", "--", ":", "-", "/", "-", "+", "-", " ", "-", ",", "")
)

const (
	pcapDir     = "testdata/pcap"
	datDir      = "testdata/dat"
	goldenDir   = "testdata/golden"
	datSourceIP = "192.0.2.1"
)

// DatTests specifies the .dat files associated with test cases.
type DatTests struct {
	Tests map[string][]string `yaml:"tests"`
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

	for name, files := range tests.Tests {
		t.Run(name, func(t *testing.T) {
			goldenName := filepath.Join(goldenDir, sanitizer.Replace(name)+".golden.json")
			result := getFlowsFromDat(t, name, files...)

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

func getFlowsFromDat(t testing.TB, name string, datFiles ...string) TestResult {
	t.Helper()

	config := decoder.NewConfig().
		WithProtocols(protocol.Registry.All()...).
		WithSequenceResetEnabled(false).
		WithExpiration(0).
		WithLogOutput(test.TestLogWriter{t})

	decoder, err := decoder.NewDecoder(config)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}

	source := test.MakeAddress(t, datSourceIP+":4444")
	var events []beat.Event
	for _, f := range datFiles {
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
				ev[i] = toBeatEvent(flows[i])
			}
			//return TestResult{Name: name, Error: err.Error(), Events: flowsToEvents(flows)}
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
		WithLogOutput(test.TestLogWriter{t})

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
			ev[i] = toBeatEvent(flows[i])
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
