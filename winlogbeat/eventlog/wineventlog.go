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

//go:build windows

package eventlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	wininfo "github.com/elastic/go-sysinfo/providers/windows"
)

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	config      config
	query       string
	id          string                   // Identifier of this event log.
	channelName string                   // Name of the channel from which to read.
	file        bool                     // Reading from file rather than channel.
	maxRead     int                      // Maximum number returned in one Read.
	lastRead    checkpoint.EventLogState // Record number of the last read event.
	log         *logp.Logger

	iterator *win.EventIterator
	renderer win.EventRenderer

	metrics *inputMetrics
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(options *conf.C) (EventLog, error) {
	var err error

	c := config{BatchReadSize: 512}
	if err := readConfig(options, &c); err != nil {
		return nil, err
	}

	id := c.ID
	if id == "" {
		id = c.Name
	}

	l := &winEventLog{
		config:      c,
		id:          id,
		channelName: c.Name,
		maxRead:     c.BatchReadSize,
		log:         logp.NewLogger("wineventlog").With("id", id),
	}

	if c.XMLQuery != "" {
		if l.skipQueryFilters() {
			l.log.Warn("you are using a custom XML query with Windows Server 2025 and forwarded events, " +
				"this is not recommended due to a known issue with that can crash the Event Log service if using" +
				" query filters. Please use a custom query without filters or use the default query")
		}
		l.query = c.XMLQuery
	} else {
		l.log = l.log.With("channel", c.Name)
		queryLog := c.Name
		if info, err := os.Stat(c.Name); err == nil && info.Mode().IsRegular() {
			path, err := filepath.Abs(c.Name)
			if err != nil {
				return nil, err
			}
			l.file = true
			queryLog = "file://" + path
		}

		winQuery := win.Query{
			Log: queryLog,
		}

		if !l.skipQueryFilters() {
			winQuery.IgnoreOlder = c.SimpleQuery.IgnoreOlder
			winQuery.Level = c.SimpleQuery.Level
			winQuery.EventID = c.SimpleQuery.EventID
			winQuery.Provider = c.SimpleQuery.Provider
		} else {
			l.log.Warn("skipping query filters for Windows Server 2025 due to known issue" +
				" with Event Log API and forwarded events")
		}

		l.query, err = winQuery.Build()
		if err != nil {
			return nil, err
		}
	}

	switch c.IncludeXML {
	case true:
		l.renderer = win.NewXMLRenderer(
			win.RenderConfig{
				IsForwarded: l.isForwarded(),
				Locale:      c.EventLanguage,
			},
			win.NilHandle, l.log)
	case false:
		l.renderer, err = win.NewRenderer(
			win.RenderConfig{
				IsForwarded: l.isForwarded(),
				Locale:      c.EventLanguage,
			},
			win.NilHandle, l.log)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (l *winEventLog) isForwarded() bool {
	c := l.config
	return (c.Forwarded != nil && *c.Forwarded) || (c.Forwarded == nil && c.Name == "ForwardedEvents")
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.id
}

// Channel returns the event log's channel name.
func (l *winEventLog) Channel() string {
	return l.channelName
}

// IsFile returns true if the event log is an evtx file.
func (l *winEventLog) IsFile() bool {
	return l.file
}

func (l *winEventLog) Open(state checkpoint.EventLogState) error {
	l.lastRead = state
	// we need to defer metrics initialization since when the event log
	// is used from winlog input it would register it twice due to CheckConfig calls
	if l.metrics == nil {
		l.metrics = newInputMetrics(l.channelName, l.id)
	}

	var err error
	l.iterator, err = win.NewEventIterator(
		win.WithSubscriptionFactory(func() (handle win.EvtHandle, err error) {
			return l.open(l.lastRead)
		}),
		win.WithBatchSize(l.maxRead))
	return err
}

func (l *winEventLog) open(state checkpoint.EventLogState) (win.EvtHandle, error) {
	var bookmark win.Bookmark
	if len(state.Bookmark) > 0 {
		var err error
		bookmark, err = win.NewBookmarkFromXML(state.Bookmark)
		if err != nil {
			return win.NilHandle, err
		}
		defer bookmark.Close()
	}

	if l.file {
		return l.openFile(state, bookmark)
	}
	return l.openChannel(bookmark)
}

func (l *winEventLog) openFile(state checkpoint.EventLogState, bookmark win.Bookmark) (win.EvtHandle, error) {
	path := l.channelName

	h, err := win.EvtQuery(0, path, l.query, win.EvtQueryFilePath|win.EvtQueryForwardDirection)
	if err != nil {
		return win.NilHandle, fmt.Errorf("failed to get handle to event log file %v: %w", path, err)
	}

	if bookmark > 0 {
		l.log.Debugf("Seeking to bookmark. timestamp=%v bookmark=%v",
			state.Timestamp, state.Bookmark)

		// This seeks to the last read event and strictly validates that the
		// bookmarked record number exists.
		if err = win.EvtSeek(h, 0, win.EvtHandle(bookmark), win.EvtSeekRelativeToBookmark|win.EvtSeekStrict); err == nil {
			// Then we advance past the last read event to avoid sending that
			// event again. This won't fail if we're at the end of the file.
			if seekErr := win.EvtSeek(h, 1, win.EvtHandle(bookmark), win.EvtSeekRelativeToBookmark); seekErr != nil {
				err = fmt.Errorf("failed to seek past bookmarked position: %w", seekErr)
			}
		} else {
			l.log.Warnf("s Failed to seek to bookmarked location in %v (error: %v). "+
				"Recovering by reading the log from the beginning. (Did the file "+
				"change since it was last read?)", path, err)
			if seekErr := win.EvtSeek(h, 0, 0, win.EvtSeekRelativeToFirst); seekErr != nil {
				err = fmt.Errorf("failed to seek to beginning of log: %w", seekErr)
			}
		}

		if err != nil {
			return win.NilHandle, err
		}
	}

	return h, err
}

func (l *winEventLog) openChannel(bookmark win.Bookmark) (win.EvtHandle, error) {
	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return win.NilHandle, err
	}
	defer windows.CloseHandle(signalEvent) //nolint:errcheck // This is just a resource release.

	var flags win.EvtSubscribeFlag
	if bookmark > 0 {
		flags = win.EvtSubscribeStartAfterBookmark
		if !l.isForwarded() {
			// Use EvtSubscribeStrict to detect when the bookmark is missing and be able to
			// subscribe again from the beginning.
			flags |= win.EvtSubscribeStrict
		}
	} else {
		flags = win.EvtSubscribeStartAtOldestRecord
	}

	l.log.Debugw("Using subscription query.", "winlog.query", l.query)
	h, err := win.Subscribe(
		0, // Session - nil for localhost
		signalEvent,
		"",                      // Channel - empty b/c channel is in the query
		l.query,                 // Query - nil means all events
		win.EvtHandle(bookmark), // Bookmark - for resuming from a specific event
		flags)

	switch err { //nolint:errorlint // This is an errno or nil.
	case nil:
		return h, nil
	case win.ERROR_NOT_FOUND, win.ERROR_EVT_QUERY_RESULT_STALE, win.ERROR_EVT_QUERY_RESULT_INVALID_POSITION:
		// The bookmarked event was not found, we retry the subscription from the start.
		incrementMetric(readErrors, err)
		return win.Subscribe(0, signalEvent, "", l.query, 0, win.EvtSubscribeStartAtOldestRecord)
	default:
		return 0, err
	}
}

func (l *winEventLog) Read() ([]Record, error) {
	//nolint:prealloc // Avoid unnecessary preallocation for each reader every second when event log is inactive.
	var records []Record
	defer func() {
		l.metrics.log(records)
	}()

	for h, ok := l.iterator.Next(); ok; h, ok = l.iterator.Next() {
		record, err := l.processHandle(h)
		if err != nil {
			l.metrics.logError(err)
			l.log.Warnw("Dropping event due to rendering error.", "error", err)
			l.metrics.logDropped(err)
			incrementMetric(dropReasons, err)
			continue
		}
		records = append(records, *record)

		// It has read the maximum requested number of events.
		if len(records) >= l.maxRead {
			return records, nil
		}
	}

	// An error occurred while retrieving more events.
	if err := l.iterator.Err(); err != nil {
		l.metrics.logError(err)
		return records, err
	}

	// Reader is configured to stop when there are no more events.
	if Stop == l.config.NoMoreEvents {
		return records, io.EOF
	}

	return records, nil
}

func (l *winEventLog) processHandle(h win.EvtHandle) (*Record, error) {
	defer h.Close()

	// NOTE: Render can return an error and a partial event.
	evt, xml, err := l.renderer.Render(h)
	if evt == nil {
		return nil, err
	}
	if err != nil {
		evt.RenderErr = append(evt.RenderErr, err.Error())
	}

	r := &Record{
		Event: *evt,
	}

	if l.config.IncludeXML {
		r.XML = xml
	}

	if l.file {
		r.File = l.id
	}

	r.Offset = checkpoint.EventLogState{
		Name:         l.id,
		RecordNumber: r.RecordID,
		Timestamp:    r.TimeCreated.SystemTime,
	}
	if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
		l.metrics.logError(err)
		l.log.Warnw("Failed creating bookmark.", "error", err)
	}
	l.lastRead = r.Offset
	return r, nil
}

