// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilityManager(t *testing.T) {

	t.Run("filter", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.False(t, blocked, "not expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		val, found := newMap["key"]
		assert.True(t, found, "filter filters additional keys")
		assert.Equal(t, "val", val, "filter modifies additional keys")
	})

	t.Run("filter before block", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
				blockCap{},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.True(t, blocked, "expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		val, found := newMap["key"]
		assert.True(t, found, "filter filters additional keys")
		assert.Equal(t, "val", val, "filter modifies additional keys")
	})

	t.Run("filter after block", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
				blockCap{},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.True(t, blocked, "expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		val, found := newMap["key"]
		assert.True(t, found, "filter filters additional keys")
		assert.Equal(t, "val", val, "filter modifies additional keys")
	})

	t.Run("filter before keep", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
				keepAsIsCap{},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.False(t, blocked, "not expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		val, found := newMap["key"]
		assert.True(t, found, "filter filters additional keys")
		assert.Equal(t, "val", val, "filter modifies additional keys")
	})

	t.Run("filter after keep", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
				keepAsIsCap{},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.False(t, blocked, "not expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		val, found := newMap["key"]
		assert.True(t, found, "filter filters additional keys")
		assert.Equal(t, "val", val, "filter modifies additional keys")
	})

	t.Run("filter before filter", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
				filterKeywordCap{keyWord: "key"},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.False(t, blocked, "not expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		_, found = newMap["key"]
		assert.False(t, found, "filter filters additional keys")
	})
	t.Run("filter after filter", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "key"},
				filterKeywordCap{keyWord: "filter"},
			},
		}

		blocked, newIn := mgr.Apply(m)
		assert.False(t, blocked, "not expecting to block")

		newMap, ok := newIn.(map[string]string)
		assert.True(t, ok, "new input is not a map")

		_, found := newMap["filter"]
		assert.False(t, found, "filter does not filter keyword")

		_, found = newMap["key"]
		assert.False(t, found, "filter filters additional keys")
	})
}

type keepAsIsCap struct{}

func (keepAsIsCap) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

type blockCap struct{}

func (blockCap) Apply(in interface{}) (bool, interface{}) {
	return true, in
}

type filterKeywordCap struct {
	keyWord string
}

func (f filterKeywordCap) Apply(in interface{}) (bool, interface{}) {
	mm, ok := in.(map[string]string)
	if !ok {
		return false, in
	}

	delete(mm, f.keyWord)
	return false, mm
}

func getConfig() map[string]string {
	return map[string]string{
		"filter": "f_val",
		"key":    "val",
	}
}
