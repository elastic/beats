// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"go.elastic.co/apm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func TestServer_Register(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	_, err := srv.Register(app, initConfig)
	assert.NoError(t, err)
	_, err = srv.Register(app, initConfig)
	assert.Equal(t, ErrApplicationAlreadyRegistered, err)
}

func TestServer_Get(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	expected, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	observed, ok := srv.Get(app)
	assert.True(t, ok)
	assert.Equal(t, expected, observed)
	_, found := srv.Get(&StubApp{})
	assert.False(t, found)
}

func TestServer_InitialCheckIn(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// client should get initial check-in
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))

	// set status as healthy and running
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)

	// application state should be updated
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))
}

func TestServer_MultiClients(t *testing.T) {
	initConfig1 := "initial_config_1"
	initConfig2 := "initial_config_2"
	app1 := &StubApp{}
	app2 := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as1, err := srv.Register(app1, initConfig1)
	require.NoError(t, err)
	cImpl1 := &StubClientImpl{}
	c1 := newClientFromApplicationState(t, as1, cImpl1)
	require.NoError(t, c1.Start(context.Background()))
	defer c1.Stop()
	as2, err := srv.Register(app2, initConfig2)
	require.NoError(t, err)
	cImpl2 := &StubClientImpl{}
	c2 := newClientFromApplicationState(t, as2, cImpl2)
	require.NoError(t, c2.Start(context.Background()))
	defer c2.Stop()

	// clients should get initial check-ins
	require.NoError(t, waitFor(func() error {
		if cImpl1.Config() != initConfig1 {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	require.NoError(t, waitFor(func() error {
		if cImpl2.Config() != initConfig2 {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))

	// set status differently
	c1.Status(proto.StateObserved_HEALTHY, "Running", nil)
	c2.Status(proto.StateObserved_DEGRADED, "No upstream connection", nil)

	// application states should be updated
	assert.NoError(t, waitFor(func() error {
		if app1.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))
	assert.NoError(t, waitFor(func() error {
		if app2.Status() != proto.StateObserved_DEGRADED {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))
}

func TestServer_PreventCheckinStream(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	as.checkinConn = false // prevent connection to check-in stream
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	assert.NoError(t, waitFor(func() error {
		if cImpl.Error() == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error())
		if !ok {
			return fmt.Errorf("client didn't get a status error")
		}
		if s.Code() != codes.Unavailable {
			return fmt.Errorf("client didn't get unavaible error")
		}
		return nil
	}))
}

func TestServer_PreventActionsStream(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	as.actionsConn = false // prevent connection to check-in stream
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	assert.NoError(t, waitFor(func() error {
		if cImpl.Error() == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error())
		if !ok {
			return fmt.Errorf("client didn't get a status error")
		}
		if s.Code() != codes.Unavailable {
			return fmt.Errorf("client didn't get unavaible error")
		}
		return nil
	}))
}

func TestServer_DestroyPreventConnectAtTLS(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	as.Destroy()
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	assert.NoError(t, waitFor(func() error {
		if cImpl.Error() == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error())
		if !ok {
			return fmt.Errorf("client didn't get a status error")
		}
		if s.Code() != codes.Unavailable {
			return fmt.Errorf("client didn't get unavaible error")
		}
		if !strings.Contains(s.Message(), "authentication handshake failed") {
			return fmt.Errorf("client didn't get authentication handshake failed error")
		}
		return nil
	}))
}

func TestServer_UpdateConfig(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// push same config; should not increment config index
	preIdx := as.expectedConfigIdx
	require.NoError(t, as.UpdateConfig(initConfig))
	assert.Equal(t, preIdx, as.expectedConfigIdx)

	// push new config; should update the client
	newConfig := "new_config"
	require.NoError(t, as.UpdateConfig(newConfig))
	assert.Equal(t, preIdx+1, as.expectedConfigIdx)
	assert.NoError(t, waitFor(func() error {
		if cImpl.Config() != newConfig {
			return fmt.Errorf("client never got updated config")
		}
		return nil
	}))
}

func TestServer_UpdateConfigDisconnected(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// stop the client, then update the config
	c.Stop()
	newConfig := "new_config"
	require.NoError(t, as.UpdateConfig(newConfig))

	// reconnect, client should get latest config
	require.NoError(t, c.Start(context.Background()))
	assert.NoError(t, waitFor(func() error {
		if cImpl.Config() != newConfig {
			return fmt.Errorf("client never got updated config")
		}
		return nil
	}))
}

func TestServer_UpdateConfigStopping(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// perform stop try to update config (which will error)
	done := make(chan bool)
	go func() {
		_ = as.Stop(500 * time.Millisecond)
		close(done)
	}()
	err = as.UpdateConfig("new_config")
	assert.Error(t, ErrApplicationStopping, err)
	<-done
}

func TestServer_Stop(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// send stop to the client
	done := make(chan bool)
	var stopErr error
	go func() {
		stopErr = as.Stop(time.Second * 5)
		close(done)
	}()

	// process of testing the flow
	//   1. server sends stop
	//   2. client sends configuring
	//   3. server sends stop again
	//   4. client sends stopping
	//   5. client disconnects
	require.NoError(t, waitFor(func() error {
		if cImpl.Stop() == 0 {
			return fmt.Errorf("client never got expected stop")
		}
		return nil
	}))
	c.Status(proto.StateObserved_CONFIGURING, "Configuring", nil)
	require.NoError(t, waitFor(func() error {
		if cImpl.Stop() < 1 {
			return fmt.Errorf("client never got expected stop again")
		}
		return nil
	}))
	c.Status(proto.StateObserved_STOPPING, "Stopping", nil)
	require.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_STOPPING {
			return fmt.Errorf("server never updated to stopping")
		}
		return nil
	}))
	c.Stop()
	<-done

	// no error on stop
	assert.NoError(t, stopErr)
}

func TestServer_StopJustDisconnect(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// send stop to the client
	done := make(chan bool)
	var stopErr error
	go func() {
		stopErr = as.Stop(time.Second * 5)
		close(done)
	}()

	// process of testing the flow
	//   1. server sends stop
	//   2. client disconnects
	require.NoError(t, waitFor(func() error {
		if cImpl.Stop() == 0 {
			return fmt.Errorf("client never got expected stop")
		}
		return nil
	}))
	c.Stop()
	<-done

	// no error on stop
	assert.NoError(t, stopErr)
}

func TestServer_StopTimeout(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl)
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// clients should get initial check-ins then set as healthy
	require.NoError(t, waitFor(func() error {
		if cImpl.Config() != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	c.Status(proto.StateObserved_HEALTHY, "Running", nil)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))

	// send stop to the client
	done := make(chan bool)
	var stopErr error
	go func() {
		stopErr = as.Stop(time.Millisecond)
		close(done)
	}()

	// don't actually stop the client

	// timeout error on stop
	<-done
	assert.Equal(t, ErrApplicationStopTimedOut, stopErr)
}

func TestServer_WatchdogFailApp(t *testing.T) {
	initConfig := "initial_config"
	checkMinTimeout := 300 * time.Millisecond
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{}, func(s *Server) {
		s.watchdogCheckInterval = 100 * time.Millisecond
		s.checkInMinTimeout = checkMinTimeout
	})
	defer srv.Stop()
	_, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_DEGRADED {
			return fmt.Errorf("app status nevers set to degraded")
		}
		return nil
	}))
	assert.Equal(t, "Missed last check-in", app.Message())
	assert.NoError(t, waitFor(func() error {
		if app.Status() != proto.StateObserved_FAILED {
			return fmt.Errorf("app status nevers set to degraded")
		}
		return nil
	}))
	assert.Equal(t, "Missed two check-ins", app.Message())
}

