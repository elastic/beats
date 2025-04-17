// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var _ inputcursor.Publisher = (*publisher)(nil)

type publisher struct {
	m sync.Mutex

	events  []beat.Event
	cursors []*time.Time
}

func (p *publisher) Publish(e beat.Event, cursor interface{}) error {
	p.m.Lock()
	defer p.m.Unlock()

	p.events = append(p.events, e)
	var c *time.Time
	if cursor != nil {
		cv, _ := cursor.(time.Time)
		c = &cv
	}
	p.cursors = append(p.cursors, c)
	return nil
}

func TestInput(t *testing.T) {
	archivePath, err := openArchive()
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(archivePath) })

	testCases := []struct {
		name                 string
		skip                 func(*testing.T) bool
		cfg                  config
		timeUntilClose       time.Duration
		assertFunc           func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time)
		expectedLogStreamCmd string
		expectedLogShowCmd   string
		expectedRunErrorMsg  string
	}{
		{
			name:                 "Default stream",
			cfg:                  config{},
			timeUntilClose:       time.Second,
			expectedLogStreamCmd: "/usr/bin/log stream --style ndjson",
			assertFunc: func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time) {
				require.NotEmpty(collect, events)
				require.NotEmpty(collect, cursors)
				assert.Equal(collect, len(events), len(cursors))
				lastEvent := events[len(events)-1]
				lastCursor := cursors[len(cursors)-1]
				assert.EqualValues(collect, &lastEvent.Timestamp, lastCursor)
			},
		},
		{
			name: "Archive not found",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: "notfound.logarchive",
				},
			},
			timeUntilClose:      time.Second,
			expectedLogShowCmd:  "/usr/bin/log show --style ndjson --archive notfound.logarchive",
			expectedRunErrorMsg: "\"/usr/bin/log show --style ndjson --archive notfound.logarchive\" exited with an error: exit status 64",
		},
		{
			name: "Archived file",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s", archivePath),
			assertFunc:         eventsAndCursorAssertN(462),
		},
		{
			name: "Trace file",
			cfg: config{
				ShowConfig: showConfig{
					TraceFile: path.Join(archivePath, "logdata.LiveData.tracev3"),
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --file %s", path.Join(archivePath, "logdata.LiveData.tracev3")),
			assertFunc:         eventsAndCursorAssertN(7),
		},
		{
			name: "With start date",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
					Start:       "2024-12-04 13:46:00+0200",
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --start 2024-12-04 13:46:00+0200", archivePath),
			assertFunc:         eventsAndCursorAssertN(314),
		},
		{
			name: "With start and end dates",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
					Start:       "2024-12-04 13:45:00+0200",
					End:         "2024-12-04 13:46:00+0200",
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --start 2024-12-04 13:45:00+0200 --end 2024-12-04 13:46:00+0200", archivePath),
			assertFunc:         eventsAndCursorAssertN(149),
		},
		{
			name: "With end date",
			skip: func(t *testing.T) bool {
				const sequoiaPrefix = "15."
				version, err := exec.Command("sw_vers", "-productVersion").CombinedOutput()
				if err != nil {
					t.Fatalf("failed to get macOS version: %v", err)
					return true
				}
				if strings.HasPrefix(strings.TrimSpace(string(version)), sequoiaPrefix) {
					t.Skip("macOS 15.x does not support the --end flag correctly")
					return true
				}
				return false
			},
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
					End:         "2024-12-04 13:46:00+0200",
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --end 2024-12-04 13:46:00+0200", archivePath),
			assertFunc:         eventsAndCursorAssertN(149),
		},
		{
			name: "With predicate",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
				},
				CommonConfig: commonConfig{
					Predicate: []string{
						`processImagePath == "/kernel"`,
					},
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --predicate processImagePath == \"/kernel\"", archivePath),
			assertFunc:         eventsAndCursorAssertN(460),
		},
		{
			name: "With process",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
				},
				CommonConfig: commonConfig{
					Process: []string{
						"0",
					},
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --process 0", archivePath),
			assertFunc:         eventsAndCursorAssertN(462),
		},
		{
			name: "With optional flags",
			cfg: config{
				ShowConfig: showConfig{
					ArchiveFile: archivePath,
				},
				CommonConfig: commonConfig{
					Info:               true,
					Debug:              true,
					Backtrace:          true,
					Signpost:           true,
					MachContinuousTime: true,
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --info --debug --backtrace --signpost --mach-continuous-time", archivePath),
			assertFunc:         eventsAndCursorAssertN(462),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip != nil && tc.skip(t) {
				return
			}
			_, cursorInput := newCursorInput(tc.cfg)
			input, _ := cursorInput.(*input)

			ctx, cancel := context.WithCancel(context.Background())

			pub := &publisher{}
			log, buf := logp.NewInMemory("unifiedlogs_test", logp.JSONEncoderConfig())

			var wg sync.WaitGroup
			wg.Add(1)
			go func(t *testing.T) {
				defer wg.Done()
				err := input.runWithMetrics(ctx, pub, testMetricsRegistry(), log)
				if tc.expectedRunErrorMsg == "" {
					assert.NoError(t, err)
				} else {
					assert.ErrorContains(t, err, tc.expectedRunErrorMsg)
				}
			}(t)

			select {
			case <-ctx.Done():
			case <-time.After(tc.timeUntilClose):
			}

			cancel()
			wg.Wait()

			assert.EventuallyWithT(t,
				func(collect *assert.CollectT) {
					assert.Equal(collect, tc.expectedLogStreamCmd, filterStartLogStreamLogline(buf.Bytes()))
					assert.Equal(collect, tc.expectedLogShowCmd, filterStartLogShowLogline(buf.Bytes()))
					if tc.assertFunc != nil {
						tc.assertFunc(collect, pub.events, pub.cursors)
					}
				},
				60*time.Second, time.Second,
			)
		})
	}
}

