// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration
// +build !integration

package log

import (
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestInputFileExclude(t *testing.T) {
	p := Input{
		config: config{
			ExcludeFiles: []match.Matcher{match.MustCompile(`\.gz$`)},
		},
	}

	assert.True(t, p.isFileExcluded("/tmp/log/logw.gz"))
	assert.False(t, p.isFileExcluded("/tmp/log/logw.log"))
}

var cleanInactiveTests = []struct {
	cleanInactive time.Duration
	fileTime      time.Time
	result        bool
}{
	{
		cleanInactive: 0,
		fileTime:      time.Now(),
		result:        false,
	},
	{
		cleanInactive: 1 * time.Second,
		fileTime:      time.Now().Add(-5 * time.Second),
		result:        true,
	},
	{
		cleanInactive: 10 * time.Second,
		fileTime:      time.Now().Add(-5 * time.Second),
		result:        false,
	},
}

func TestIsCleanInactive(t *testing.T) {
	for _, test := range cleanInactiveTests {

		l := Input{
			config: config{
				CleanInactive: test.cleanInactive,
			},
		}
		state := file.State{
			Fileinfo: TestFileInfo{
				time: test.fileTime,
			},
		}

		assert.Equal(t, test.result, l.isCleanInactive(state))
	}
}

func TestInputLifecycle(t *testing.T) {
	cases := []struct {
		title  string
		closer func(input.Context, *Input)
	}{
		{
			title: "explicitly closed",
			closer: func(_ input.Context, input *Input) {
				input.Wait()
			},
		},
		{
			title: "context done",
			closer: func(ctx input.Context, _ *Input) {
				close(ctx.Done)
			},
		},
		{
			title: "beat context done",
			closer: func(ctx input.Context, _ *Input) {
				close(ctx.Done)
				close(ctx.BeatDone)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			context := input.Context{
				Done:     make(chan struct{}),
				BeatDone: make(chan struct{}),
			}
			testInputLifecycle(t, context, c.closer)
		})
	}
}

// TestInputLifecycle performs blackbock testing of the log input
func testInputLifecycle(t *testing.T, context input.Context, closer func(input.Context, *Input)) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Prepare a log file
	tmpdir, err := ioutil.TempDir(os.TempDir(), "input-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	logs := []byte("some log line\nother log line\n")
	err = ioutil.WriteFile(path.Join(tmpdir, "some.log"), logs, 0o644)
	assert.NoError(t, err)

	// Setup the input
	config, _ := common.NewConfigFrom(mapstr.M{
		"paths":     path.Join(tmpdir, "*.log"),
		"close_eof": true,
	})

	events := make(chan beat.Event, 100)
	defer close(events)
	capturer := NewEventCapturer(events)
	defer capturer.Close()
	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(capturer), nil
	})

	input, err := NewInput(config, connector, context)
	if err != nil {
		t.Error(err)
		return
	}

	// Run the input and wait for finalization
	input.Run()

	timeout := time.After(30 * time.Second)
	done := make(chan struct{})
	for {
		select {
		case event := <-events:
			if state, ok := event.Private.(file.State); ok && state.Finished {
				assert.Equal(t, len(logs), int(state.Offset), "file has not been fully read")
				go func() {
					closer(context, input.(*Input))
					close(done)
				}()
			}
		case <-done:
			return
		case <-timeout:
			t.Fatal("timeout waiting for closed state")
		}
	}
}

func TestNewInputDone(t *testing.T) {
	config := mapstr.M{
		"paths": path.Join(os.TempDir(), "logs", "*.log"),
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}

func TestNewInputError(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	config := common.NewConfig()

	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return inputtest.Outlet{}, nil
	})

	context := input.Context{}

	_, err := NewInput(config, connector, context)
	assert.Error(t, err)
}

func TestMatchesMeta(t *testing.T) {
	tests := []struct {
		Input  *Input
		Meta   map[string]string
		Result bool
	}{
		{
			Input: &Input{
				meta: map[string]string{
					"it": "matches",
				},
			},
			Meta: map[string]string{
				"it": "matches",
			},
			Result: true,
		},
		{
			Input: &Input{
				meta: map[string]string{
					"it":     "doesnt",
					"doesnt": "match",
				},
			},
			Meta: map[string]string{
				"it": "doesnt",
			},
			Result: false,
		},
		{
			Input: &Input{
				meta: map[string]string{
					"it": "doesnt",
				},
			},
			Meta: map[string]string{
				"it":     "doesnt",
				"doesnt": "match",
			},
			Result: false,
		},
		{
			Input: &Input{
				meta: map[string]string{},
			},
			Meta:   map[string]string{},
			Result: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Result, test.Input.matchesMeta(test.Meta))
	}
}

type TestFileInfo struct {
	time time.Time
}

func (t TestFileInfo) Name() string       { return "" }
func (t TestFileInfo) Size() int64        { return 0 }
func (t TestFileInfo) Mode() os.FileMode  { return 0 }
func (t TestFileInfo) ModTime() time.Time { return t.time }
func (t TestFileInfo) IsDir() bool        { return false }
func (t TestFileInfo) Sys() interface{}   { return nil }

type eventCapturer struct {
	closed    bool
	c         chan struct{}
	closeOnce sync.Once
	events    chan beat.Event
}

func NewEventCapturer(events chan beat.Event) channel.Outleter {
	return &eventCapturer{
		c:      make(chan struct{}),
		events: events,
	}
}

func (o *eventCapturer) OnEvent(event beat.Event) bool {
	o.events <- event
	return true
}

func (o *eventCapturer) Close() error {
	o.closeOnce.Do(func() {
		o.closed = true
		close(o.c)
	})
	return nil
}

func (o *eventCapturer) Done() <-chan struct{} {
	return o.c
}
