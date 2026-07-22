// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
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

// mockStorageBeater is a minimal Beater that also implements
// backend.WithESStateStoreExtension so that BeatReceiver.Start's storage
// preflight path is exercised.
type mockStorageBeater struct {
	mockReceiverBeater
}

func (m *mockStorageBeater) WithESStateStoreExtension(_ backend.Registry) {}

// TestBeatReceiverStartFailureShutdownDoesNotHang is a regression test for the
// nil-runDone hang.
func TestBeatReceiverStartFailureShutdownDoesNotHang(t *testing.T) {
	mb := &mockStorageBeater{
		mockReceiverBeater: mockReceiverBeater{
			acked:    &atomic.Int64{},
			initDone: make(chan struct{}),
			done:     make(chan struct{}),
		},
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	cfg := map[string]any{
		"path.home": t.TempDir(),
		// Reference a storage extension that will not be present in the host.
		// This causes BeatReceiver.Start to return an error before launching
		// beater.Run, leaving runDone nil on the buggy path.
		"storage": "elasticsearch_storage/missing",
	}
	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		cfg,
		consumertest.NewNop(),
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")

	br, err := NewBeatReceiver(t.Context(), b, creator)
	require.NoError(t, err, "creating the beat receiver should succeed")

	// Reproduce the async wrapper pattern used by all beat receivers.
	var wg sync.WaitGroup
	wg.Go(func() {
		// Swallow the error, exactly as the wrapper receivers do.
		_ = br.Start(componenttest.NewNopHost())
	})
	// Wait for the goroutine to complete. Start has failed and returned; on the
	// buggy path runDone is still nil at this point.
	wg.Wait()

	// Shutdown must complete promptly even though Start failed before launching
	// beater.Run. Use t.Context() (no deadline during test execution) so that a
	// nil runDone would block indefinitely — a bounded context would mask the
	// bug by releasing the select via ctx.Done().
	done := make(chan error, 1)
	go func() { done <- br.Shutdown(t.Context()) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown hung after BeatReceiver.Start failed — nil runDone not fixed")
	}
}
