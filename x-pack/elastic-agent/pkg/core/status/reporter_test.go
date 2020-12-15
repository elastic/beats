// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func TestReporter(t *testing.T) {
	l, _ := logger.New("")
	t.Run("healthy by default", func(t *testing.T) {
		r := NewController(l)
		assert.Equal(t, Healthy, r.Status())
		assert.Equal(t, "online", r.StatusString())
	})

	t.Run("healthy when all healthy", func(t *testing.T) {
		r := NewController(l)
		r1 := r.Register("r1")
		r2 := r.Register("r2")
		r3 := r.Register("r3")

		r1.Update(Healthy)
		r2.Update(Healthy)
		r3.Update(Healthy)

		assert.Equal(t, Healthy, r.Status())
		assert.Equal(t, "online", r.StatusString())
	})

	t.Run("degraded when one degraded", func(t *testing.T) {
		r := NewController(l)
		r1 := r.Register("r1")
		r2 := r.Register("r2")
		r3 := r.Register("r3")

		r1.Update(Healthy)
		r2.Update(Degraded)
		r3.Update(Healthy)

		assert.Equal(t, Degraded, r.Status())
		assert.Equal(t, "degraded", r.StatusString())
	})

	t.Run("failed when one failed", func(t *testing.T) {
		r := NewController(l)
		r1 := r.Register("r1")
		r2 := r.Register("r2")
		r3 := r.Register("r3")

		r1.Update(Healthy)
		r2.Update(Failed)
		r3.Update(Healthy)

		assert.Equal(t, Failed, r.Status())
		assert.Equal(t, "error", r.StatusString())
	})

	t.Run("failed when one failed and one degraded", func(t *testing.T) {
		r := NewController(l)
		r1 := r.Register("r1")
		r2 := r.Register("r2")
		r3 := r.Register("r3")

		r1.Update(Healthy)
		r2.Update(Failed)
		r3.Update(Degraded)

		assert.Equal(t, Failed, r.Status())
		assert.Equal(t, "error", r.StatusString())
	})

	t.Run("degraded when degraded and healthy, failed unregistered", func(t *testing.T) {
		r := NewController(l)
		r1 := r.Register("r1")
		r2 := r.Register("r2")
		r3 := r.Register("r3")

		r1.Update(Healthy)
		r2.Update(Failed)
		r3.Update(Degraded)

		r2.Unregister()

		assert.Equal(t, Degraded, r.Status())
		assert.Equal(t, "degraded", r.StatusString())
	})
}
