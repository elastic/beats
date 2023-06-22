// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux

package browser

import (
	"encoding/json"
	"fmt"
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
	_, e := NewSourceJob(cfg)
	require.Error(t, e)
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
	s, e := NewSourceJob(cfg)
	require.NoError(t, e)
	require.NotNil(t, s)
	require.Equal(t, script, s.browserCfg.Source.Inline.Script)
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
	_, e := NewSourceJob(cfg)
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
	_, e := NewSourceJob(cfg)
	require.Regexp(t, ErrIdRequired, e)
}

func TestEmptySource(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"source": mapstr.M{},
	})
	s, e := NewSourceJob(cfg)

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
		ui   bool
	}{
		{
			"no args",
			&Config{},
			[]string{},
			false,
		},
		{
			"default",
			DefaultConfig(),
			[]string{"--screenshots", "on"},
			false,
		},
		{
			"sandbox",
			&Config{Sandbox: true},
			[]string{"--sandbox"},
			false,
		},
		{
			"throttling truthy",
			&Config{Throttling: true},
			[]string{},
			false,
		},
		{
			"disable throttling",
			&Config{Throttling: false},
			[]string{"--no-throttling"},
			false,
		},
		{
			"override throttling - text format",
			&Config{Throttling: "10d/3u/20l"},
			[]string{"--throttling", "10d/3u/20l"},
			false,
		},
		{
			"override throttling - JSON format",
			&Config{Throttling: map[string]interface{}{
				"download": 10,
				"upload":   3,
				"latency":  20,
			}},
			[]string{"--throttling", `{"download":10,"latency":20,"upload":3}`},
			false,
		},
		{
			"ignore_https_errors",
			&Config{IgnoreHTTPSErrors: true},
			[]string{"--ignore-https-errors"},
			false,
		},
		{
			"screenshots",
			&Config{Screenshots: "off"},
			[]string{"--screenshots", "off"},
			false,
		},
		{
			"capabilities",
			&Config{SyntheticsArgs: []string{"--capability", "trace", "ssblocks"}},
			[]string{"--capability", "trace", "ssblocks"},
			false,
		},
		{
			"playwright options",
			&Config{
				PlaywrightOpts: playWrightOpts,
			},
			[]string{"--playwright-options", string(playwrightOptsJsonBytes)},
			false,
		},
		{
			"kitchen sink",
			&Config{SyntheticsArgs: []string{"--capability", "trace", "ssblocks"}, Sandbox: true},
			[]string{"--capability", "trace", "ssblocks", "--sandbox"},
			false,
		},
		{
			"does not filter dev flags on non-ui origin",
			&Config{SyntheticsArgs: []string{"--pause-on-error", "--dry-run", "--quiet-exit-code", "--outfd", "testfd"}, Sandbox: true},
			[]string{"--pause-on-error", "--dry-run", "--quiet-exit-code", "--outfd", "testfd", "--sandbox"},
			false,
		},
		{
			"filters dev flags on ui origin",
			&Config{SyntheticsArgs: []string{"--pause-on-error", "--dry-run", "--quiet-exit-code", "--outfd", "testfd"}, Sandbox: true},
			[]string{"--sandbox"},
			true,
		},
		{
			"filters variadic dev flags on ui origin",
			&Config{SyntheticsArgs: []string{"--tags", "tag1", "tag2", "tag3", "--match", "tag4", "tag5", "--sandbox", "-r", "require1", "require2", "--require", "require3", "require4", "require5"}},
			[]string{"--sandbox"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SourceJob{
				browserCfg: tt.cfg,
			}
			if got := s.extraArgs(tt.ui); !reflect.DeepEqual(got, tt.want) {
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
	s, e := NewSourceJob(cfg)

	require.NoError(t, e)
	require.NotNil(t, s)
	require.Equal(t, s.browserCfg.Timeout, defaults.Timeout)
}

func TestFilterDevFlags(t *testing.T) {
	allFlags := []string{}
	for k := range filterMap {
		allFlags = append(allFlags, k)
	}

	variadicGen := func(flag string, n int) []string {
		params := []string{"dummy"}
		params = append(params, flag)
		for i := 0; i < n; i++ {
			params = append(params, fmt.Sprintf("flag-%d", i))
		}

		return params
	}
	tests := []struct {
		name           string
		syntheticsArgs []string
		want           []string
	}{
		{
			"no args",
			nil,
			[]string{},
		},
		{
			"no args",
			[]string{},
			[]string{},
		},
		{
			"all filtered",
			allFlags,
			[]string{},
		},
		{
			"keep unfiltered",
			append([]string{"unfiltered"}, allFlags...),
			[]string{"unfiltered"},
		},
		{
			"filter associated params",
			[]string{"--help", "malformed1", "--outfd", "param1", "malformed2", "--reporter", "-malformed3"},
			[]string{"malformed1", "malformed2", "-malformed3"},
		},
		{
			"filter variadic flags - tags - 10",
			variadicGen("--tags", 10),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - tags - 50",
			variadicGen("--tags", 50),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - tags - 100",
			variadicGen("--tags", 100),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - require - 10",
			variadicGen("--require", 10),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - require - 50",
			variadicGen("--require", 50),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - require - 100",
			variadicGen("-r", 100),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - match - 10",
			variadicGen("--match", 10),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - match - 50",
			variadicGen("--match", 50),
			[]string{"dummy"},
		},
		{
			"filter variadic flags - match - 100",
			variadicGen("--match", 100),
			[]string{"dummy"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterDevFlags(tt.syntheticsArgs, filterMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("syntheticsArgs = %v, want %v", got, tt.want)
			}
		})
	}
}
