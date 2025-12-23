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

//go:build integration

package filestream

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/go-concert/unison"
)

type inputTestingEnvironment struct {
	testLogger *logptest.Logger
	t          *testing.T
	workingDir string
	stateStore statestore.States
	pipeline   *mockPipelineConnector
	monitoring beat.Monitoring

	pluginInitOnce sync.Once
	plugin         v2.Plugin

	wg  sync.WaitGroup
	grp unison.TaskGroup
}

type registryEntry struct {
	Cursor struct {
		Offset int `json:"offset"`
	} `json:"cursor"`
	Meta any `json:"meta,omitempty"`
}

func newInputTestingEnvironment(t *testing.T) *inputTestingEnvironment {
	logger := logptest.NewFileLogger(
		t,
		filepath.Join("..", "..", "build", "integration-tests"),
	)

	return &inputTestingEnvironment{
		testLogger: logger,
		t:          t,
		workingDir: t.TempDir(),
		stateStore: openTestStatestore(),
		pipeline:   &mockPipelineConnector{},
		monitoring: beat.NewMonitoring(),
	}
}

func (e *inputTestingEnvironment) mustCreateInput(config map[string]any) v2.Input {
	e.t.Helper()
	e.grp = unison.TaskGroup{}
	manager := e.getManager()
	_ = manager.Init(&e.grp)
	c := conf.MustNewConfigFrom(config)
	inp, err := manager.Create(c)
	if err != nil {
		e.t.Fatalf("failed to create input using manager: %+v", err)
	}
	return inp
}

func (e *inputTestingEnvironment) createInput(config map[string]any) (v2.Input, error) {
	e.grp = unison.TaskGroup{}
	manager := e.getManager()
	_ = manager.Init(&e.grp)
	c := conf.MustNewConfigFrom(config)
	inp, err := manager.Create(c)
	if err != nil {
		return nil, err
	}

	return inp, nil
}

func (e *inputTestingEnvironment) getManager() v2.InputManager {
	e.pluginInitOnce.Do(func() {
		e.plugin = Plugin(e.testLogger.Logger, e.stateStore)
	})
	return e.plugin.Manager
}

func (e *inputTestingEnvironment) startInput(ctx context.Context, id string, inp v2.Input) {
	e.wg.Add(1)
	go func(wg *sync.WaitGroup, grp *unison.TaskGroup) {
		defer wg.Done()
		defer func() { _ = grp.Stop() }()

		logger := e.testLogger.Named("metrics-registry")
		reg := inputmon.NewMetricsRegistry(
			id, inp.Name(), e.monitoring.InputsRegistry(), logger)
		defer inputmon.CancelMetricsRegistry(
			id, inp.Name(), e.monitoring.InputsRegistry(), logger)

		inputCtx := v2.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     ctx,
			MetricsRegistry: reg,
			Logger:          e.testLogger.Named("input.filestream"),
		}
		_ = inp.Run(inputCtx, e.pipeline)
	}(&e.wg, &e.grp)
}

func (e *inputTestingEnvironment) waitUntilInputStops() {
	e.wg.Wait()
}

// mustWriteToFile writes data to file and returns the full path
func (e *inputTestingEnvironment) mustWriteToFile(filename string, data []byte) string {
	path := e.abspath(filename)
	err := os.WriteFile(path, data, 0o644)
	if err != nil {
		e.t.Fatalf("failed to write file '%s': %+v", path, err)
	}

	return path
}

func (e *inputTestingEnvironment) mustAppendToFile(filename string, data []byte) {
	path := e.abspath(filename)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		e.t.Fatalf("failed to open file '%s': %+v", path, err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		e.t.Fatalf("append data to file '%s': %+v", path, err)
	}
}

func (e *inputTestingEnvironment) mustRenameFile(oldname, newname string) {
	err := os.Rename(e.abspath(oldname), e.abspath(newname))
	if err != nil {
		e.t.Fatalf("failed to rename file '%s': %+v", oldname, err)
	}
}

func (e *inputTestingEnvironment) mustRemoveFile(filename string) {
	path := e.abspath(filename)
	err := os.Remove(path)
	if err != nil {
		e.t.Fatalf("failed to rename file '%s': %+v", path, err)
	}
}

func (e *inputTestingEnvironment) mustSymlink(filename, symlinkname string) {
	err := os.Symlink(e.abspath(filename), e.abspath(symlinkname))
	if err != nil {
		e.t.Fatalf("failed to create symlink to file '%s': %+v", filename, err)
	}
}

