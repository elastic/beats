// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestFilestreamRegistryIsInDiagnostics(t *testing.T) {
	filebeat := NewFilebeat(t)
	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.GenerateLogFile(t, logfile, 2, false)
	input := proto.UnitExpected{
		Id:             "input-" + t.Name(),
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "unit-filestream-" + t.Name(),
			Type: "filestream",
			Name: "Filestream-" + t.Name(),
			Streams: []*proto.Stream{
				{
					Id: "stream-filestream-" + t.Name(),
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":            "stream-filestream-" + t.Name(),
						"enabled":       true,
						"type":          "filestream",
						"paths":         []interface{}{logfile},
						"file.identity": map[string]any{},
						"prospector.scanner.fingerprint": map[string]any{
							"enabled": false,
						},
					}),
				},
			},
		},
	}

	output := proto.UnitExpected{
		Id:             "unit-output-" + t.Name(),
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "output-" + t.Name(),
			Type: "file",
			Name: "file",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":     "file",
					"path":     filebeat.TempDir(),
					"filename": "output",
				}),
		},
	}
	outputGlob := filepath.Join(filebeat.TempDir(), "output*")

	var units = []*proto.UnitExpected{
		&output,
		&input,
	}

	waitingForDiagnostics := atomic.Bool{}
	testDone := make(chan struct{})

	server := &mock.StubServerV2{
		ActionImpl:  func(response *proto.ActionResponse) error { return nil },
		ActionsChan: make(chan *mock.PerformAction),
		SentActions: map[string]*mock.PerformAction{},
	}

	server.CheckinV2Impl = func(observed *proto.CheckinObserved) *proto.CheckinExpected {
		// No matter the current state, we always return the same units
		checkinExpected := proto.CheckinExpected{
			Units: units,
		}

		// If any unit is not healthy, just return the expected state
		for _, unit := range observed.Units {
			if unit.GetState() != proto.State_HEALTHY {
				return &checkinExpected
			}
		}

		// All units are healthy, we can request the diagnostics.
		// Ensure we don't have any diagnostics being requested already.
		if waitingForDiagnostics.CompareAndSwap(false, true) {
			// Request the diagnostics asynchronously
			go requestDiagnosticsAndVerifyRegistry(
				t,
				filebeat,
				outputGlob,
				logfile,
				100,
				&waitingForDiagnostics,
				server,
				testDone,
				false)
		}
		return &checkinExpected
	}

	if err := server.Start(); err != nil {
		t.Fatalf("cannot start gRPC server: %s", err)
	}

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "queue.mem.flush.timeout=0",
	)

	<-testDone
}

func TestEmptyegistryIsInDiagnostics(t *testing.T) {
	filebeat := NewFilebeat(t)
	input := proto.UnitExpected{
		Id:             "input-" + t.Name(),
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "unit-filestream-" + t.Name(),
			Type: "benchmark",
			Name: "Benchmark-" + t.Name(),
			Streams: []*proto.Stream{
				{
					Id: "stream-benchmark-" + t.Name(),
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":      "stream-benchmark-" + t.Name(),
						"enabled": true,
						"count":   2,
					}),
				},
			},
		},
	}

	output := proto.UnitExpected{
		Id:             "unit-output-" + t.Name(),
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "output-" + t.Name(),
			Type: "file",
			Name: "file",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":     "file",
					"path":     filebeat.TempDir(),
					"filename": "output",
				}),
		},
	}
	outputGlob := filepath.Join(filebeat.TempDir(), "output*")

	var units = []*proto.UnitExpected{
		&output,
		&input,
	}

	waitingForDiagnostics := atomic.Bool{}
	testDone := make(chan struct{})

	server := &mock.StubServerV2{
		ActionImpl:  func(response *proto.ActionResponse) error { return nil },
		ActionsChan: make(chan *mock.PerformAction),
		SentActions: map[string]*mock.PerformAction{},
	}

	server.CheckinV2Impl = func(observed *proto.CheckinObserved) *proto.CheckinExpected {
		// No matter the current state, we always return the same units
		checkinExpected := proto.CheckinExpected{
			Units: units,
		}

		// If any unit is not healthy, just return the expected state
		for _, unit := range observed.Units {
			if unit.GetState() != proto.State_HEALTHY {
				return &checkinExpected
			}
		}

		// All units are healthy, we can request the diagnostics.
		// Ensure we don't have any diagnostics being requested already.
		if waitingForDiagnostics.CompareAndSwap(false, true) {
			// Request the diagnostics asynchronously
			go requestDiagnosticsAndVerifyRegistry(t, filebeat, outputGlob, "", 0, &waitingForDiagnostics, server, testDone, true)
		}
		return &checkinExpected
	}

	if err := server.Start(); err != nil {
		t.Fatalf("cannot start gRPC server: %s", err)
	}

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "queue.mem.flush.timeout=0",
	)

	<-testDone
}

