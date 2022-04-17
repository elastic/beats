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

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	input "github.com/menderesk/beats/v7/filebeat/input/v2"
	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/acker"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/statestore"
	"github.com/menderesk/beats/v7/libbeat/statestore/storetest"
	"github.com/menderesk/go-concert/unison"
)

type inputTestingEnvironment struct {
	t          *testing.T
	workingDir string
	stateStore *testInputStore
	pipeline   *mockPipelineConnector

	pluginInitOnce sync.Once
	plugin         v2.Plugin

	wg  sync.WaitGroup
	grp unison.TaskGroup
}

func newInputTestingEnvironment(t *testing.T) *inputTestingEnvironment {
	return &inputTestingEnvironment{
		t:          t,
		workingDir: t.TempDir(),
		stateStore: openTestStatestore(),
		pipeline:   &mockPipelineConnector{},
	}
}

func (e *inputTestingEnvironment) getManager() v2.InputManager {
	e.pluginInitOnce.Do(func() {
		e.plugin = Plugin(logp.L(), e.stateStore)
	})
	return e.plugin.Manager
}

func (e *inputTestingEnvironment) mustCreateInput(config map[string]interface{}) v2.Input {
	e.t.Helper()
	e.grp = unison.TaskGroup{}
	manager := e.getManager()
	if err := manager.Init(&e.grp, v2.ModeRun); err != nil {
		e.t.Fatalf("failed to initialise manager: %+v", err)
	}

	c := common.MustNewConfigFrom(config)
	inp, err := manager.Create(c)
	if err != nil {
		e.t.Fatalf("failed to create input using manager: %+v", err)
	}

	return inp
}

func (e *inputTestingEnvironment) startInput(ctx context.Context, inp v2.Input) {
	e.wg.Add(1)
	go func(wg *sync.WaitGroup, grp *unison.TaskGroup) {
		defer wg.Done()
		defer grp.Stop()

		inputCtx := input.Context{Logger: logp.L(), Cancelation: ctx}
		inp.Run(inputCtx, e.pipeline)
	}(&e.wg, &e.grp)
}

// waitUntilEventCount waits until total count events arrive to the client.
func (e *inputTestingEnvironment) waitUntilEventCount(count int) {
	e.t.Helper()
	for {
		sum := len(e.pipeline.GetAllEvents())
		if sum == count {
			return
		}
		if count < sum {
			e.t.Fatalf("too many events; expected: %d, actual: %d", count, sum)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (e *inputTestingEnvironment) waitUntilInputStops() {
	e.wg.Wait()
}

func (e *inputTestingEnvironment) abspath(filename string) string {
	return filepath.Join(e.workingDir, filename)
}

func (e *inputTestingEnvironment) mustWriteFile(filename string, lines []byte) {
	e.t.Helper()
	path := e.abspath(filename)
	if err := os.WriteFile(path, lines, 0o644); err != nil {
		e.t.Fatalf("failed to write file '%s': %+v", path, err)
	}
}

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() *testInputStore {
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
	publishing []beat.Event
	published  []beat.Event
	ackHandler beat.ACKer
	closed     bool
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

	for _, event := range events {
		c.published = append(c.published, event)
	}
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

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
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

func (pc *mockPipelineConnector) cancelClient(i int) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	if len(pc.clients) < i+1 {
		return
	}

	pc.clients[i].canceler()
}

func newMockACKHandler(starter context.Context, blocking bool, config beat.ClientConfig) beat.ACKer {
	if !blocking {
		return config.ACKHandler
	}

	return acker.Combine(blockingACKer(starter), config.ACKHandler)
}

func blockingACKer(starter context.Context) beat.ACKer {
	return acker.EventPrivateReporter(func(acked int, private []interface{}) {
		for starter.Err() == nil {
		}
	})
}
