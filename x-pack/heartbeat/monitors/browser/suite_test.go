// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/synthexec"
)

func TestValidLocal(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	path := path.Join(filepath.Dir(filename), "source/fixtures/todos")
	testParams := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	cfg := common.MustNewConfigFrom(common.MapStr{
		"name":   "My Name",
		"id":     "myId",
		"params": testParams,
		"filter_journeys": synthexec.FilterJourneyConfig{
			Tags:  []string{"*"},
			Match: "*",
		},
		"source": common.MapStr{
			"local": common.MapStr{
				"path": path,
			},
		},
	})
	s, e := NewSuite(cfg)
	require.NoError(t, e)
	require.NotNil(t, s)
	_, ok := s.InlineSource()
	require.False(t, ok)

	source.GoOffline()
	defer source.GoOnline()
	require.NoError(t, s.Fetch())
	defer require.NoError(t, s.Close())
	require.Regexp(t, "\\w{1,}", s.Workdir())
	require.Equal(t, testParams, s.Params())

	e = s.Close()
	require.NoError(t, e)
}

func TestValidInline(t *testing.T) {
	script := "a script"
	testParams := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	cfg := common.MustNewConfigFrom(common.MapStr{
		"name":   "My Name",
		"id":     "myId",
		"params": testParams,
		"source": common.MapStr{
			"inline": common.MapStr{
				"script": script,
			},
		},
	})
	s, e := NewSuite(cfg)
	require.NoError(t, e)
	require.NotNil(t, s)
	sSrc, ok := s.InlineSource()
	require.True(t, ok)
	require.Equal(t, script, sSrc)
	require.Equal(t, "", s.Workdir())
	require.Equal(t, testParams, s.Params())

	e = s.Close()
	require.NoError(t, e)
}

func TestNameRequired(t *testing.T) {
	cfg := common.MustNewConfigFrom(common.MapStr{
		"id": "myId",
		"source": common.MapStr{
			"inline": common.MapStr{
				"script": "a script",
			},
		},
	})
	_, e := NewSuite(cfg)
	require.Regexp(t, ErrNameRequired, e)
}

func TestIDRequired(t *testing.T) {
	cfg := common.MustNewConfigFrom(common.MapStr{
		"name": "My Name",
		"source": common.MapStr{
			"inline": common.MapStr{
				"script": "a script",
			},
		},
	})
	_, e := NewSuite(cfg)
	require.Regexp(t, ErrIdRequired, e)
}

func TestEmptySource(t *testing.T) {
	cfg := common.MustNewConfigFrom(common.MapStr{
		"source": common.MapStr{},
	})
	s, e := NewSuite(cfg)

	require.Regexp(t, ErrBadConfig(source.ErrInvalidSource), e)
	require.Nil(t, s)
}

func TestExtraArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want []string
	}{
		{
			"no args",
			&Config{},
			nil,
		},
		{
			"default",
			DefaultConfig(),
			[]string{"--screenshots", "on"},
		},
		{
			"sandbox",
			&Config{Sandbox: true},
			[]string{"--sandbox"},
		},
		{
			"throttling truthy",
			&Config{Throttling: true},
			nil,
		},
		{
			"disable throttling",
			&Config{Throttling: false},
			[]string{"--no-throttling"},
		},
		{
			"override throttling",
			&Config{Throttling: "10d/3u/20l"},
			[]string{"--throttling", "10d/3u/20l"},
		},
		{
			"ignore_https_errors",
			&Config{IgnoreHTTPSErrors: true},
			[]string{"--ignore-https-errors"},
		},
		{
			"screenshots",
			&Config{Screenshots: "off"},
			[]string{"--screenshots", "off"},
		},
		{
			"capabilities",
			&Config{SyntheticsArgs: []string{"--capability", "trace", "ssblocks"}},
			[]string{"--capability", "trace", "ssblocks"},
		},
		{
			"kitchen sink",
			&Config{SyntheticsArgs: []string{"--capability", "trace", "ssblocks"}, Sandbox: true},
			[]string{"--capability", "trace", "ssblocks", "--sandbox"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Suite{
				suiteCfg: tt.cfg,
			}
			if got := s.extraArgs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Suite.extraArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