func (e *inputTestingEnvironment) mustTruncateFile(filename string, size int64) {
	path := e.abspath(filename)
	err := os.Truncate(path, size)
	if err != nil {
		e.t.Fatalf("failed to truncate file '%s': %+v", path, err)
	}
}

func (e *inputTestingEnvironment) abspath(filename string) string {
	return filepath.Join(e.workingDir, filename)
}

func (e *inputTestingEnvironment) requireRegistryEntryCount(expectedCount int) {
	inputStore, _ := e.stateStore.StoreFor("")

	actual := 0
	err := inputStore.Each(func(_ string, _ statestore.ValueDecoder) (bool, error) {
		actual += 1
		return true, nil
	})
	if err != nil {
		e.t.Fatalf("error while iterating through registry: %+v", err)
	}

	require.Equal(e.t, actual, expectedCount)
}

// requireOffsetInRegistry checks if the expected offset is set for a file.
func (e *inputTestingEnvironment) requireOffsetInRegistry(filename, inputID string, expectedOffset int) {
	e.t.Helper()
	require.EventuallyWithT(e.t, func(ct *assert.CollectT) {
		var offsetStr strings.Builder

		filepath := e.abspath(filename)
		fi, err := os.Stat(filepath)
		assert.NoError(ct, err, "cannot stat file when checking for offset")

		id := getIDFromPath(filepath, inputID, fi)
		var entry registryEntry
		offsetStr.Reset()

		entry, err = e.getRegistryState(id)
		assert.NoError(ct, err, "error getting state for ID '%s' from the registry", id)

		fmt.Fprint(&offsetStr, entry.Cursor.Offset)
		assert.Equal(ct, expectedOffset, entry.Cursor.Offset, "expected offset does not match")
	},
		10*time.Second,
		100*time.Millisecond,
		"failed to get expected registry offset")
}