func validateLastRegistryEntry(t *testing.T, reader io.Reader, expectedSize int, expectedPath string) {
	t.Helper()

	sc := bufio.NewScanner(reader)

	var lastLine []byte
	for sc.Scan() {
		lastLine = sc.Bytes()
	}

	entry := struct {
		Data struct {
			Meta struct {
				Path string `json:"source"`
			} `json:"meta"`
			Cursor struct {
				Offset int `json:"offset"`
			} `json:"cursor"`
		} `json:"v"`
	}{}

	if err := json.Unmarshal(lastLine, &entry); err != nil {
		t.Errorf("cannot unmarshal last registry entry: %s", err)
	}

	if entry.Data.Meta.Path != expectedPath {
		t.Errorf(
			"expecting path in registry to be '%s', got '%s' instead",
			expectedPath,
			entry.Data.Meta.Path)
	}

	if entry.Data.Cursor.Offset != expectedSize {
		t.Errorf(
			"expecting offset to be %d, got %d instead",
			expectedSize,
			entry.Data.Cursor.Offset)
	}
}

// requestDiagnosticsAndVerifyRegistry runs on a different goroutine, we cannot call t.Fatal
func requestDiagnosticsAndVerifyRegistry(
	t *testing.T,
	filebeat *integration.BeatProc,
	outputGlob,
	logfile string,
	logfileOffset int,
	waitingForDiagnostics *atomic.Bool,
	server *mock.StubServerV2,
	testDone chan<- struct{},
	emtpyRegistyLogFile bool) {

	assert.Eventuallyf(
		t,
		func() bool {
			return filebeat.CountFileLines(outputGlob) == 2
		},
		1*time.Minute,
		100*time.Millisecond,
		"output file '%s' does not contain two events", outputGlob)

	// Once we're done, set it back to false
	defer waitingForDiagnostics.Store(false)
	server.ActionsChan <- &mock.PerformAction{
		Type:  proto.ActionRequest_DIAGNOSTICS,
		Name:  "diagnostics",
		Level: proto.ActionRequest_COMPONENT, // aka diagnostics for the whole Beat
		DiagCallback: func(diagResults []*proto.ActionDiagnosticUnitResult, diagErr error) {
			// Let the test finish when this callback finishes
			defer func() {
				testDone <- struct{}{}
			}()

			if diagErr != nil {
				t.Errorf("diagnostics failed: %s", diagErr)
				return
			}

			for _, dr := range diagResults {
				if dr.Name != "registry" {
					continue
				}

				if len(dr.Content) == 0 {
					t.Errorf("registry cannot be an empty file")
					return
				}

				gzipReader, err := gzip.NewReader(bytes.NewReader(dr.Content))
				if err != nil {
					t.Errorf("cannot create gzip reader: '%s'", err)
					return
				}
				defer gzipReader.Close()

				tarReader := tar.NewReader(gzipReader)
				for {
					header, err := tarReader.Next()
					if errors.Is(err, io.EOF) {
						t.Error("registry log file not found in tar archive")
						return
					}

					if header.Name != "registry/filebeat/log.json" {
						continue
					}

					if emtpyRegistyLogFile {
						if header.Size != 0 {
							t.Errorf("expecting registry log file to be empty, got %d bytes instead", header.Size)
						}
						return
					}

					validateLastRegistryEntry(t, tarReader, logfileOffset, logfile)
					return
				}
			}
			t.Error("diagnostics do not contain a valid registry")
		},
	}
}
