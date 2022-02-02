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
// +build windows

package eventlog

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

const (
	// winEventLogExpApiName is the name used to identify the Windows Event Log API
	// as both an event type and an API.
	winEventLogExpAPIName = "wineventlog-experimental"
)

// winEventLogExp implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLogExp struct {
	config      winEventLogConfig
	query       string
	id          string                   // Identifier of this event log.
	channelName string                   // Name of the channel from which to read.
	file        bool                     // Reading from file rather than channel.
	maxRead     int                      // Maximum number returned in one Read.
	lastRead    checkpoint.EventLogState // Record number of the last read event.
	log         *logp.Logger

	iterator *win.EventIterator
	renderer *win.Renderer
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLogExp) Name() string {
	return l.id
}

func (l *winEventLogExp) Open(state checkpoint.EventLogState) error {
	l.lastRead = state

	var err error
	l.iterator, err = win.NewEventIterator(
		win.WithSubscriptionFactory(func() (handle win.EvtHandle, err error) {
			return l.open(l.lastRead)
		}),
		win.WithBatchSize(l.maxRead))
	return err
}

func (l *winEventLogExp) open(state checkpoint.EventLogState) (win.EvtHandle, error) {
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

func (l *winEventLogExp) openChannel(bookmark win.Bookmark) (win.EvtHandle, error) {
	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return win.NilHandle, err
	}
	defer windows.CloseHandle(signalEvent)

	var flags win.EvtSubscribeFlag
	if bookmark > 0 {
		// Use EvtSubscribeStrict to detect when the bookmark is missing and be able to
		// subscribe again from the beginning.
		flags = win.EvtSubscribeStartAfterBookmark | win.EvtSubscribeStrict
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

	switch err {
	case nil:
		return h, nil
	case win.ERROR_NOT_FOUND:
		// The bookmarked event was not found, we retry the subscription from the start.
		return win.Subscribe(0, signalEvent, "", l.query, 0, win.EvtSubscribeStartAtOldestRecord)
	default:
		return 0, err
	}
}

func (l *winEventLogExp) openFile(state checkpoint.EventLogState, bookmark win.Bookmark) (win.EvtHandle, error) {
	path := l.channelName

	h, err := win.EvtQuery(0, path, "", win.EvtQueryFilePath|win.EvtQueryForwardDirection)
	if err != nil {
		return win.NilHandle, errors.Wrapf(err, "failed to get handle to event log file %v", path)
	}

	if bookmark > 0 {
		l.log.Debugf("Seeking to bookmark. timestamp=%v bookmark=%v",
			state.Timestamp, state.Bookmark)

		// This seeks to the last read event and strictly validates that the
		// bookmarked record number exists.
		if err = win.EvtSeek(h, 0, win.EvtHandle(bookmark), win.EvtSeekRelativeToBookmark|win.EvtSeekStrict); err == nil {
			// Then we advance past the last read event to avoid sending that
			// event again. This won't fail if we're at the end of the file.
			err = errors.Wrap(
				win.EvtSeek(h, 1, win.EvtHandle(bookmark), win.EvtSeekRelativeToBookmark),
				"failed to seek past bookmarked position")
		} else {
			l.log.Warnf("s Failed to seek to bookmarked location in %v (error: %v). "+
				"Recovering by reading the log from the beginning. (Did the file "+
				"change since it was last read?)", path, err)
			err = errors.Wrap(
				win.EvtSeek(h, 0, 0, win.EvtSeekRelativeToFirst),
				"failed to seek to beginning of log")
		}

		if err != nil {
			return win.NilHandle, err
		}
	}

	return h, err
}

func (l *winEventLogExp) Read() ([]Record, error) {
	var records []Record

	for h, ok := l.iterator.Next(); ok; h, ok = l.iterator.Next() {
		record, err := l.processHandle(h)
		if err != nil {
			l.log.Warnw("Dropping event due to rendering error.", "error", err)
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
		return records, err
	}

	// Reader is configured to stop when there are no more events.
	if Stop == l.config.NoMoreEvents {
		return records, io.EOF
	}

	return records, nil
}

func (l *winEventLogExp) processHandle(h win.EvtHandle) (*Record, error) {
	defer h.Close()

	// NOTE: Render can return an error and a partial event.
	evt, err := l.renderer.Render(h)
	if evt == nil {
		return nil, err
	}
	if err != nil {
		evt.RenderErr = append(evt.RenderErr, err.Error())
	}

	// TODO: Need to add XML when configured.

	r := &Record{
		API:   winEventLogExpAPIName,
		Event: *evt,
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
		l.log.Warnw("Failed creating bookmark.", "error", err)
	}
	l.lastRead = r.Offset
	return r, nil
}

func (l *winEventLogExp) createBookmarkFromEvent(evtHandle win.EvtHandle) (string, error) {
	bookmark, err := win.NewBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new bookmark from event handle")
	}
	defer bookmark.Close()

	return bookmark.XML()
}

func (l *winEventLogExp) Close() error {
	l.log.Debug("Closing event log reader handles.")
	return multierr.Combine(
		l.iterator.Close(),
		l.renderer.Close(),
	)
}

// newWinEventLogExp creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLogExp(options *common.Config) (EventLog, error) {
	var xmlQuery string
	var err error
	var isFile bool
	var log *logp.Logger

	cfgwarn.Experimental("The %s event log reader is experimental.", winEventLogExpAPIName)

	c := winEventLogConfig{BatchReadSize: 512}
	if err := readConfig(options, &c); err != nil {
		return nil, err
	}

	id := c.ID
	if id == "" {
		id = c.Name
	}

	if c.XMLQuery != "" {
		xmlQuery = c.XMLQuery
		log = logp.NewLogger("wineventlog").With("id", id)
	} else {
		queryLog := c.Name
		if info, err := os.Stat(c.Name); err == nil && info.Mode().IsRegular() {
			path, err := filepath.Abs(c.Name)
			if err != nil {
				return nil, err
			}
			isFile = true
			queryLog = "file://" + path
		}

		xmlQuery, err = win.Query{
			Log:         queryLog,
			IgnoreOlder: c.SimpleQuery.IgnoreOlder,
			Level:       c.SimpleQuery.Level,
			EventID:     c.SimpleQuery.EventID,
			Provider:    c.SimpleQuery.Provider,
		}.Build()
		if err != nil {
			return nil, err
		}

		log = logp.NewLogger("wineventlog").With("id", id).With("channel", c.Name)
	}

	renderer, err := win.NewRenderer(win.NilHandle, log)
	if err != nil {
		return nil, err
	}

	l := &winEventLogExp{
		config:      c,
		query:       xmlQuery,
		id:          id,
		channelName: c.Name,
		file:        isFile,
		maxRead:     c.BatchReadSize,
		renderer:    renderer,
		log:         log,
	}

	return l, nil
}

func init() {
	// Register wineventlog API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogExpAPIName, 10, newWinEventLogExp, win.Channels)
	}
}