// requireMetaInRegistry checks if the expected metadata is saved to the registry.
func (e *inputTestingEnvironment) waitUntilMetaInRegistry(filename, inputID string, expectedMeta fileMeta) {
	for {
		filepath := e.abspath(filename)
		fi, err := os.Stat(filepath)
		if err != nil {
			continue
		}

		id := getIDFromPath(filepath, inputID, fi)
		entry, err := e.getRegistryState(id)
		if err != nil {
			continue
		}

		if entry.Meta == nil {
			continue
		}

		var meta fileMeta
		err = typeconv.Convert(&meta, entry.Meta)
		if err != nil {
			e.t.Fatalf("cannot convert: %+v", err)
		}

		if requireMetadataEquals(expectedMeta, meta) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func requireMetadataEquals(one, other fileMeta) bool {
	return one == other
}

// waitUntilOffsetInRegistry waits for the expected offset is set for a file.
// If timeout is reached or there is an error getting the state from the
// registry, the test fails
func (e *inputTestingEnvironment) waitUntilOffsetInRegistry(
	filename, inputID string,
	expectedOffset int,
	timeout time.Duration) {

	var cursorString strings.Builder
	var fileSizeString strings.Builder

	filepath := e.abspath(filename)
	fi, err := os.Stat(filepath)
	if err != nil {
		e.t.Fatalf("cannot stat file when cheking for offset: %+v", err)
	}

	id := getIDFromPath(filepath, inputID, fi)

	require.Eventuallyf(e.t, func() bool {
		cursorString.Reset()
		fileSizeString.Reset()

		entry, err := e.getRegistryState(id)
		if err != nil {
			e.t.Fatalf(
				"error getting state for ID '%s' from the registry, err: %s",
				id, err)
		}

		fi, err := os.Stat(filepath)
		if err != nil {
			e.t.Fatalf("could not stat '%s', err: %s", filepath, err)
		}

		fileSizeString.WriteString(fmt.Sprint(fi.Size()))
		cursorString.WriteString(fmt.Sprint(entry.Cursor.Offset))

		return entry.Cursor.Offset == expectedOffset
	},
		timeout,
		100*time.Millisecond,
		"expected offset: '%d', cursor offset: '%s', file size: '%s'",
		expectedOffset,
		&cursorString,
		&fileSizeString)
}

func (e *inputTestingEnvironment) requireNoEntryInRegistry(filename, inputID string) {
	filepath := e.abspath(filename)
	fi, err := os.Stat(filepath)
	if err != nil {
		e.t.Fatalf("cannot stat file when cheking for offset: %+v", err)
	}

	inputStore, _ := e.stateStore.StoreFor("")
	id := getIDFromPath(filepath, inputID, fi)

	var entry registryEntry
	err = inputStore.Get(id, &entry)
	if err == nil {
		e.t.Fatalf("key is not expected to be present '%s'", id)
	}
}

// requireOffsetInRegistry checks if the expected offset is set for a file.
func (e *inputTestingEnvironment) requireOffsetInRegistryByID(key string, expectedOffset int) {
	entry, err := e.getRegistryState(key)
	if err != nil {
		e.t.Fatal(err.Error())
	}

	require.Equal(e.t, expectedOffset, entry.Cursor.Offset)
}

func (e *inputTestingEnvironment) getRegistryState(key string) (registryEntry, error) {
	inputStore, _ := e.stateStore.StoreFor("")

	var entry registryEntry
	err := inputStore.Get(key, &entry)
	if err != nil {
		var keys []string
		_ = inputStore.Each(func(key string, _ statestore.ValueDecoder) (bool, error) {
			keys = append(keys, key)
			return false, nil
		})
		e.t.Logf("keys in store: %v", keys)

		return registryEntry{},
			fmt.Errorf("error when getting expected key '%s' from store: %w",
				key, err)
	}

	return entry, nil
}

func getIDFromPath(filepath, inputID string, fi os.FileInfo) string {
	identifier, _ := newINodeDeviceIdentifier(nil, nil)
	src := identifier.GetSource(loginp.FSEvent{
		Descriptor: loginp.FileDescriptor{
			Info: file.ExtendFileInfo(fi),
		},
		Op:      loginp.OpCreate,
		NewPath: filepath,
	})
	return "filestream::" + inputID + "::" + src.Name()
}

// waitUntilEventCount waits until total count events arrive to the client.
func (e *inputTestingEnvironment) waitUntilEventCount(count int) {
	e.t.Helper()
	msg := &strings.Builder{}
	require.Eventuallyf(e.t, func() bool {
		msg.Reset()

		events := e.pipeline.GetAllEvents()
		sum := len(events)
		if sum == count {
			return true
		}
		fmt.Fprintf(msg, "unexpected number of events; expected: %d, actual: %d\n",
			count, sum)

		return false
	}, 2*time.Minute, 10*time.Millisecond, "%s", msg)
}

// waitUntilEventCountCtx calls waitUntilEventCount, but fails if ctx is cancelled.
func (e *inputTestingEnvironment) waitUntilEventCountCtx(ctx context.Context, count int) {
	e.t.Helper()
	ch := make(chan struct{})

	go func() {
		e.waitUntilEventCount(count)
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		logLines := map[string][]string{}
		for _, evt := range e.pipeline.GetAllEvents() {
			flat := evt.Fields.Flatten()
			pathi, _ := flat.GetValue("log.file.path")
			path, ok := pathi.(string)
			if !ok {
				e.t.Fatalf("waitUntilEventCountCtx: path is not a string: %v", pathi)
			}
			msgi, _ := flat.GetValue("message")
			msg, ok := msgi.(string)
			if !ok {
				e.t.Fatalf("waitUntilEventCountCtx: message is not a string: %v", msgi)
			}
			logLines[path] = append(logLines[path], msg)
		}

		e.t.Fatalf("waitUntilEventCountCtx: %v. Want %d events, got %d: %v",
			ctx.Err(),
			count,
			len(e.pipeline.GetAllEvents()),
			logLines)
	case <-ch:
		return
	}
}

// waitUntilAtLeastEventCount waits until at least count events arrive to the client.
func (e *inputTestingEnvironment) waitUntilAtLeastEventCount(count int) {
	for {
		sum := len(e.pipeline.GetAllEvents())
		if count <= sum {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitUntilHarvesterIsDone detects Harvester stop by checking if the last client has been closed
// as when a Harvester stops the client is closed.
func (e *inputTestingEnvironment) waitUntilHarvesterIsDone() {
	require.Eventually(
		e.t,
		func() bool {
			return e.pipeline.clients[len(e.pipeline.clients)-1].closed.Load()
		},
		time.Second*10,
		time.Millisecond*10,
		"The last connected client has not closed it's connection")
}

// requireEventsReceived requires that the list of messages has made it into the output.
func (e *inputTestingEnvironment) requireEventsReceived(events []string) {
	foundEvents := make([]bool, len(events))
	checkedEventCount := 0
	for _, c := range e.pipeline.clients {
		for _, evt := range c.GetEvents() {
			if len(events) == checkedEventCount {
				e.t.Fatalf("not enough expected elements")
			}
			message, ok := evt.Fields["message"].(string)
			if !ok {
				e.t.Fatalf("message is not string %+v", evt.Fields["message"])
			}
			if message == events[checkedEventCount] {
				foundEvents[checkedEventCount] = true
			}
			checkedEventCount += 1
		}
	}

	var missingEvents []string
	for i, found := range foundEvents {
		if !found {
			missingEvents = append(missingEvents, events[i])
		}
	}

	require.Equal(e.t, 0, len(missingEvents),
		"following events are missing: %+v", missingEvents)
}

func (e *inputTestingEnvironment) getOutputMessages() []string {
	messages := make([]string, 0)
	for _, c := range e.pipeline.clients {
		for _, evt := range c.GetEvents() {
			//nolint:errcheck // It's a test, we can force the type cast
			messages = append(messages, evt.Fields["message"].(string))
		}
	}
	return messages
}

func (e *inputTestingEnvironment) requireEventContents(nr int, key, value string) {
	events := make([]beat.Event, 0)
	for _, c := range e.pipeline.clients {
		events = append(events, c.GetEvents()...)
	}

	selectedEvent := events[nr]
	v, err := selectedEvent.Fields.GetValue(key)
	if err != nil {
		e.t.Fatalf("cannot find key %s in event %+v", key, selectedEvent)
	}

	val, ok := v.(string)
	if !ok {
		e.t.Fatalf("value is not string %+v", v)
	}
	require.Equal(e.t, value, val)
}

func (e *inputTestingEnvironment) requireEventTimestamp(nr int, ts string) {
	tm, err := time.Parse("2006-01-02T15:04:05.999", ts)
	if err != nil {
		e.t.Fatal(err)
	}
	events := make([]beat.Event, 0)
	for _, c := range e.pipeline.clients {
		events = append(events, c.GetEvents()...)
	}

	selectedEvent := events[nr]
	require.True(e.t, selectedEvent.Timestamp.Equal(tm), "got: %s, expected: %s", selectedEvent.Timestamp.String(), tm.String())
}

// logContains ensures s is a sub string on any log line.
// If s is not found, the test fails
func (e *inputTestingEnvironment) logContains(s string) {
	e.t.Helper()
	e.testLogger.LogContains(e.t, s)
}

func (e *inputTestingEnvironment) WaitLogsContains(s string, timeout time.Duration, msgAndArgs ...any) {
	e.t.Helper()
	e.testLogger.WaitLogsContains(e.t, s, timeout, msgAndArgs...)
}

var _ statestore.States = (*testInputStore)(nil)

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() statestore.States {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	s.registry.Close()
}

func (s *testInputStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}

type mockClient struct {
	publishing []beat.Event
	published  []beat.Event
	ackHandler beat.EventListener
	closed     atomic.Bool
	mtx        sync.Mutex
	canceler   context.CancelFunc
}

// GetEvents returns the published events
func (c *mockClient) GetEvents() []beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.published
}

// Publish mocks the Client Publish method
func (c *mockClient) Publish(e beat.Event) {
	c.PublishAll([]beat.Event{e})
}

// PublishAll mocks the Client PublishAll method
func (c *mockClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.publishing = append(c.publishing, events...)
	for _, event := range events {
		c.ackHandler.AddEvent(event, true)
	}
	c.ackHandler.ACKEvents(len(events))

	c.published = append(c.published, events...)
}

func (c *mockClient) waitUntilPublishingHasStarted() {
	for len(c.publishing) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
}

// Close mocks the Client Close method
func (c *mockClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("mock client already closed")
	}

	c.closed.Store(true)
	return nil
}

// mockPipelineConnector mocks the PipelineConnector interface
type mockPipelineConnector struct {
	blocking bool
	clients  []*mockClient
	mtx      sync.Mutex
}

// GetAllEvents returns all events associated with a pipeline
func (pc *mockPipelineConnector) GetAllEvents() []beat.Event {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

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

	ctx, cancel := context.WithCancel(context.Background())
	c := &mockClient{
		canceler:   cancel,
		ackHandler: newMockACKHandler(ctx, pc.blocking, config),
	}

	pc.clients = append(pc.clients, c)

	return c, nil
}

func (pc *mockPipelineConnector) cancelAllClients() {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	for _, client := range pc.clients {
		client.canceler()
	}
}

func newMockACKHandler(starter context.Context, blocking bool, config beat.ClientConfig) beat.EventListener {
	if !blocking {
		return config.EventListener
	}

	return acker.Combine(blockingACKer(starter), config.EventListener)
}

func blockingACKer(starter context.Context) beat.EventListener {
	return acker.EventPrivateReporter(func(acked int, private []any) {
		for starter.Err() == nil {
		}
	})
}

func (pc *mockPipelineConnector) clientsCount() int {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	return len(pc.clients)
}

func (pc *mockPipelineConnector) invertBlocking() {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	pc.blocking = !pc.blocking
}
