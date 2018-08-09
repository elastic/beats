// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type simpleSourceAcker struct {
	c int
}

func (s *simpleSourceAcker) AckEvents(data []interface{}) {
	s.c = len(data)
}

func TestAckMultiplexer(t *testing.T) {
	t.Run("when acker exists", func(t *testing.T) {
		a := NewAckMultiplexer()
		s := &simpleSourceAcker{}
		a.AddAcker(s)
		a.AckEvents([]interface{}{SourceMetadata{Acker: s}})
		assert.Equal(t, 1, s.c)
	})

	t.Run("when acker don't exist", func(t *testing.T) {
		a := NewAckMultiplexer()
		s := &simpleSourceAcker{}
		a.AddAcker(s)
		a.AckEvents([]interface{}{SourceMetadata{Acker: &simpleSourceAcker{}}})
		assert.Equal(t, 0, s.c)
	})

	t.Run("when multiple simultanous ackers exists", func(t *testing.T) {
		a := NewAckMultiplexer()
		s1 := &simpleSourceAcker{}
		a.AddAcker(s1)

		s2 := &simpleSourceAcker{}
		a.AddAcker(s2)

		a.AckEvents([]interface{}{
			SourceMetadata{Acker: s1},
			SourceMetadata{Acker: s2},
			SourceMetadata{Acker: s1},
			SourceMetadata{Acker: s2},
			SourceMetadata{Acker: s2},
		})
		assert.Equal(t, 2, s1.c)
		assert.Equal(t, 3, s2.c)
	})

	t.Run("when acker exists will stop receiving events when removed", func(t *testing.T) {
		a := NewAckMultiplexer()
		s := &simpleSourceAcker{}
		a.AddAcker(s)
		a.AckEvents([]interface{}{SourceMetadata{Acker: s}})
		assert.Equal(t, 1, s.c)
		a.RemoveAcker(s)
		a.AckEvents([]interface{}{SourceMetadata{Acker: s}})
		assert.Equal(t, 1, s.c)
	})
}
