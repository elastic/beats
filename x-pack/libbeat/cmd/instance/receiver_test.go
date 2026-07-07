// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// mockReceiverBeater is a minimal Beater that publishes a fixed number of
// events through the receiver's publisher pipeline and blocks until Stop. On
// stop it closes its client (stage one of the two-stage shutdown), modeling a
// well-behaved Beater that owns its inputs' shutdown (issue #49794).
type mockReceiverBeater struct {
	npub     int
	acked    *atomic.Int64
	initDone chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func (m *mockReceiverBeater) Run(b *beat.Beat) error {
	client, err := b.Publisher.ConnectWith(beat.ClientConfig{
		EventListener: acker.RawCounting(func(n int) { m.acked.Add(int64(n)) }),
	})
	if err != nil {
		return err
	}

	for i := 0; i < m.npub; i++ {
		client.Publish(beat.Event{
			Timestamp: time.Now(),
			Fields:    mapstr.M{"n": i},
		})
	}
	close(m.initDone)

	<-m.done
	// Beater owns shutdown sequencing: close the client before Run returns so
	// the pipeline can drain and finalize acknowledgments on Disconnect.
	_ = client.Close()
	return nil
}

func (m *mockReceiverBeater) Stop() {
	m.stopOnce.Do(func() { close(m.done) })
}

// TestBeatReceiverStartShutdown exercises the full beat-receiver lifecycle end
// to end: it builds a real BeatReceiver backed by the slabqueue-pool publisher
// pipeline (NewForReceiver), starts it, publishes events, and shuts it down. It
// verifies that:
//   - Shutdown completes promptly (it is bounded by receiverPublisherCloseTimeout,
//     so it never hangs even though the output drains during disconnect), and
//   - every published event is acknowledged by the time Shutdown returns, which
//     proves the output stays running and drains acks while the pipeline is
//     being disconnected (issues #50104, #50105, #49794).
func TestBeatReceiverStartShutdown(t *testing.T) {
	const npub = 5
	acked := &atomic.Int64{}
	mb := &mockReceiverBeater{
		npub:     npub,
		acked:    acked,
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	cfg := map[string]any{"path.home": t.TempDir()}
	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		cfg,
		consumertest.NewNop(), // accepts every batch -> events get acknowledged
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	// Start blocks in beater.Run, so run it in a goroutine.
	startErr := make(chan error, 1)
	go func() { startErr <- br.Start(componenttest.NewNopHost()) }()

	// Wait until the beater is running and has published its events.
	select {
	case <-mb.initDone:
	case <-time.After(30 * time.Second):
		t.Fatal("beater did not start")
	}

	// Shutdown must complete promptly. If the output were torn down before the
	// queue drained, this would block for the full close timeout (or forever);
	// the timeout guard here catches a hang.
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- br.Shutdown(t.Context()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown should not error")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown hung — the output is likely not draining acknowledgments during disconnect")
	}

	// beater.Run (and therefore Start) must have returned after Stop.
	select {
	case err := <-startErr:
		require.NoError(t, err, "beater.Run should return cleanly")
	case <-time.After(10 * time.Second):
		t.Fatal("beater.Run did not return after Stop")
	}

	// Every published event must have been acknowledged by the time Shutdown
	// returned: this is the key end-to-end assertion that the output kept
	// consuming and acking while the pipeline was disconnected.
	assert.Equal(t, int64(npub), acked.Load(),
		"all published events must be acknowledged by the time Shutdown returns")
}