func TestBackfillAndStream(t *testing.T) {
	archivePath, err := openArchive()
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(archivePath) })

	cfg := config{
		Backfill: true,
		ShowConfig: showConfig{
			Start: time.Now().Add(-5 * time.Second).Format("2006-01-02 15:04:05"),
		},
		CommonConfig: commonConfig{
			Info:               true,
			Debug:              true,
			Backtrace:          true,
			Signpost:           true,
			MachContinuousTime: true,
		},
	}

	expectedLogShowCmd := fmt.Sprintf("/usr/bin/log show --style ndjson --info --debug --backtrace --signpost --mach-continuous-time --start %v", time.Now().Format("2006-01-02"))
	expectedLogStreamCmd := "/usr/bin/log stream --style ndjson --info --debug --backtrace --signpost --mach-continuous-time"

	_, cursorInput := newCursorInput(cfg)
	input := cursorInput.(*input)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	pub := &publisher{}
	log, buf := logp.NewInMemory("unifiedlogs_test", logp.JSONEncoderConfig())

	var wg sync.WaitGroup
	wg.Add(1)
	go func(t *testing.T) {
		defer wg.Done()
		err := input.runWithMetrics(ctx, pub, testMetricsRegistry(), log)
		assert.NoError(t, err)
	}(t)

	var firstStreamedEventTime *time.Time
	assert.EventuallyWithT(t,
		func(collect *assert.CollectT) {
			showCmdLog := filterStartLogShowLogline(buf.Bytes())
			assert.Equal(collect, expectedLogStreamCmd, filterStartLogStreamLogline(buf.Bytes()))
			assert.True(collect, strings.HasPrefix(showCmdLog, expectedLogShowCmd))
			require.NotEmpty(collect, pub.events)
			require.NotEmpty(collect, pub.cursors)

			var endTime time.Time
			regex := regexp.MustCompile(`--end\s+(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}[+-]\d{4})`)
			matches := regex.FindStringSubmatch(showCmdLog)
			require.Equal(collect, 2, len(matches))
			endTime, _ = time.Parse("2006-01-02 15:04:05-0700", matches[1])
			endTime = endTime.Truncate(time.Second)

			if firstStreamedEventTime == nil {
				for i := range pub.events {
					if pub.cursors[i] == nil {
						first := pub.events[i].Timestamp.Add(time.Second).Truncate(time.Second)
						firstStreamedEventTime = &first
						break
					}
				}
			}
			require.NotNil(collect, firstStreamedEventTime)
			assert.EqualValues(collect, endTime, *firstStreamedEventTime)
			assert.True(collect, strings.HasPrefix(showCmdLog, filterEndLogShowLogline(buf.Bytes())))
		},
		30*time.Second, time.Second,
	)

	cancel()
	wg.Wait()
}

const (
	cmdStartPrefix = "exec command start: "
	cmdEndPrefix   = "exec command end: "
)

func filterStartLogStreamLogline(buf []byte) string {
	const cmd = "/usr/bin/log stream"
	return filterLogCmdLine(buf, cmd, cmdStartPrefix)
}

func filterStartLogShowLogline(buf []byte) string {
	const cmd = "/usr/bin/log show"
	return filterLogCmdLine(buf, cmd, cmdStartPrefix)
}

func filterEndLogShowLogline(buf []byte) string {
	const cmd = "/usr/bin/log show"
	return filterLogCmdLine(buf, cmd, cmdEndPrefix)
}

func filterLogCmdLine(buf []byte, cmd, cmdPrefix string) string {
	scanner := bufio.NewScanner(bytes.NewBuffer(buf))
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(text, "\t")
		if len(parts) != 4 {
			continue
		}

		trimmed := strings.TrimPrefix(parts[3], cmdPrefix)
		if strings.HasPrefix(trimmed, cmd) {
			return trimmed
		}
	}
	return ""
}

func eventsAndCursorAssertN(n int) func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time) {
	return func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time) {
		assert.Equal(collect, n, len(events))
		assert.Equal(collect, n, len(cursors))
		lastEvent := events[len(events)-1]
		lastCursor := cursors[len(cursors)-1]
		assert.EqualValues(collect, &lastEvent.Timestamp, lastCursor)
	}
}

func openArchive() (string, error) {
	return extractTarGz(path.Join("testdata", "test.logarchive.tar.gz"))
}

func extractTarGz(tarGzPath string) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "extracted-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Use the 'tar' command to extract the .tar.gz file
	cmd := exec.Command("tar", "-xzf", tarGzPath, "-C", tempDir)

	// Run the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract .tar.gz: %w", err)
	}

	return path.Join(tempDir, "test.logarchive"), nil
}

func testMetricsRegistry() *monitoring.Registry {
	reg := inputmon.NewMetricsRegistry(
		"", "", monitoring.NewRegistry(), logp.NewLogger("test"))
	return reg
}
