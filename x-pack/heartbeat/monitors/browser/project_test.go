// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin
// +build linux darwin

package browser

import (
	"encoding/json"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/synthexec"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestValidLocal(t *testing.T) {
	timeout := 30
	_, filename, _, _ := runtime.Caller(0)
	path := path.Join(filepath.Dir(filename), "source/fixtures/todos")
	testParams := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"name":   "My Name",
		"id":     "myId",
		"params": testParams,
		"filter_journeys": synthexec.FilterJourneyConfig{
			Tags:  []string{"*"},
			Match: "*",
		},
		"source": mapstr.M{
			"local": mapstr.M{
				"path": path,
			},
		},
		"timeout": timeout,
	})
	s, e := NewProject(cfg)
	require.NoError(t, e)
	require.NotNil(t, s)

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
	timeout := 30
	script := "a script"
	testParams := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"name":   "My Name",
		"id":     "myId",
		"params": testParams,
		"source": mapstr.M{
			"inline": mapstr.M{
				"script": script,
			},
		},
		"timeout": timeout,
	})
	s, e := NewProject(cfg)
	require.NoError(t, e)
	require.NotNil(t, s)
	require.Equal(t, script, s.projectCfg.Source.Inline.Script)
	require.Equal(t, "", s.Workdir())
	require.Equal(t, testParams, s.Params())

	e = s.Close()
	require.NoError(t, e)
}

func TestNameRequired(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"id": "myId",
		"source": mapstr.M{
			"inline": mapstr.M{
				"script": "a script",
			},
		},
	})
	_, e := NewProject(cfg)
	require.Regexp(t, ErrNameRequired, e)
}

func TestIDRequired(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"name": "My Name",
		"source": mapstr.M{
			"inline": mapstr.M{
				"script": "a script",
			},
		},
	})
	_, e := NewProject(cfg)
	require.Regexp(t, ErrIdRequired, e)
}

func TestEmptySource(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"source": mapstr.M{},
	})
	s, e := NewProject(cfg)

	require.Regexp(t, ErrBadConfig(source.ErrInvalidSource), e)
	require.Nil(t, s)
}

func TestExtraArgs(t *testing.T) {
	playWrightOpts := map[string]interface{}{
		"simpleOption": "simpleValue",
		"extraHTTPHeaders": map[string]interface{}{
			"foo": "bar",
		},
	}
	playwrightOptsJsonBytes, err := json.Marshal(playWrightOpts)
	require.NoError(t, err)
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
			"override throttling - text format",
			&Config{Throttling: "10d/3u/20l"},
			[]string{"--throttling", "10d/3u/20l"},
		},
		{
			"override throttling - JSON format",
			&Config{Throttling: map[string]interface{}{
				"download": 10,
				"upload":   3,
				"latency":  20,
			}},
			[]string{"--throttling", `{"download":10,"latency":20,"upload":3}`},
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
			"playwright options",
			&Config{
				PlaywrightOpts: playWrightOpts,
			},
			[]string{"--playwright-options", string(playwrightOptsJsonBytes)},
		},
		{
			"kitchen sink",
			&Config{SyntheticsArgs: []string{"--capability", "trace", "ssblocks"}, Sandbox: true},
			[]string{"--capability", "trace", "ssblocks", "--sandbox"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Project{
				projectCfg: tt.cfg,
			}
			if got := s.extraArgs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Project.extraArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmptyTimeout(t *testing.T) {
	defaults := DefaultConfig()
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"name": "My Name",
		"id":   "myId",
		"source": mapstr.M{
			"inline": mapstr.M{
				"script": "script",
			},
		},
	})
	s, e := NewProject(cfg)

	require.NoError(t, e)
	require.NotNil(t, s)
	require.Equal(t, s.projectCfg.Timeout, defaults.Timeout)
}
