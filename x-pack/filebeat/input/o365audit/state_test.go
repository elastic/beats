// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoopState(t *testing.T) {
	const (
		ct = "content-type"
		tn = "my_tenant"
	)
	myStream := stream{tn, ct}
	t.Run("new state", func(t *testing.T) {
		st := newStateStorage(noopPersister{})
		cur, err := st.Load(myStream)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		empty := newCursor(myStream, time.Time{})
		assert.Equal(t, empty, cur)
	})
	t.Run("update state", func(t *testing.T) {
		st := newStateStorage(noopPersister{})
		cur, err := st.Load(myStream)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		advanced := cur.TryAdvance(content{
			Type:       tn,
			ID:         "1234",
			URI:        "http://localhost.test/my_uri",
			Created:    time.Now(),
			Expiration: time.Now().Add(time.Hour),
		})
		assert.True(t, advanced)
		err = st.Save(cur)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		saved, err := st.Load(myStream)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, cur, saved)
	})
	t.Run("forbid reversal", func(t *testing.T) {
		st := newStateStorage(noopPersister{})
		cur := newCursor(myStream, time.Now())
		next := cur.ForNextLine()
		err := st.Save(next)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		err = st.Save(cur)
		assert.Equal(t, errNoUpdate, err)
	})
	t.Run("multiple contexts", func(t *testing.T) {
		st := newStateStorage(noopPersister{})
		cursors := []cursor{
			newCursor(myStream, time.Time{}),
			newCursor(stream{"tenant2", ct}, time.Time{}),
			newCursor(stream{ct, "bananas"}, time.Time{}),
		}
		for idx, cur := range cursors {
			msg := fmt.Sprintf("idx:%d cur:%+v", idx, cur)
			err := st.Save(cur)
			if !assert.NoError(t, err, msg) {
				t.Fatal(err)
			}
		}
		for idx, cur := range cursors {
			msg := fmt.Sprintf("idx:%d cur:%+v", idx, cur)
			saved, err := st.Load(cur.stream)
			if !assert.NoError(t, err, msg) {
				t.Fatal(err)
			}
			assert.Equal(t, cur, saved)
		}
		for idx, cur := range cursors {
			cur = cur.ForNextLine()
			cursors[idx] = cur
			msg := fmt.Sprintf("idx:%d cur:%+v", idx, cur)
			err := st.Save(cur)
			if !assert.NoError(t, err, msg) {
				t.Fatal(err)
			}
		}
		for idx, cur := range cursors {
			msg := fmt.Sprintf("idx:%d cur:%+v", idx, cur)
			saved, err := st.Load(cur.stream)
			if !assert.NoError(t, err, msg) {
				t.Fatal(err)
			}
			assert.Equal(t, cur, saved)
		}
	})
}
