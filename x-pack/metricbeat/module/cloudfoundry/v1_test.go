// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package cloudfoundry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	cfcommon "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

func TestDispatcher(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))
	log := logp.NewLogger("cloudfoundry")

	assertEventType := func(t *testing.T, expected string, e mb.Event) {
		t.Helper()
		cf := e.RootFields["cloudfoundry"].(common.MapStr)
		assert.Equal(t, expected, cf["type"])
	}

	waitFor := func(t *testing.T, expected string, r pushReporter) {
		t.Helper()
		select {
		case e := <-r.events:
			assertEventType(t, expected, e)
		default:
			t.Errorf("expected %s event", expected)
		}
	}

	t.Run("subscribe to one type", func(t *testing.T) {
		d := newEventDispatcher(log)
		r := pushReporter{events: make(chan mb.Event, 1)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &r,
		})

		d.dispatch(&cfcommon.EventCounter{})
		waitFor(t, "counter", r)
	})

	t.Run("subscribe and unsubscribe", func(t *testing.T) {
		d := newEventDispatcher(log)
		r := pushReporter{events: make(chan mb.Event, 1)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &r,
		})

		d.dispatch(&cfcommon.EventCounter{})
		waitFor(t, "counter", r)

		d.handleSubscription(subscription{
			eventType:   cfcommon.EventTypeCounter,
			reporter:    &r,
			unsubscribe: true,
		})

		assert.True(t, d.empty())
		d.dispatch(&cfcommon.EventCounter{})

		select {
		case <-r.events:
			t.Errorf("shouldn't receive on this reporter")
		default:
		}
	})

	t.Run("subscribe to two types", func(t *testing.T) {
		d := newEventDispatcher(log)

		counterReporter := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &counterReporter,
		})

		valueReporter := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeValueMetric,
			reporter:  &valueReporter,
		})

		d.dispatch(&cfcommon.EventCounter{})
		d.dispatch(&cfcommon.EventValueMetric{})

		waitFor(t, "counter", counterReporter)
		waitFor(t, "value", valueReporter)
	})

	t.Run("subscribe to two types, receive only from one", func(t *testing.T) {
		d := newEventDispatcher(log)

		counterReporter := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &counterReporter,
		})

		valueReporter := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeValueMetric,
			reporter:  &valueReporter,
		})

		d.dispatch(&cfcommon.EventCounter{})
		d.dispatch(&cfcommon.EventCounter{})

		select {
		case <-valueReporter.events:
			t.Errorf("shouldn't receive on this reporter")
		default:
		}

		waitFor(t, "counter", counterReporter)
		waitFor(t, "counter", counterReporter)
	})

	t.Run("subscribe twice to same type, ignore second", func(t *testing.T) {
		d := newEventDispatcher(log)
		first := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &first,
		})
		d.dispatch(&cfcommon.EventCounter{})
		waitFor(t, "counter", first)

		second := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &second,
		})

		d.dispatch(&cfcommon.EventCounter{})
		select {
		case <-second.events:
			t.Errorf("shouldn't receive on this reporter")
		default:
		}
		waitFor(t, "counter", first)
	})

	t.Run("unsubscribe not subscribed reporters, first one continues subscribed", func(t *testing.T) {
		d := newEventDispatcher(log)
		r := pushReporter{events: make(chan mb.Event, 2)}
		d.handleSubscription(subscription{
			eventType: cfcommon.EventTypeCounter,
			reporter:  &r,
		})
		d.dispatch(&cfcommon.EventCounter{})
		waitFor(t, "counter", r)

		d.handleSubscription(subscription{
			eventType:   cfcommon.EventTypeCounter,
			reporter:    &pushReporter{},
			unsubscribe: true,
		})
		d.dispatch(&cfcommon.EventCounter{})
		waitFor(t, "counter", r)
	})
}

type pushReporter struct {
	events chan mb.Event
}

func (r *pushReporter) Done() <-chan struct{} { return nil }
func (r *pushReporter) Error(err error) bool  { return true }
func (r *pushReporter) Event(e mb.Event) bool {
	r.events <- e
	return true
}
