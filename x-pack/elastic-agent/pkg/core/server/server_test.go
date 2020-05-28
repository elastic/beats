// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		if cImpl.Config != initConfig {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))

	// set status as healthy and running
	c.Status(proto.StateObserved_HEALTHY, "Running")

	// application state should be updated
	assert.NoError(t, waitFor(func() error {
		if app.status != proto.StateObserved_HEALTHY {
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
		if cImpl1.Config != initConfig1 {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))
	require.NoError(t, waitFor(func() error {
		if cImpl2.Config != initConfig2 {
			return fmt.Errorf("client never got intial config")
		}
		return nil
	}))

	// set status differently
	c1.Status(proto.StateObserved_HEALTHY, "Running")
	c2.Status(proto.StateObserved_DEGRADED, "No upstream connection")

	// application states should be updated
	assert.NoError(t, waitFor(func() error {
		if app1.status != proto.StateObserved_HEALTHY {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))
	assert.NoError(t, waitFor(func() error {
		if app2.status != proto.StateObserved_DEGRADED {
			return fmt.Errorf("server never updated currect application state")
		}
		return nil
	}))
}

func TestServer_PreventMultipleStreams(t *testing.T) {
	initConfig := "initial_config"
	app := &StubApp{}
	srv := createAndStartServer(t, &StubHandler{})
	defer srv.Stop()
	as, err := srv.Register(app, initConfig)
	require.NoError(t, err)
	cImpl1 := &StubClientImpl{}
	c1 := newClientFromApplicationState(t, as, cImpl1)
	require.NoError(t, c1.Start(context.Background()))
	defer c1.Stop()
	cImpl2 := &StubClientImpl{}
	c2 := newClientFromApplicationState(t, as, cImpl2)
	require.NoError(t, c2.Start(context.Background()))
	defer c2.Stop()

	assert.NoError(t, waitFor(func() error {
		if cImpl2.Error == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl2.Error)
		if !ok {
			return fmt.Errorf("client didn't get a status error")
		}
		if s.Code() != codes.AlreadyExists {
			return fmt.Errorf("client didn't get already exists error")
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
		if cImpl.Error == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error)
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
		if cImpl.Error == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error)
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
		if cImpl.Error == nil {
			return fmt.Errorf("client never got error trying to connect twice")
		}
		s, ok := status.FromError(cImpl.Error)
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

func newErrorLogger(t *testing.T) *logger.Logger {
	t.Helper()
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"logging": map[string]interface{}{
			"level": "error",
		},
	})
	require.NoError(t, err)
	log, err := logger.NewFromConfig(cfg)
	require.NoError(t, err)
	return log
}

func createAndStartServer(t *testing.T, handler Handler, extraConfigs ...func(*Server)) *Server {
	t.Helper()
	srv, err := New(newErrorLogger(t), ":6688", handler)
	require.NoError(t, err)
	for _, extra := range extraConfigs {
		extra(srv)
	}
	require.NoError(t, srv.Start())
	return srv
}

func newClientFromApplicationState(t *testing.T, as *ApplicationState, impl client.StateInterface) *client.Client {
	t.Helper()

	var err error
	var c *client.Client
	var wg sync.WaitGroup
	r, w := io.Pipe()
	wg.Add(1)
	go func() {
		c, err = client.NewFromReader(r, impl)
		wg.Done()
	}()

	require.NoError(t, as.WriteConnInfo(w))
	wg.Wait()
	require.NoError(t, err)
	return c
}

type StubApp struct{
	status proto.StateObserved_Status
	message string
}

type StubHandler struct {}

func (h *StubHandler) OnStatusChange(as *ApplicationState, status proto.StateObserved_Status, message string) {
	stub := as.app.(*StubApp)
	stub.status = status
	stub.message = message
}

type StubClientImpl struct {
	Lock   sync.Mutex
	Config string
	Stop   bool
	Error  error
}

func (c *StubClientImpl) OnConfig(config string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.Config = config
}

func (c *StubClientImpl) OnStop() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.Stop = true
}

func (c *StubClientImpl) OnError(err error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.Error = err
}

func waitFor(check func() error) error {
	started := time.Now()
	for {
		err := check()
		if err == nil {
			return nil
		}
		if time.Now().Sub(started) >= 5*time.Second {
			return fmt.Errorf("check timed out after 5 second: %s", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