func TestServer_PerformAction(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{}, func(s *Server) {
		s.watchdogCheckInterval = 50 * time.Millisecond
	})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl := &StubClientImpl{}
	c := newClientFromApplicationState(t, as, cImpl, &EchoAction{}, &SleepAction{})
	require.NoError(t, c.Start(context.Background()))
	defer c.Stop()

	// successful action
	resp, err := as.PerformAction("echo", map[string]interface{}{
		"echo": "hello world",
	}, 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"echo": "hello world",
	}, resp)

	// action error client-side
	_, err = as.PerformAction("echo", map[string]interface{}{
		"bad_param": "hello world",
	}, 5*time.Second)
	require.Error(t, err)

	// very slow action that times out
	_, err = as.PerformAction("sleep", map[string]interface{}{
		"sleep": time.Second,
	}, 10*time.Millisecond)
	require.Error(t, err)
	assert.Equal(t, ErrActionTimedOut, err)

	// try slow action again with the client disconnected (should timeout the same)
	c.Stop()
	require.NoError(t, waitFor(func() error {
		as.actionsLock.RLock()
		defer as.actionsLock.RUnlock()
		if as.actionsDone != nil {
			return fmt.Errorf("client never disconnected the actions stream")
		}
		return nil
	}))
	_, err = as.PerformAction("sleep", map[string]interface{}{
		"sleep": time.Second,
	}, 10*time.Millisecond)
	require.Error(t, err)
	assert.Equal(t, ErrActionTimedOut, err)

	// perform action, reconnect client, and then action should be performed
	done := make(chan bool)
	go func() {
		_, err = as.PerformAction("sleep", map[string]interface{}{
			"sleep": 100 * time.Millisecond,
		}, 5*time.Second)
		close(done)
	}()
	require.NoError(t, c.Start(context.Background()))
	<-done
	require.NoError(t, err)

	// perform action, destroy application
	done = make(chan bool)
	go func() {
		_, err = as.PerformAction("sleep", map[string]interface{}{
			"sleep": time.Second,
		}, 5*time.Second)
		close(done)
	}()
	<-time.After(100 * time.Millisecond)
	as.Destroy()
	<-done
	require.Error(t, err)
	assert.Equal(t, ErrActionCancelled, err)

	// perform action after destroy returns cancelled
	_, err = as.PerformAction("sleep", map[string]interface{}{
		"sleep": time.Second,
	}, 5*time.Second)
	assert.Equal(t, ErrActionCancelled, err)
}

