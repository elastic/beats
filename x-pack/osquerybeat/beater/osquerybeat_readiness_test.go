// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package beater

import (
	"context"
	"fmt"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/osquery/osquery-go"
	genosquery "github.com/osquery/osquery-go/gen/osquery"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Readiness must travel from osqueryd's first configuration pull through the
// runner to both the manager and the per-input status reporters.
func TestOsquerybeatReportsReadiness(t *testing.T) {
	inputs := &testManager{} // Records per-input status updates.
	ob, b, mgr := newStatusTestBeater(t, func(ob *osquerybeat) {
		ob.pub = &mockBeatPublisher{}
		ob.config.Inputs = []config.InputConfig{{Name: "osquery-manager-1", Type: osqueryInputType}}
		ob.osquerydFactory = func(socket string, _ ...osqd.Option) (osqd.Runner, error) {
			return &fakeOsqueryd{socket: socket, logf: t.Logf}, nil
		}
		ob.otelStatusFactoryWrapper = func(f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
			return reportingRunnerFactory{RunnerFactory: f, reporter: inputs}
		}
	})
	errCh := make(chan error, 1)
	go func() { errCh <- ob.Run(b) }()

	applied := statusEvent{Status: status.Running, Message: "Osqueryd configuration applied"}
	recorded := func(m *testManager) func() bool {
		return func() bool {
			m.mx.Lock()
			defer m.mx.Unlock()
			return slices.Contains(m.events, applied)
		}
	}
	assert.Eventually(t, recorded(mgr), 10*time.Second, 10*time.Millisecond, "the manager must hear that osqueryd is ready")
	assert.Eventually(t, recorded(inputs), time.Second, 10*time.Millisecond, "the inputs must hear that osqueryd is ready")

	ob.Stop()
	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled, "Run must stop after Stop")
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not stop")
	}
}

// reportingRunnerFactory gives each created runner a status reporter, as the
// OTel status wrapper does.
type reportingRunnerFactory struct {
	cfgfile.RunnerFactory
	reporter status.StatusReporter
}

func (f reportingRunnerFactory) Create(p beat.PipelineConnector, c *conf.C) (cfgfile.Runner, error) {
	runner, err := f.RunnerFactory.Create(p, c)
	if err != nil {
		return nil, err
	}
	runner.(status.WithStatusReporter).SetStatusReporter(f.reporter)
	return runner, nil
}

// fakeOsqueryd serves the osquery extension manager API on its socket and,
// like osqueryd, pulls the configuration once the extension registers.
type fakeOsqueryd struct {
	mockOsqueryd
	socket string
	logf   func(string, ...any)
}

func (f *fakeOsqueryd) SocketPath() string { return f.socket }

func (f *fakeOsqueryd) Run(ctx context.Context, _ osqd.Flags, onExit func(error)) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", f.socket)
	if err != nil {
		onExit(err)
		return err
	}
	var wg sync.WaitGroup
	manager := &fakeExtensionManager{ctx: ctx, socket: f.socket, logf: f.logf, wg: &wg}
	wg.Go(func() {
		<-ctx.Done()
		listener.Close()
	})
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		wg.Go(func() { serveExtensionManager(ctx, conn, manager) })
	}
	wg.Wait()
	onExit(ctx.Err())
	return ctx.Err()
}

func serveExtensionManager(ctx context.Context, conn net.Conn, handler genosquery.ExtensionManager) {
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	transport := thrift.NewTSocketFromConnConf(conn, &thrift.TConfiguration{})
	defer transport.Close()
	protocol := thrift.NewTBinaryProtocolConf(transport, nil)
	processor := genosquery.NewExtensionManagerProcessor(handler)
	for {
		if _, err := processor.Process(ctx, protocol, protocol); err != nil {
			return
		}
	}
}

type fakeExtensionManager struct {
	genosquery.ExtensionManager
	ctx    context.Context
	socket string
	logf   func(string, ...any)
	wg     *sync.WaitGroup
}

func (m *fakeExtensionManager) Ping(context.Context) (*genosquery.ExtensionStatus, error) {
	return &genosquery.ExtensionStatus{}, nil
}

func (m *fakeExtensionManager) Query(context.Context, string) (*genosquery.ExtensionResponse, error) {
	return &genosquery.ExtensionResponse{Status: &genosquery.ExtensionStatus{}}, nil
}

func (m *fakeExtensionManager) RegisterExtension(context.Context, *genosquery.InternalExtensionInfo, genosquery.ExtensionRegistry) (*genosquery.ExtensionStatus, error) {
	const uuid = 1
	m.wg.Go(func() { m.pullConfig(fmt.Sprintf("%s.%d", m.socket, uuid)) })
	return &genosquery.ExtensionStatus{UUID: uuid}, nil
}

func (m *fakeExtensionManager) DeregisterExtension(context.Context, genosquery.ExtensionRouteUUID) (*genosquery.ExtensionStatus, error) {
	return &genosquery.ExtensionStatus{}, nil
}

func (m *fakeExtensionManager) pullConfig(socket string) {
	client, err := osquery.NewClient(socket, 5*time.Second)
	if err != nil {
		m.logf("fake osqueryd: connect to extension: %v", err)
		return
	}
	defer client.Close()
	res, err := client.CallContext(m.ctx, "config", configPluginName, genosquery.ExtensionPluginRequest{"action": "genConfig"})
	if err != nil {
		m.logf("fake osqueryd: pull configuration: %v", err)
		return
	}
	if res.Status.Code != 0 {
		m.logf("fake osqueryd: pull configuration: %s", res.Status.Message)
	}
}
