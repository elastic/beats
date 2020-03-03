// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

type rOp int

const (
	createOp rOp = iota + 1
	executeOp
	closeOp
)

func (r *rOp) String() string {
	m := map[rOp]string{
		1: "create",
		2: "execute",
		3: "close",
	}
	v, ok := m[*r]
	if !ok {
		return "unknown operation"
	}
	return v
}

type event struct {
	rk routingKey
	op rOp
}

type notifyFunc func(routingKey, rOp, ...interface{})

func TestRouter(t *testing.T) {
	programs := []program.Program{program.Program{Spec: program.Supported[1]}}

	t.Run("create new and destroy unused stream", func(t *testing.T) {
		recorder := &recorder{}
		r, err := newRouter(nil, recorder.factory)
		require.NoError(t, err)
		r.Dispatch("hello", map[routingKey][]program.Program{
			defautlRK: programs,
		})

		assertOps(t, []event{
			e(defautlRK, createOp),
			e(defautlRK, executeOp),
		}, recorder.events)

		recorder.reset()

		nk := "NEW_KEY"
		r.Dispatch("hello-2", map[routingKey][]program.Program{
			nk: programs,
		})

		assertOps(t, []event{
			e(nk, createOp),
			e(nk, executeOp),
			e(defautlRK, closeOp),
		}, recorder.events)
	})

	t.Run("multiples create new and destroy unused stream", func(t *testing.T) {
		k1 := "KEY_1"
		k2 := "KEY_2"

		recorder := &recorder{}
		r, err := newRouter(nil, recorder.factory)
		require.NoError(t, err)
		r.Dispatch("hello", map[routingKey][]program.Program{
			defautlRK: programs,
			k1:        programs,
			k2:        programs,
		})

		assertOps(t, []event{
			e(defautlRK, createOp),
			e(defautlRK, executeOp),

			e(k1, createOp),
			e(k1, executeOp),

			e(k2, createOp),
			e(k2, executeOp),
		}, recorder.events)

		recorder.reset()

		nk := "SECOND_DISPATCH"
		r.Dispatch("hello-2", map[routingKey][]program.Program{
			nk: programs,
		})

		assertOps(t, []event{
			e(nk, createOp),
			e(nk, executeOp),

			e(defautlRK, closeOp),
			e(k1, closeOp),
			e(k2, closeOp),
		}, recorder.events)
	})

	t.Run("create new and delegate program to existing stream", func(t *testing.T) {
		recorder := &recorder{}
		r, err := newRouter(nil, recorder.factory)
		require.NoError(t, err)
		r.Dispatch("hello", map[routingKey][]program.Program{
			defautlRK: programs,
		})

		assertOps(t, []event{
			e(defautlRK, createOp),
			e(defautlRK, executeOp),
		}, recorder.events)

		recorder.reset()

		r.Dispatch("hello-2", map[routingKey][]program.Program{
			defautlRK: programs,
		})

		assertOps(t, []event{
			e(defautlRK, executeOp),
		}, recorder.events)
	})

	t.Run("when no stream are detected we shutdown all the running streams", func(t *testing.T) {
		k1 := "KEY_1"
		k2 := "KEY_2"

		recorder := &recorder{}
		r, err := newRouter(nil, recorder.factory)
		require.NoError(t, err)
		r.Dispatch("hello", map[routingKey][]program.Program{
			defautlRK: programs,
			k1:        programs,
			k2:        programs,
		})

		assertOps(t, []event{
			e(defautlRK, createOp),
			e(defautlRK, executeOp),
			e(k1, createOp),
			e(k1, executeOp),
			e(k2, createOp),
			e(k2, executeOp),
		}, recorder.events)

		recorder.reset()

		r.Dispatch("hello-2", map[routingKey][]program.Program{})

		assertOps(t, []event{
			e(defautlRK, closeOp),
			e(k1, closeOp),
			e(k2, closeOp),
		}, recorder.events)
	})
}

type recorder struct {
	events []event
}

func (r *recorder) factory(_ *logger.Logger, rk routingKey) (stream, error) {
	return newMockStream(rk, r.notify), nil
}

func (r *recorder) notify(rk routingKey, op rOp, args ...interface{}) {
	r.events = append(r.events, e(rk, op))
}

func (r *recorder) reset() {
	r.events = nil
}

type mockStream struct {
	rk     routingKey
	notify notifyFunc
}

func newMockStream(rk routingKey, notify notifyFunc) *mockStream {
	notify(rk, createOp)
	return &mockStream{
		rk:     rk,
		notify: notify,
	}
}

func (m *mockStream) Execute(req *configRequest) error {
	m.event(executeOp, req)
	return nil
}

func (m *mockStream) Close() error {
	m.event(closeOp)
	return nil
}

func (m *mockStream) event(op rOp, args ...interface{}) {
	m.notify(m.rk, op, args...)
}

func assertOps(t *testing.T, expected []event, received []event) {
	require.Equal(t, len(expected), len(received), "Received number of operation doesn't match")
	require.Equal(t, expected, received)
}

func e(rk routingKey, op rOp) event {
	return event{rk: rk, op: op}
}