func newErrorLogger(t *testing.T) *logger.Logger {
	t.Helper()

	loggerCfg := logger.DefaultLoggingConfig()
	loggerCfg.Level = logp.ErrorLevel

	log, err := logger.NewFromConfig("", loggerCfg, false)
	require.NoError(t, err)
	return log
}

func createAndStartServer(t *testing.T, handler Handler, extraConfigs ...func(*Server)) *Server {
	t.Helper()
	tracer := apm.DefaultTracer
	srv, err := New(newErrorLogger(t), "localhost:0", handler, tracer)
	require.NoError(t, err)
	for _, extra := range extraConfigs {
		extra(srv)
	}
	require.NoError(t, srv.Start())
	return srv
}

func newClientFromApplicationState(t *testing.T, as *ApplicationState, impl client.StateInterface, actions ...client.Action) client.Client {
	t.Helper()

	var err error
	var c client.Client
	var wg sync.WaitGroup
	r, w := io.Pipe()
	wg.Add(1)
	go func() {
		c, err = client.NewFromReader(r, impl, actions...)
		wg.Done()
	}()

	require.NoError(t, as.WriteConnInfo(w))
	wg.Wait()
	require.NoError(t, err)
	return c
}

type StubApp struct {
	lock    sync.RWMutex
	status  proto.StateObserved_Status
	message string
	payload map[string]interface{}
}

func (a *StubApp) Status() proto.StateObserved_Status {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.status
}

func (a *StubApp) Message() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.message
}

type StubHandler struct{}

func (h *StubHandler) OnStatusChange(as *ApplicationState, status proto.StateObserved_Status, message string, payload map[string]interface{}) {
	stub := as.app.(*StubApp)
	stub.lock.Lock()
	defer stub.lock.Unlock()
	stub.status = status
	stub.message = message
	stub.payload = payload
}

type StubClientImpl struct {
	Lock   sync.RWMutex
	config string
	stop   int
	error  error
}

func (c *StubClientImpl) Config() string {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	return c.config
}

func (c *StubClientImpl) Stop() int {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	return c.stop
}

func (c *StubClientImpl) Error() error {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	return c.error
}

func (c *StubClientImpl) OnConfig(config string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.config = config
}

func (c *StubClientImpl) OnStop() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.stop++
}

func (c *StubClientImpl) OnError(err error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.error = err
}

type EchoAction struct{}

func (*EchoAction) Name() string {
	return "echo"
}

func (*EchoAction) Execute(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	echoRaw, ok := request["echo"]
	if !ok {
		return nil, fmt.Errorf("missing required param of echo")
	}
	return map[string]interface{}{
		"echo": echoRaw,
	}, nil
}

type SleepAction struct{}

func (*SleepAction) Name() string {
	return "sleep"
}

func (*SleepAction) Execute(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	sleepRaw, ok := request["sleep"]
	if !ok {
		return nil, fmt.Errorf("missing required param of slow")
	}
	sleep, ok := sleepRaw.(float64)
	if !ok {
		return nil, fmt.Errorf("sleep param must be a number")
	}
	timer := time.NewTimer(time.Duration(sleep))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	return map[string]interface{}{}, nil
}

func waitFor(check func() error) error {
	started := time.Now()
	for {
		err := check()
		if err == nil {
			return nil
		}
		if time.Since(started) >= 5*time.Second {
			return fmt.Errorf("check timed out after 5 second: %s", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
