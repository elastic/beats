// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type testEventAccumulator struct {
	events []bus.Event
	lock   sync.Mutex
}

func (tea *testEventAccumulator) add(e bus.Event) {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	tea.events = append(tea.events, e)
}

func (tea *testEventAccumulator) len() int {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	return len(tea.events)
}

func (tea *testEventAccumulator) get() []bus.Event {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	res := make([]bus.Event, len(tea.events))
	copy(res, tea.events)
	return res
}

func (tea *testEventAccumulator) waitForNumEvents(t *testing.T, targetLen int, timeout time.Duration) {
	start := time.Now()

	for time.Since(start) < timeout {
		if tea.len() >= targetLen {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("Timed out waiting for num events to be %d", targetLen)
}

func Test_internalBuilder(t *testing.T) {
	log := logp.NewLogger("elb")
	lbl := fakeLbl()
	lbls := []*lbListener{lbl}
	fetcher := newMockFetcher(lbls, nil)
	pBus := bus.New(log, "test")

	cfg := &awsauto.Config{
		Regions: []string{"us-east-1a", "us-west-1b"},
		Period:  time.Nanosecond,
	}

	uuid, _ := uuid.NewV4()
	k, _ := keystore.NewFileKeystore("test")
	provider, err := internalBuilder(uuid, pBus, cfg, fetcher, k, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	startListener := pBus.Subscribe("start")
	stopListener := pBus.Subscribe("stop")
	listenerDone := make(chan struct{})
	defer close(listenerDone)

	var events testEventAccumulator
	go func() {
		for {
			select {
			case e := <-startListener.Events():
				events.add(e)
			case e := <-stopListener.Events():
				events.add(e)
			case <-listenerDone:
				return
			}
		}
	}()

	// Let run twice to ensure that duplicates don't create two start events
	// Since we're turning a list of assets into a list of changes the second once() call should be a noop
	require.NoError(t, provider.watcher.once())
	require.NoError(t, provider.watcher.once())
	events.waitForNumEvents(t, 1, time.Second)

	assert.Equal(t, 1, events.len())

	expectedStartEvent := bus.Event{
		"id":       lbl.arn(),
		"provider": uuid,
		"start":    true,
		"host":     *lbl.lb.DNSName,
		"port":     *lbl.listener.Port,
		"aws": mapstr.M{
			"elb": lbl.toMap(),
		},
		"cloud": lbl.toCloudMap(),
		"meta": mapstr.M{
			"aws": mapstr.M{
				"elb": lbl.toMap(),
			},
			"cloud": lbl.toCloudMap(),
		},
	}

	require.Equal(t, expectedStartEvent, events.get()[0])

	fetcher.setLbls([]*lbListener{})

	// Let run twice to ensure that duplicates don't cause an issue
	require.NoError(t, provider.watcher.once())
	require.NoError(t, provider.watcher.once())
	events.waitForNumEvents(t, 2, time.Second)

	require.Equal(t, 2, events.len())

	expectedStopEvent := bus.Event{
		"stop":     true,
		"id":       lbl.arn(),
		"provider": uuid,
	}

	require.Equal(t, expectedStopEvent, events.get()[1])

	// Test that in an error situation nothing changes.
	preErrorEventCount := events.len()
	fetcher.setError(errors.New("oops"))

	// Let run twice to ensure that duplicates don't cause an issue
	//nolint:errcheck // ignore
	provider.watcher.once()
	//nolint:errcheck // ignore
	provider.watcher.once()

	assert.Equal(t, preErrorEventCount, events.len())
}