func (l *winEventLog) createBookmarkFromEvent(evtHandle win.EvtHandle) (string, error) {
	bookmark, err := win.NewBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", fmt.Errorf("failed to create new bookmark from event handle: %w", err)
	}
	defer bookmark.Close()

	return bookmark.XML()
}

func (l *winEventLog) Reset() error {
	l.log.Debug("Closing event log reader handles for reset.")
	return l.close()
}

func (l *winEventLog) Close() error {
	l.log.Debug("Closing event log reader handles.")
	l.metrics.close()
	return l.close()
}

func (l *winEventLog) close() error {
	if l.iterator == nil {
		return l.renderer.Close()
	}
	return errors.Join(
		l.iterator.Close(),
		l.renderer.Close(),
	)
}

// FIXME: Windows Server 2025 has a bug in the Windows Event Log API that causes
// the Event Log Service to crash when using some combinations of filters with
// forwarded events. This is a workaround to skip the query filters for
// Windows Server 2025 in such scenarios.
func (l *winEventLog) skipQueryFilters() bool {
	if l.config.Bypass2025Workaround {
		return false
	}
	osinfo, err := wininfo.OperatingSystem()
	if err != nil {
		l.log.Warnf("failed to get OS info: %v", err)
		return false
	}
	return l.isForwarded() && strings.Contains(osinfo.Name, "2025")
}
