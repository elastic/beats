// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestServerStart(t *testing.T) {
	origSetupRetryInterval := setupRetryInterval
	setupRetryInterval = 10 * time.Millisecond
	defer func() {
		setupRetryInterval = origSetupRetryInterval
	}()

	t.Run("succees", func(t *testing.T) {
		ms := newTestMetricset(t)
		err := ms.startServer(context.TODO())
		require.NoError(t, err)
		ms.stopServer()
	})

	t.Run("retry if port is in use", func(t *testing.T) {
		fakeServer, port, teardown := newTestUDPListener(t)
		defer teardown()

		ms := newTestMetricset(t, map[string]interface{}{"port": port})

		var serverErr error
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			serverErr = ms.startServer(context.TODO())
			if serverErr == nil {
				defer ms.stopServer()
			}
		}()

		time.Sleep(500 * time.Millisecond)
		fakeServer.Close()
		wg.Wait() // this blocks if server did not startup
		require.NoError(t, serverErr)
	})

	t.Run("cancel retry during shutdown", func(t *testing.T) {
		_, port, teardown := newTestUDPListener(t)
		defer teardown()

		ms := newTestMetricset(t, map[string]interface{}{"port": port})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var serverErr error
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			serverErr = ms.startServer(ctx)
			if serverErr == nil {
				defer ms.stopServer()
			}
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()
		require.Equal(t, context.Canceled, serverErr)
	})

	t.Run("no panic on shutdown if not started", func(t *testing.T) {
		_, port, teardown := newTestUDPListener(t)
		defer teardown()

		ms := newTestMetricset(t, map[string]interface{}{"port": port})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			ms.Run(ctx, nil)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()
	})
}

func newTestMetricset(t *testing.T, extraSettings ...map[string]interface{}) *MetricSet {
	settings := map[string]interface{}{
		"module":     "statsd",
		"metricsets": []string{"server"},
	}

	for _, other := range extraSettings {
		for k, v := range other {
			settings[k] = v
		}
	}

	ms := mbtest.NewPushMetricSetV2WithContext(t, settings)
	return ms.(*MetricSet)
}

func newTestUDPListener(t *testing.T) (*net.UDPConn, string, func()) {
	fakeServer, err := net.ListenUDP("udp4", nil)
	if err != nil {
		t.Fatal(err)
	}

	addr := fakeServer.LocalAddr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Fake UDP server with invalid address: %v", addr)
	}

	return fakeServer, port, func() {
		fakeServer.Close()
	}
}
