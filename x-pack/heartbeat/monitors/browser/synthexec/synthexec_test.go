// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || synthetics

package synthexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
					ID:   "inline",
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

			if diff := cmp.Diff(gotRes, tt.synthEvent, cmpopts.IgnoreUnexported(SynthEvent{})); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func goCmd(args ...string) *exec.Cmd {
	goBinary := "go" // relative by default
	// GET the GOROOT if defined, this helps in scenarios where
	// GOROOT is defined, but GOROOT/bin is not in the path
	// This can happen when targeting WSL from intellij running on windows
	goRoot := os.Getenv("GOROOT")
	if goRoot != "" {
		goBinary = filepath.Join(goRoot, "bin", "go")
	}
	return exec.Command(goBinary, args...)
}

func TestRunCmd(t *testing.T) {
	cmd := goCmd("run", "./main.go")

	stdinStr := "MY_STDIN"
	synthEvents := runAndCollect(t, cmd, stdinStr, 15*time.Minute)

	t.Run("has echo'd stdin to stdout", func(t *testing.T) {
		stdoutEvents := eventsWithType(Stdout, synthEvents)
		require.Len(t, stdoutEvents, 1)
		require.Equal(t, stdinStr, stdoutEvents[0].Payload["message"])
	})
	t.Run("has echo'd two lines to stderr", func(t *testing.T) {
		stdoutEvents := eventsWithType("stderr", synthEvents)
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
		JourneyStart,
		StepEnd,
		JourneyEnd,
		CmdStatus,
	}
	for _, typ := range expectedEventTypes {
		t.Run(fmt.Sprintf("Should have at least one event of type %s", typ), func(t *testing.T) {
			require.GreaterOrEqual(t, len(eventsWithType(typ, synthEvents)), 1)
		})
	}
}

func TestRunBadExitCodeCmd(t *testing.T) {
	cmd := goCmd("run", "./main.go", "exit")
	synthEvents := runAndCollect(t, cmd, "", 15*time.Minute)

	// go run outputs "exit status 123" to stderr so we have two messages
	require.Len(t, synthEvents, 2)

	t.Run("has a stderr line", func(t *testing.T) {
		stderrEvents := eventsWithType(Stderr, synthEvents)
		require.Len(t, stderrEvents, 1)
		require.Equal(t, "exit status 123", stderrEvents[0].Payload["message"])
	})
	t.Run("has a cmd status event", func(t *testing.T) {
		stdoutEvents := eventsWithType(CmdStatus, synthEvents)
		require.Len(t, stdoutEvents, 1)
	})
}

func TestRunTimeoutExitCodeCmd(t *testing.T) {
	cmd := goCmd("run", "./main.go")
	synthEvents := runAndCollect(t, cmd, "", 0*time.Second)

	// go run should not produce any additional stderr output in this case
	require.Len(t, synthEvents, 1)

	t.Run("has a cmd status event", func(t *testing.T) {
		stdoutEvents := eventsWithType(CmdStatus, synthEvents)
		require.Len(t, stdoutEvents, 1)
		require.Equal(t, synthEvents[0].Error.Code, "CMD_TIMEOUT")
	})
}

func runAndCollect(t *testing.T, cmd *exec.Cmd, stdinStr string, cmdTimeout time.Duration) []*SynthEvent {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	cmd.Dir = filepath.Join(cwd, "testcmd")
	ctx := context.WithValue(context.TODO(), SynthexecTimeout, cmdTimeout)

	mpx, err := runCmd(ctx, &SynthCmd{cmd}, &stdinStr, nil, FilterJourneyConfig{})
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

	return synthEvents
}

func eventsWithType(typ string, synthEvents []*SynthEvent) (matched []*SynthEvent) {
	for _, se := range synthEvents {
		if se.Type == typ {
			matched = append(matched, se)
		}
	}
	return matched
}

func TestProjectCommandFactory(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	origPath := filepath.Join(filepath.Dir(filename), "..", "source", "fixtures", "todos")
	projectPath, err := filepath.Abs(origPath)
	require.NoError(t, err)
	binPath := filepath.Join(projectPath, "node_modules", ".bin", "elastic-synthetics")

	tests := []struct {
		name        string
		projectPath string
		extraArgs   []string
		want        []string
		wantErr     bool
	}{
		{
			"no args",
			projectPath,
			nil,
			[]string{binPath, projectPath},
			false,
		},
		{
			"with args",
			projectPath,
			[]string{"--capability", "foo", "bar", "--rich-events"},
			[]string{binPath, projectPath, "--capability", "foo", "bar", "--rich-events"},
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
			factory, err := projectCommandFactory(tt.projectPath, tt.extraArgs...)

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
