// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	testCases := []struct {
		name                 string
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

			time.AfterFunc(tc.timeUntilClose, cancel)
			wg.Wait()

			assert.EventuallyWithT(t,
				func(collect *assert.CollectT) {
					assert.Equal(collect, tc.expectedLogStreamCmd, filterLogStreamLogline(buf.Bytes()))
					assert.Equal(collect, tc.expectedLogShowCmd, filterLogShowLogline(buf.Bytes()))
					if tc.assertFunc != nil {
						tc.assertFunc(collect, pub.events, pub.cursors)
					}
				},
				10*time.Second, time.Second,
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
