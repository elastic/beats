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
	"github.com/elastic/elastic-agent-libs/logp"
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
		cv := cursor.(time.Time)
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
		cfg                  config
		timeUntilClose       time.Duration
		assertFunc           func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time, cmd ...string)
		expectedLogStreamCmd string
		expectedLogShowCmd   string
		expectedRunErrorMsg  string
	}{
		{
			name:                 "Default stream",
			cfg:                  config{},
			timeUntilClose:       time.Second,
			expectedLogStreamCmd: "/usr/bin/log stream --style ndjson",
			assertFunc: func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time, cmd ...string) {
				assert.NotEmpty(collect, events)
				assert.NotEmpty(collect, cursors)
				assert.Equal(collect, len(events), len(cursors))
				lastEvent := events[len(events)-1]
				lastCursor := cursors[len(cursors)-1]
				assert.EqualValues(collect, &lastEvent.Timestamp, lastCursor)
			},
		},
		{
			name: "Archive not found",
			cfg: config{
				showConfig: showConfig{
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
				showConfig: showConfig{
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
				showConfig: showConfig{
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
				showConfig: showConfig{
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
				showConfig: showConfig{
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
			cfg: config{
				showConfig: showConfig{
					ArchiveFile: archivePath,
					End:         "2024-12-04 13:46:00+0200",
				},
			},
			timeUntilClose:     time.Second,
			expectedLogShowCmd: fmt.Sprintf("/usr/bin/log show --style ndjson --archive %s --end 2024-12-04 13:46:00+0200", archivePath),
			assertFunc:         eventsAndCursorAssertN(462),
		},
		{
			name: "With predicate",
			cfg: config{
				showConfig: showConfig{
					ArchiveFile: archivePath,
				},
				commonConfig: commonConfig{
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
				showConfig: showConfig{
					ArchiveFile: archivePath,
				},
				commonConfig: commonConfig{
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
				showConfig: showConfig{
					ArchiveFile: archivePath,
				},
				commonConfig: commonConfig{
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
		{
			name: "Stream and Backfill",
			cfg: config{
				Backfill: true,
				showConfig: showConfig{
					Start: time.Now().Add(-5 * time.Second).Format("2006-01-02 15:04:05"),
				},
				commonConfig: commonConfig{
					Info:               true,
					Debug:              true,
					Backtrace:          true,
					Signpost:           true,
					MachContinuousTime: true,
				},
			},
			timeUntilClose:       2 * time.Second,
			expectedLogShowCmd:   fmt.Sprintf("/usr/bin/log show --style ndjson --info --debug --backtrace --signpost --mach-continuous-time --start %v", time.Now().Format("2006-01-02")),
			expectedLogStreamCmd: "/usr/bin/log stream --style ndjson --info --debug --backtrace --signpost --mach-continuous-time",
			assertFunc: func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time, cmd ...string) {
				assert.Less(collect, 0, len(events))
				assert.Less(collect, 0, len(cursors))

				var endTime time.Time
				regex := regexp.MustCompile(`--end\s+(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}[+-]\d{4})`)
				if len(cmd) > 0 {
					matches := regex.FindStringSubmatch(cmd[0])
					assert.Equal(collect, 2, len(matches))
					endTime, _ = time.Parse("2006-01-02 15:04:05-0700", matches[1])
				}
				endTime = endTime.Truncate(time.Second)

				for i := range events {
					if cursors[i] == nil {
						firstStreamedEventTime := events[i].Timestamp
						firstStreamedEventTime = firstStreamedEventTime.Add(time.Second).Truncate(time.Second)
						assert.Equal(collect, endTime, firstStreamedEventTime)
						break
					}
				}

			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, cursorInput := newCursorInput(tc.cfg)
			input := cursorInput.(*input)

			ctx, cancel := context.WithCancel(context.Background())

			pub := &publisher{}
			log, buf := logp.NewInMemory("unifiedlogs_test", logp.JSONEncoderConfig())

			var wg sync.WaitGroup
			wg.Add(1)
			go func(t *testing.T) {
				defer wg.Done()
				err := input.runWithMetrics(ctx, pub, log)
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
					assert.Equal(collect, tc.expectedLogStreamCmd, filterLogStreamLogline(buf.Bytes()))
					assert.Equal(collect, true, strings.HasPrefix(filterLogShowLogline(buf.Bytes()), tc.expectedLogShowCmd))
					if tc.assertFunc != nil {
						tc.assertFunc(collect, pub.events, pub.cursors, filterLogShowLogline(buf.Bytes()))
					}
				},
				30*time.Second, time.Second,
			)
		})
	}
}

const cmdStartPrefix = "exec command start: "

func filterLogStreamLogline(buf []byte) string {
	const cmd = "/usr/bin/log stream"
	return filterLogCmdLine(buf, cmd)
}

func filterLogShowLogline(buf []byte) string {
	const cmd = "/usr/bin/log show"
	return filterLogCmdLine(buf, cmd)
}

func filterLogCmdLine(buf []byte, cmdPrefix string) string {
	scanner := bufio.NewScanner(bytes.NewBuffer(buf))
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(text, "\t")
		if len(parts) != 4 {
			continue
		}

		cmd := strings.TrimPrefix(parts[3], cmdStartPrefix)
		if strings.HasPrefix(cmd, cmdPrefix) {
			return cmd
		}
	}
	return ""
}

func eventsAndCursorAssertN(n int) func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time, cmd ...string) {
	return func(collect *assert.CollectT, events []beat.Event, cursors []*time.Time, cmd ...string) {
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
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	// Use the 'tar' command to extract the .tar.gz file
	cmd := exec.Command("tar", "-xzf", tarGzPath, "-C", tempDir)

	// Run the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract .tar.gz: %v", err)
	}

	return path.Join(tempDir, "test.logarchive"), nil
}
