// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
	"github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/test"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func Test_internalBuilder(t *testing.T) {
	instance := fakeEC2Instance()
	instances := []*ec2Instance{instance}
	fetcher := newMockFetcher(instances, nil)
	log := logp.NewLogger("ec2")
	pBus := bus.New(log, "test")

	cfg := &awsauto.Config{
		Regions: []string{"us-east-1a", "us-west-1b"},
		Period:  time.Nanosecond,
	}

	uuid, _ := uuid.NewV4()
	k, _ := keystore.NewFileKeystore("test")
	provider, err := internalBuilder(uuid, pBus, cfg, fetcher, k)
	require.NoError(t, err)

	startListener := pBus.Subscribe("start")
	stopListener := pBus.Subscribe("stop")
	listenerDone := make(chan struct{})
	defer close(listenerDone)

	var events test.TestEventAccumulator
	go func() {
		for {
			select {
			case e := <-startListener.Events():
				events.Add(e)
			case e := <-stopListener.Events():
				events.Add(e)
			case <-listenerDone:
				return
			}
		}
	}()

	// Let run twice to ensure that duplicates don't create two start events
	// Since we're turning a list of assets into a list of changes the second once() call should be a noop
	provider.watcher.once()
	provider.watcher.once()
	events.WaitForNumEvents(t, 1, time.Second)

	assert.Equal(t, 1, events.Len())

	expectedStartEvent := bus.Event{
		"id":       instance.instanceID(),
		"provider": uuid,
		"start":    true,
		"aws": mapstr.M{
			"ec2": instance.toMap(),
		},
		"cloud": instance.toCloudMap(),
		"meta": mapstr.M{
			"aws": mapstr.M{
				"ec2": instance.toMap(),
			},
			"cloud": instance.toCloudMap(),
		},
	}

	require.Equal(t, expectedStartEvent, events.Get()[0])

	fetcher.setEC2s([]*ec2Instance{})

	// Let run twice to ensure that duplicates don't cause an issue
	provider.watcher.once()
	provider.watcher.once()
	events.WaitForNumEvents(t, 2, time.Second)

	require.Equal(t, 2, events.Len())

	expectedStopEvent := bus.Event{
		"stop":     true,
		"id":       awsauto.SafeString(instance.ec2Instance.InstanceId),
		"provider": uuid,
	}

	require.Equal(t, expectedStopEvent, events.Get()[1])

	// Test that in an error situation nothing changes.
	preErrorEventCount := events.Len()
	fetcher.setError(errors.New("oops"))

	// Let run twice to ensure that duplicates don't cause an issue
	provider.watcher.once()
	provider.watcher.once()

	assert.Equal(t, preErrorEventCount, events.Len())
}
