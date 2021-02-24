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

package filestream

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
)

// test_close_renamed from test_harvester.py
func TestFilestreamCloseRenamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	tmpDir := t.TempDir()

	testlogPath := filepath.Join(tmpDir, "test.log")
	err := ioutil.WriteFile(testlogPath, []byte("first log line\n"), 0644)
	if err != nil {
		t.Fatalf("cannot write log lines to test file: %+v", err)
	}

	testStore := openTestStatestore()
	inp := getInputFromConfig(
		t,
		testStore,
		map[string]interface{}{
			"paths":                                []string{testlogPath + "*"},
			"prospector.scanner.check_interval":    "1ms",
			"close.on_state_change.check_interval": "1ms",
			"close.on_state_change.renamed":        "true",
		},
	)

	ctx, cancelInput := context.WithCancel(context.Background())
	inputCtx := input.Context{Logger: logp.L(), Cancelation: ctx}
	pipeline := &mockPipelineConnector{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		inp.Run(inputCtx, pipeline)
	}()

	// first event has made it successfully
	waitUntilEventCount(1, pipeline)
	// check registry
	checkOffsetInRegistry(t, testStore, testlogPath, len([]byte("first log line\n")))

	testlogPathRenamed := filepath.Join(tmpDir, "test.log.rotated")
	err = os.Rename(testlogPath, testlogPathRenamed)
	if err != nil {
		t.Fatalf("failed to rename file '%s': %+v", testlogPath, err)
	}

	err = ioutil.WriteFile(testlogPath, []byte("new first log line\nnew second log line\n"), 0644)
	if err != nil {
		t.Fatalf("cannot write log lines to test file: %+v", err)
	}

	// new two events are read
	waitUntilEventCount(2, pipeline)

	cancelInput()
	wg.Wait()

	checkOffsetInRegistry(t, testStore, testlogPathRenamed, len([]byte("first log line\n")))
	checkOffsetInRegistry(t, testStore, testlogPath, len([]byte("new first log line\nnew second log line\n")))
}

// test_close_eof from test_harvester.py
func TestFilestreamCloseEOF(t *testing.T) {
	tmpDir := t.TempDir()

	testlogPath := filepath.Join(tmpDir, "test.log")
	err := ioutil.WriteFile(testlogPath, []byte("first log line\n"), 0644)
	if err != nil {
		t.Fatalf("cannot write log lines to test file: %+v", err)
	}

	testStore := openTestStatestore()
	inp := getInputFromConfig(
		t,
		testStore,
		map[string]interface{}{
			"paths":                             []string{testlogPath},
			"prospector.scanner.check_interval": "24h",
			"close.reader.on_eof":               "true",
		},
	)

	ctx, cancelInput := context.WithCancel(context.Background())
	inputCtx := input.Context{Logger: logp.L(), Cancelation: ctx}
	pipeline := &mockPipelineConnector{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		inp.Run(inputCtx, pipeline)
	}()

	// first event has made it successfully
	waitUntilEventCount(1, pipeline)
	// check registry
	checkOffsetInRegistry(t, testStore, testlogPath, len([]byte("first log line\n")))

	// the second log line will not be picked up as scan_interval is set to one day.
	err = ioutil.WriteFile(testlogPath, []byte("first log line\nsecond log line\n"), 0644)
	if err != nil {
		t.Fatalf("cannot write log lines to test file: %+v", err)
	}

	// only one event is read
	waitUntilEventCount(1, pipeline)

	cancelInput()
	wg.Wait()

	checkOffsetInRegistry(t, testStore, testlogPath, len([]byte("first log line\n")))
}

func checkOffsetInRegistry(t *testing.T, store loginp.StateStore, filepath string, expectedOffset int) {
	fi, err := os.Stat(filepath)
	if err != nil {
		t.Fatalf("cannot stat file when cheking for offset: %+v", err)
	}

	identifier, _ := newINodeDeviceIdentifier(nil)
	src := identifier.GetSource(loginp.FSEvent{Info: fi, Op: loginp.OpCreate, NewPath: filepath})
	entry := getRegistryState(t, store, src.Name())

	require.Equal(t, expectedOffset, entry.Cursor.Offset)
}

func getInputFromConfig(t *testing.T, s loginp.StateStore, c map[string]interface{}) input.Input {
	manager := &loginp.InputManager{
		Logger:     logp.L(),
		StateStore: s,
		Type:       pluginName,
		Configure:  configure,
	}

	inp, err := manager.Create(common.MustNewConfigFrom(c))
	if err != nil {
		t.Fatalf("cannot create filestream input: %+v", err)
	}
	return inp
}

type registryEntry struct {
	Cursor struct {
		Offset int `json:"offset"`
	} `json:"cursor"`
}

func getRegistryState(t *testing.T, s loginp.StateStore, key string) registryEntry {
	inputStore, _ := s.Access()

	var e registryEntry
	err := inputStore.Get(key, &e)
	if err != nil {
		t.Fatalf("error when getting expected key '%s' from store: %+v", key, err)
	}

	return e
}

func waitUntilEventCount(count int, pipeline *mockPipelineConnector) {
	for {
		for _, c := range pipeline.clients {
			if len(c.GetEvents()) == count {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() loginp.StateStore {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	s.registry.Close()
}

func (s *testInputStore) Access() (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}

type mockClient struct {
	publishes  []beat.Event
	ackHandler beat.ACKer
	closed     bool
	mtx        sync.Mutex
}

// GetEvents returns the published events
func (c *mockClient) GetEvents() []beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.publishes
}

// Publish mocks the Client Publish method
func (c *mockClient) Publish(e beat.Event) {
	c.PublishAll([]beat.Event{e})
}

// PublishAll mocks the Client PublishAll method
func (c *mockClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, event := range events {
		c.publishes = append(c.publishes, event)
		c.ackHandler.AddEvent(event, true)
	}
	c.ackHandler.ACKEvents(len(events))
}

// Close mocks the Client Close method
func (c *mockClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

// mockPipelineConnector mocks the PipelineConnector interface
type mockPipelineConnector struct {
	clients []*mockClient
	mtx     sync.Mutex
}

// GetAllEvents returns all events associated with a pipeline
func (pc *mockPipelineConnector) GetAllEvents() []beat.Event {
	var evList []beat.Event
	for _, clientEvents := range pc.clients {
		evList = append(evList, clientEvents.GetEvents()...)
	}

	return evList
}

// Connect mocks the PipelineConnector Connect method
func (pc *mockPipelineConnector) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

// ConnectWith mocks the PipelineConnector ConnectWith method
func (pc *mockPipelineConnector) ConnectWith(config beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &mockClient{
		ackHandler: config.ACKHandler,
	}

	pc.clients = append(pc.clients, c)

	return c, nil
}
