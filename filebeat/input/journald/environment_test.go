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

// This file was contributed to by generative AI

//go:build linux

package journald

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/unison"
)

type inputTestingEnvironment struct {
	t              *testing.T
	workingDir     string
	stateStore     *testInputStore
	pipeline       *mockPipelineConnector
	statusReporter *mockStatusReporter

	pluginInitOnce sync.Once
	plugin         v2.Plugin

	inputLogger *logp.Logger
	logBuffer   *bytes.Buffer

	wg  sync.WaitGroup
	grp unison.TaskGroup
}

func newInputTestingEnvironment(t *testing.T) *inputTestingEnvironment {
	return &inputTestingEnvironment{
		t:              t,
		workingDir:     t.TempDir(),
		stateStore:     openTestStatestore(),
		pipeline:       &mockPipelineConnector{},
		statusReporter: &mockStatusReporter{},
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
	if err := manager.Init(&e.grp); err != nil {
		e.t.Fatalf("failed to initialise manager: %+v", err)
	}

	c := conf.MustNewConfigFrom(config)
	inp, err := manager.Create(c)
	if err != nil {
		e.t.Fatalf("failed to create input using manager: %+v", err)
	}

	return inp
}

func (e *inputTestingEnvironment) startInput(ctx context.Context, inp v2.Input) {
	e.wg.Add(1)
	t := e.t

	e.inputLogger, e.logBuffer = newInMemoryJSON()
	e.t.Cleanup(func() {
		if t.Failed() {
			folder := filepath.Join("..", "..", "build", "input-test")
			if err := os.MkdirAll(folder, 0o750); err != nil {
				t.Logf("cannot create folder for error logs: %s", err)
				return
			}

			f, err := os.CreateTemp(folder, "Filebeat-Test-Journald"+"-*")
			if err != nil {
				t.Logf("cannot create file for error logs: %s", err)
				return
			}
			defer f.Close()
			fullLogPath, err := filepath.Abs(f.Name())
			if err != nil {
				t.Logf("cannot get full path from log file: %s", err)
			}

			if _, err := f.Write(e.logBuffer.Bytes()); err != nil {
				t.Logf("cannot write to file: %s", err)
				return
			}

			t.Logf("Test Failed, logs from input at %q", fullLogPath)
		}
	})

	go func(wg *sync.WaitGroup, grp *unison.TaskGroup) {
		defer wg.Done()
		defer func() {
			if err := grp.Stop(); err != nil {
				e.t.Errorf("could not stop input: %s", err)
			}
		}()

		id := uuid.Must(uuid.NewV4()).String()
		inputCtx := v2.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     ctx,
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          e.inputLogger,
		}
		inputCtx = inputCtx.WithStatusReporter(e.statusReporter)
		if err := inp.Run(inputCtx, e.pipeline); err != nil {
			e.t.Errorf("input 'Run' method returned an error: %s", err)
		}
	}(&e.wg, &e.grp)
}

// waitUntilEventCount waits until total count events arrive to the client.
func (e *inputTestingEnvironment) waitUntilEventCount(count int) {
	e.t.Helper()
	msg := strings.Builder{}
	fmt.Fprintf(&msg, "did not find the expected %d events", count)
	require.Eventually(e.t, func() bool {
		sum := len(e.pipeline.GetAllEvents())
		if sum == count {
			return true
		}
		if count < sum {
			msg.Reset()
			fmt.Fprintf(&msg, "too many events; expected: %d, actual: %d", count, sum)
			return false
		}

		msg.Reset()
		fmt.Fprintf(&msg, "too few events; expected: %d, actual: %d", count, sum)

		return false
	}, 5*time.Second, 10*time.Millisecond, &msg)
}

// waitUntilEventCount waits until total count events arrive to the client.
func (e *inputTestingEnvironment) waitUntilEventsPublished(published int) {
	e.t.Helper()
	msg := strings.Builder{}
	require.Eventually(e.t, func() bool {
		sum := len(e.pipeline.GetAllEvents())
		if sum >= published {
			return true
		}

		msg.Reset()
		fmt.Fprintf(&msg, "too few events; expected: %d, actual: %d", published, sum)

		return false
	}, 5*time.Second, 10*time.Millisecond, &msg)
}

func (e *inputTestingEnvironment) RequireStatuses(expected []statusUpdate) {
	t := e.t
	t.Helper()
	got := e.statusReporter.GetUpdates()
	if len(got) != len(expected) {
		t.Fatalf("expecting %d updates, got %d", len(expected), len(got))
	}

	for i := range expected {
		g, e := got[i], expected[i]
		if g != e {
			t.Errorf(
				"expecting [%d] status update to be {state:%s, msg:%s}, got  {state:%s, msg:%s}",
				i, e.state.String(), e.msg, g.state.String(), g.msg,
			)
		}
	}
}

var _ statestore.States = (*testInputStore)(nil)

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

	c.published = append(c.published, events...)
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

func newMockACKHandler(starter context.Context, blocking bool, config beat.ClientConfig) beat.EventListener {
	if !blocking {
		return config.EventListener
	}

	return acker.Combine(blockingACKer(starter), config.EventListener)
}

func blockingACKer(starter context.Context) beat.EventListener {
	return acker.EventPrivateReporter(func(acked int, private []interface{}) {
		for starter.Err() == nil {
		}
	})
}

type statusUpdate struct {
	state status.Status
	msg   string
}

type mockStatusReporter struct {
	mutex   sync.RWMutex
	updates []statusUpdate
}

func (m *mockStatusReporter) UpdateStatus(status status.Status, msg string) {
	m.mutex.Lock()
	m.updates = append(m.updates, statusUpdate{status, msg})
	m.mutex.Unlock()
}

func (m *mockStatusReporter) GetUpdates() []statusUpdate {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return append([]statusUpdate{}, m.updates...)
}

func newInMemoryJSON() (*logp.Logger, *bytes.Buffer) {
	buff := bytes.Buffer{}
	encoderConfig := ecszap.ECSCompatibleEncoderConfig(logp.JSONEncoderConfig())
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(zapcore.AddSync(&buff)),
		zap.NewAtomicLevelAt(zap.DebugLevel))
	ecszap.ECSCompatibleEncoderConfig(logp.ConsoleEncoderConfig())

	logger, _ := logp.NewDevelopmentLogger(
		"journald",
		zap.WrapCore(func(in zapcore.Core) zapcore.Core {
			return core
		}))

	return logger, &buff
}
