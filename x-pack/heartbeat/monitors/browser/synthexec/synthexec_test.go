// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
)

func TestLineToSynthEventFactory(t *testing.T) {
	testType := "mytype"
	testText := "sometext"
	f := lineToSynthEventFactory(testType)
	res, err := f([]byte(testText), testText)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, testType, res.Type)
	require.Equal(t, testText, res.Payload["message"])
	require.Greater(t, res.TimestampEpochMicros, float64(0))
}

func TestJsonToSynthEvent(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		synthEvent *SynthEvent
		wantErr    bool
	}{
		{
			name:       "an empty line",
			line:       "",
			synthEvent: nil,
		},
		{
			name:       "a blank line",
			line:       "   ",
			synthEvent: nil,
		},
		{
			name:       "an invalid line",
			line:       `{"foo": "bar"}"`,
			synthEvent: nil,
			wantErr:    true,
		},
		{
			name: "a valid line",
			line: `{"@timestamp":7165676811882692608,"type":"step/end","journey":{"name":"inline","id":"inline"},"step":{"name":"Go to home page","index":0,"status":"succeeded"},"payload":{"source":"async ({page, params}) => {await page.goto('http://www.elastic.co')}","duration_ms":3472,"url":"https://www.elastic.co/","status":"succeeded"},"url":"https://www.elastic.co/","package_version":"0.0.1"}`,
			synthEvent: &SynthEvent{
				TimestampEpochMicros: 7165676811882692608,
				Type:                 "step/end",
				Journey: &Journey{
					Name: "inline",
					Id:   "inline",
				},
				Step: &Step{
					Name:   "Go to home page",
					Index:  0,
					Status: "succeeded",
				},
				Payload: map[string]interface{}{
					"source":      "async ({page, params}) => {await page.goto('http://www.elastic.co')}",
					"duration_ms": float64(3472),
					"url":         "https://www.elastic.co/",
					"status":      "succeeded",
				},
				PackageVersion: "0.0.1",
				URL:            "https://www.elastic.co/",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := jsonToSynthEvent([]byte(tt.line), tt.line)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "for line %s", tt.line)
			}

			if diff := deep.Equal(gotRes, tt.synthEvent); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func TestRunCmd(t *testing.T) {
	cmd := exec.Command("go", "run", "./main.go")
	_, filename, _, _ := runtime.Caller(0)
	cmd.Dir = path.Join(filepath.Dir(filename), "testcmd")

	stdinStr := "MY_STDIN"

	mpx, err := runCmd(context.TODO(), cmd, &stdinStr, nil, FilterJourneyConfig{})
	require.NoError(t, err)

	var synthEvents []*SynthEvent
	timeout := time.NewTimer(time.Minute)
Loop:
	for {
		select {
		case se := <-mpx.SynthEvents():
			if se == nil {
				break Loop
			}
			synthEvents = append(synthEvents, se)
		case <-timeout.C:
			require.Fail(t, "timeout expired for testing runCmd!")
		}
	}

	eventsWithType := func(typ string) (matched []*SynthEvent) {
		for _, se := range synthEvents {
			if se.Type == typ {
				matched = append(matched, se)
			}
		}
		return
	}

	t.Run("has echo'd stdin to stdout", func(t *testing.T) {
		stdoutEvents := eventsWithType("stdout")
		require.Len(t, stdoutEvents, 1)
		require.Equal(t, stdinStr, stdoutEvents[0].Payload["message"])
	})
	t.Run("has echo'd two lines to stderr", func(t *testing.T) {
		stdoutEvents := eventsWithType("stderr")
		require.Len(t, stdoutEvents, 2)
		require.Equal(t, "Stderr 1", stdoutEvents[0].Payload["message"])
		require.Equal(t, "Stderr 2", stdoutEvents[1].Payload["message"])
	})
	t.Run("should have one event per line in sampleinput", func(t *testing.T) {
		// 27 lines are in sample.ndjson + 2 from stderr + 1 from stdout + 1 from the command exit
		expected := 28 + 2 + 1
		require.Len(t, synthEvents, expected)
	})

	expectedEventTypes := []string{
		"journey/start",
		"step/end",
		"journey/end",
		"cmd/status",
	}
	for _, typ := range expectedEventTypes {
		t.Run(fmt.Sprintf("Should have at least one event of type %s", typ), func(t *testing.T) {
			require.GreaterOrEqual(t, len(eventsWithType(typ)), 1)
		})
	}
}

func TestSuiteCommandFactory(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	origPath := path.Join(filepath.Dir(filename), "../source/fixtures/todos")
	suitePath, err := filepath.Abs(origPath)
	require.NoError(t, err)
	binPath := path.Join(suitePath, "node_modules/.bin/elastic-synthetics")

	tests := []struct {
		name      string
		suitePath string
		extraArgs []string
		want      []string
		wantErr   bool
	}{
		{
			"no args",
			suitePath,
			nil,
			[]string{binPath, suitePath},
			false,
		},
		{
			"with args",
			suitePath,
			[]string{"--capability", "foo", "bar", "--rich-events"},
			[]string{binPath, suitePath, "--capability", "foo", "bar", "--rich-events"},
			false,
		},
		{
			"no npm root",
			"/not/a/path/for/sure",
			nil,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := suiteCommandFactory(tt.suitePath, tt.extraArgs...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			cmd := factory()
			got := cmd.Args
			require.Equal(t, tt.want, got)
		})
	}
}
