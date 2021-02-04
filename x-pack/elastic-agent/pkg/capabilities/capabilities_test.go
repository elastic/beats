// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCapabilities(t *testing.T) {
	testCases := []string{
		"filter_metrics",
		"allow_metrics",
		"deny_logs",
		"no_caps",
	}

	l, _ := logger.New("test")

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			filename := filepath.Join("testdata", fmt.Sprintf("%s-capabilities.yml", tc))
			controller := status.NewController(l)
			caps, err := Load(filename, l, controller)
			assert.NoError(t, err)
			assert.NotNil(t, caps)

			cfgFile := filepath.Join("testdata", fmt.Sprintf("%s-config.yml", tc))
			configFile, err := os.Open(cfgFile)
			if err != nil {
				assert.Fail(t, err.Error())
				return
			}
			defer configFile.Close()

			cfg, err := config.NewConfigFrom(configFile)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			mm, err := cfg.ToMapStr()
			assert.NoError(t, err)
			assert.NotNil(t, mm)

			isBlocked, out := caps.Apply(mm)
			assert.False(t, isBlocked)

			resultConfig, ok := out.(map[string]interface{})
			assert.True(t, ok)

			resultFileName := filepath.Join("testdata", fmt.Sprintf("%s-result.yml", tc))
			configResultFile, err := os.Open(resultFileName)
			if err != nil {
				assert.Fail(t, err.Error())
				return
			}
			defer configFile.Close()

			expectedConfig, err := config.NewConfigFrom(configResultFile)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			expectedMap, err := expectedConfig.ToMapStr()
			fixInputsType(expectedMap)
			fixInputsType(resultConfig)

			if !assert.True(t, cmp.Equal(expectedMap, resultConfig)) {
				diff := cmp.Diff(expectedMap, resultConfig)
				if diff != "" {
					t.Errorf("%s mismatch (-want +got):\n%s", tc, diff)
				}
			}
		})
	}
}

func fixInputsType(mm map[string]interface{}) {
	if i, found := mm[inputsKey]; found {
		var inputs []interface{}

		if im, ok := i.([]map[string]interface{}); ok {
			for _, val := range im {
				inputs = append(inputs, val)
			}
		} else if im, ok := i.([]interface{}); ok {
			inputs = im
		}
		mm[inputsKey] = inputs
	}
}

func TestCapabilityManager(t *testing.T) {
	l := newErrorLogger(t)

	t.Run("filter", func(t *testing.T) {
		m := getConfig()
		mgr := &capabilitiesManager{
			caps: []Capability{
				filterKeywordCap{keyWord: "filter"},
			},
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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
			reporter: status.NewController(l).Register("test"),
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

func newErrorLogger(t *testing.T) *logger.Logger {
	t.Helper()

	loggerCfg := logger.DefaultLoggingConfig()
	loggerCfg.Level = logp.ErrorLevel

	log, err := logger.NewFromConfig("", loggerCfg)
	require.NoError(t, err)
	return log
}
