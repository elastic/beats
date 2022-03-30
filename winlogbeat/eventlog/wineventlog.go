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
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

const (
	// renderBufferSize is the size in bytes of the buffer used to render events.
	renderBufferSize = 1 << 14

	// winEventLogApiName is the name used to identify the Windows Event Log API
	// as both an event type and an API.
	winEventLogAPIName = "wineventlog"

	// eventLoggingAPIName is the name used to identify the Event Logging API
	// as both an event type and an API.
	eventLoggingAPIName = "eventlogging"
)

func init() {
	// Register wineventlog API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogAPIName, 0, newWinEventLog, win.Channels)
		Register(eventLoggingAPIName, 1, newEventLogging, win.Channels)
	}
}

type winEventLogConfig struct {
	ConfigCommon  `config:",inline"`
	BatchReadSize int                `config:"batch_read_size"` // Maximum number of events that Read will return.
	IncludeXML    bool               `config:"include_xml"`
	Forwarded     *bool              `config:"forwarded"`
	SimpleQuery   query              `config:",inline"`
	NoMoreEvents  NoMoreEventsAction `config:"no_more_events"` // Action to take when no more events are available - wait or stop.
	EventLanguage uint32             `config:"language"`
}

// NoMoreEventsAction defines what action for the reader to take when
// ERROR_NO_MORE_ITEMS is returned by the Windows API.
type NoMoreEventsAction uint8

const (
	// Wait for new events.
	Wait NoMoreEventsAction = iota
	// Stop the reader.
	Stop
)

var noMoreEventsActionNames = map[NoMoreEventsAction]string{
	Wait: "wait",
	Stop: "stop",
}

// Unpack sets the action based on the string value.
func (a *NoMoreEventsAction) Unpack(v string) error {
	v = strings.ToLower(v)
	for action, name := range noMoreEventsActionNames {
		if v == name {
			*a = action
			return nil
		}
	}
	return errors.Errorf("invalid no_more_events action: %v", v)
}

// String returns the name of the action.
func (a NoMoreEventsAction) String() string { return noMoreEventsActionNames[a] }

// defaultWinEventLogConfig is the default configuration for new wineventlog readers.
var defaultWinEventLogConfig = winEventLogConfig{
	BatchReadSize: 100,
}

// query contains parameters used to customize the event log data that is
// queried from the log.
type query struct {
	IgnoreOlder time.Duration `config:"ignore_older"` // Ignore records older than this period of time.
	EventID     string        `config:"event_id"`     // White-list and black-list of events.
	Level       string        `config:"level"`        // Severity level.
	Provider    []string      `config:"provider"`     // Provider (source name).
}

// Validate validates the winEventLogConfig data and returns an error describing
// any problems or nil.
func (c *winEventLogConfig) Validate() error {
	var errs multierror.Errors

	if c.XMLQuery != "" {
		if c.ID == "" {
			errs = append(errs, fmt.Errorf("event log is missing an 'id'"))
		}

		// Check for XML syntax errors. This does not check the validity of the query itself.
		if err := xml.Unmarshal([]byte(c.XMLQuery), &struct{}{}); err != nil {
			errs = append(errs, fmt.Errorf("invalid xml_query: %w", err))
		}

		switch {
		case c.Name != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'name'"))
		case c.SimpleQuery.IgnoreOlder != 0:
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'ignore_older'"))
		case c.SimpleQuery.Level != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'level'"))
		case c.SimpleQuery.EventID != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'event_id'"))
		case len(c.SimpleQuery.Provider) != 0:
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'provider'"))
		}
	} else if c.Name == "" {
		errs = append(errs, fmt.Errorf("event log is missing a 'name'"))
	}

	return errs.Err()
}

// Validate that winEventLog implements the EventLog interface.
var _ EventLog = &winEventLog{}

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	config       winEventLogConfig
	query        string
	id           string                   // Identifier of this event log.
	channelName  string                   // Name of the channel from which to read.
	file         bool                     // Reading from file rather than channel.
	subscription win.EvtHandle            // Handle to the subscription.
	maxRead      int                      // Maximum number returned in one Read.
	lastRead     checkpoint.EventLogState // Record number of the last read event.

	render    func(event win.EvtHandle, out io.Writer) error // Function for rendering the event to XML.
	message   func(event win.EvtHandle) (string, error)      // Message fallback function.
	renderBuf []byte                                         // Buffer used for rendering event.
	outputBuf *sys.ByteBuffer                                // Buffer for receiving XML
	cache     *messageFilesCache                             // Cached mapping of source name to event message file handles.

	logPrefix string // String to prefix on log messages.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.id
}

func (l *winEventLog) Open(state checkpoint.EventLogState) error {
	var bookmark win.EvtHandle
	var err error
	if len(state.Bookmark) > 0 {
		bookmark, err = win.CreateBookmarkFromXML(state.Bookmark)
	} else if state.RecordNumber > 0 && l.channelName != "" {
		bookmark, err = win.CreateBookmarkFromRecordID(l.channelName, state.RecordNumber)
	}
	if err != nil {
		return err
	}
	defer win.Close(bookmark)

	if l.file {
		return l.openFile(state, bookmark)
	}
	return l.openChannel(bookmark)
}

func (l *winEventLog) openChannel(bookmark win.EvtHandle) error {
	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil
	}
<<<<<<< HEAD
	defer windows.CloseHandle(signalEvent)
=======
	defer windows.CloseHandle(signalEvent) //nolint:errcheck // This is just a resource release.
>>>>>>> 34bdc3d468 (winlogbeat: fix event handling for Windows 2022 (#30942))

	var flags win.EvtSubscribeFlag
	if bookmark > 0 {
		flags = win.EvtSubscribeStartAfterBookmark
	} else {
		flags = win.EvtSubscribeStartAtOldestRecord
	}

	debugf("%s using subscription query=%s", l.logPrefix, l.query)
	subscriptionHandle, err := win.Subscribe(
		0, // Session - nil for localhost
		signalEvent,
		"",       // Channel - empty b/c channel is in the query
		l.query,  // Query - nil means all events
		bookmark, // Bookmark - for resuming from a specific event
		flags)
	if err != nil {
		return err
	}

	l.subscription = subscriptionHandle
	return nil
}

func (l *winEventLog) openFile(state checkpoint.EventLogState, bookmark win.EvtHandle) error {
	path := l.channelName

	h, err := win.EvtQuery(0, path, "", win.EvtQueryFilePath|win.EvtQueryForwardDirection)
	if err != nil {
		return errors.Wrapf(err, "failed to get handle to event log file %v", path)
	}

	if bookmark > 0 {
		debugf("%s Seeking to bookmark. timestamp=%v bookmark=%v",
			l.logPrefix, state.Timestamp, state.Bookmark)

		// This seeks to the last read event and strictly validates that the
		// bookmarked record number exists.
		if err = win.EvtSeek(h, 0, bookmark, win.EvtSeekRelativeToBookmark|win.EvtSeekStrict); err == nil {
			// Then we advance past the last read event to avoid sending that
			// event again. This won't fail if we're at the end of the file.
			err = errors.Wrap(
				win.EvtSeek(h, 1, bookmark, win.EvtSeekRelativeToBookmark),
				"failed to seek past bookmarked position")
		} else {
			logp.Warn("%s Failed to seek to bookmarked location in %v (error: %v). "+
				"Recovering by reading the log from the beginning. (Did the file "+
				"change since it was last read?)", l.logPrefix, path, err)
			err = errors.Wrap(
				win.EvtSeek(h, 0, 0, win.EvtSeekRelativeToFirst),
				"failed to seek to beginning of log")
		}

		if err != nil {
			return err
		}
	}

	l.subscription = h
	return nil
}

func (l *winEventLog) Read() ([]Record, error) {
	handles, _, err := l.eventHandles(l.maxRead)
	if err != nil || len(handles) == 0 {
		return nil, err
	}
	defer func() {
		for _, h := range handles {
			win.Close(h)
		}
	}()
	detailf("%s EventHandles returned %d handles", l.logPrefix, len(handles))

<<<<<<< HEAD
	var records []Record
	for _, h := range handles {
		l.outputBuf.Reset()
		err := l.render(h, l.outputBuf)
		if bufErr, ok := err.(sys.InsufficientBufferError); ok {
=======
	var records []Record //nolint:prealloc // This linter gives bad advice and does not take into account conditionals in loops.
	for _, h := range handles {
		l.outputBuf.Reset()
		err := l.render(h, l.outputBuf)
		var bufErr sys.InsufficientBufferError
		if errors.As(err, &bufErr) {
>>>>>>> 34bdc3d468 (winlogbeat: fix event handling for Windows 2022 (#30942))
			detailf("%s Increasing render buffer size to %d", l.logPrefix,
				bufErr.RequiredSize)
			l.renderBuf = make([]byte, bufErr.RequiredSize)
			l.outputBuf.Reset()
			err = l.render(h, l.outputBuf)
		}
		if err != nil && l.outputBuf.Len() == 0 {
			logp.Err("%s Dropping event with rendering error. %v", l.logPrefix, err)
			incrementMetric(dropReasons, err)
			continue
		}

		r, _ := l.buildRecordFromXML(l.outputBuf.Bytes(), err)
		r.Offset = checkpoint.EventLogState{
			Name:         l.id,
			RecordNumber: r.RecordID,
			Timestamp:    r.TimeCreated.SystemTime,
		}
		if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
			logp.Warn("%s failed creating bookmark: %v", l.logPrefix, err)
		}
		if r.Message == "" {
			r.Message, err = l.message(h)
			if err != nil {
				logp.Err("%s error salvaging message: %v", l.logPrefix, err)
			}
		}
		records = append(records, r)
		l.lastRead = r.Offset
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *winEventLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return win.Close(l.subscription)
}

func (l *winEventLog) eventHandles(maxRead int) ([]win.EvtHandle, int, error) {
	handles, err := win.EventHandles(l.subscription, maxRead)
<<<<<<< HEAD
	switch err {
=======
	switch err { //nolint:errorlint // This is an errno or nil.
>>>>>>> 34bdc3d468 (winlogbeat: fix event handling for Windows 2022 (#30942))
	case nil:
		if l.maxRead > maxRead {
			debugf("%s Recovered from RPC_S_INVALID_BOUND error (errno 1734) "+
				"by decreasing batch_read_size to %v", l.logPrefix, maxRead)
		}
		return handles, maxRead, nil
	case win.ERROR_NO_MORE_ITEMS:
		detailf("%s No more events", l.logPrefix)
		if l.config.NoMoreEvents == Stop {
			return nil, maxRead, io.EOF
		}
		return nil, maxRead, nil
	case win.RPC_S_INVALID_BOUND:
		incrementMetric(readErrors, err)
		if err := l.Close(); err != nil {
			return nil, 0, errors.Wrap(err, "failed to recover from RPC_S_INVALID_BOUND")
		}
		if err := l.Open(l.lastRead); err != nil {
			return nil, 0, errors.Wrap(err, "failed to recover from RPC_S_INVALID_BOUND")
		}
		return l.eventHandles(maxRead / 2)
	default:
		incrementMetric(readErrors, err)
		logp.Warn("%s EventHandles returned error %v", l.logPrefix, err)
		return nil, 0, err
	}
}

func (l *winEventLog) buildRecordFromXML(x []byte, recoveredErr error) (Record, error) {
	includeXML := l.config.IncludeXML
	e, err := winevent.UnmarshalXML(x)
	if err != nil {
		e.RenderErr = append(e.RenderErr, err.Error())
		// Add raw XML to event.original when decoding fails
		includeXML = true
	}

	err = winevent.PopulateAccount(&e.User)
	if err != nil {
		debugf("%s SID %s account lookup failed. %v", l.logPrefix,
			e.User.Identifier, err)
	}

	if e.RenderErrorCode != 0 {
		// Convert the render error code to an error message that can be
		// included in the "error.message" field.
		e.RenderErr = append(e.RenderErr, syscall.Errno(e.RenderErrorCode).Error())
	} else if recoveredErr != nil {
		e.RenderErr = append(e.RenderErr, recoveredErr.Error())
	}

	// Get basic string values for raw fields.
	winevent.EnrichRawValuesWithNames(nil, &e)
	if e.Level == "" {
		// Fallback on LevelRaw if the Level is not set in the RenderingInfo.
		e.Level = win.EventLevel(e.LevelRaw).String()
	}

	if logp.IsDebug(detailSelector) {
		detailf("%s XML=%s Event=%+v", l.logPrefix, x, e)
	}

	r := Record{
		API:   winEventLogAPIName,
		Event: e,
	}

	if l.file {
		r.File = l.id
	}

	if includeXML {
		r.XML = string(x)
	}

	return r, nil
}

func newEventLogging(options *common.Config) (EventLog, error) {
	cfgwarn.Deprecate("8.0.0", fmt.Sprintf("api %s is deprecated and %s will be used instead", eventLoggingAPIName, winEventLogAPIName))
	return newWinEventLog(options)
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(options *common.Config) (EventLog, error) {
	var xmlQuery string
	var err error

	c := defaultWinEventLogConfig
	if err = readConfig(options, &c); err != nil {
		return nil, err
	}

	id := c.ID
	if id == "" {
		id = c.Name
	}

	if c.XMLQuery != "" {
		xmlQuery = c.XMLQuery
	} else {
		xmlQuery, err = win.Query{
			Log:         c.Name,
			IgnoreOlder: c.SimpleQuery.IgnoreOlder,
			Level:       c.SimpleQuery.Level,
			EventID:     c.SimpleQuery.EventID,
			Provider:    c.SimpleQuery.Provider,
		}.Build()
		if err != nil {
			return nil, err
		}
	}

	eventMetadataHandle := func(providerName, sourceName string) sys.MessageFiles {
		mf := sys.MessageFiles{SourceName: sourceName}
		h, err := win.OpenPublisherMetadata(0, sourceName, c.EventLanguage)
		if err != nil {
			mf.Err = err
			return mf
		}

		mf.Handles = []sys.FileHandle{{Handle: uintptr(h)}}
		return mf
	}

	freeHandle := func(handle uintptr) error {
		return win.Close(win.EvtHandle(handle))
	}

	if filepath.IsAbs(c.Name) {
		c.Name = filepath.Clean(c.Name)
	}

	l := &winEventLog{
		id:          id,
		config:      c,
		query:       xmlQuery,
		channelName: c.Name,
		file:        filepath.IsAbs(c.Name),
		maxRead:     c.BatchReadSize,
		renderBuf:   make([]byte, renderBufferSize),
		outputBuf:   sys.NewByteBuffer(renderBufferSize),
		cache:       newMessageFilesCache(id, eventMetadataHandle, freeHandle),
		logPrefix:   fmt.Sprintf("WinEventLog[%s]", id),
	}

	// Forwarded events should be rendered using RenderEventXML. It is more
	// efficient and does not attempt to use local message files for rendering
	// the event's message.
	switch {
	case c.Forwarded == nil && c.Name == "ForwardedEvents",
		c.Forwarded != nil && *c.Forwarded == true:
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEventXML(event, l.renderBuf, out)
		}
	default:
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEvent(event, c.EventLanguage, l.renderBuf, l.cache.get, out)
		}
	}
	l.message = func(event win.EvtHandle) (string, error) {
		return win.Message(event, l.renderBuf, l.cache.get)
	}

	return l, nil
}

func (l *winEventLog) createBookmarkFromEvent(evtHandle win.EvtHandle) (string, error) {
	bmHandle, err := win.CreateBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", err
	}
	l.outputBuf.Reset()
	err = win.RenderBookmarkXML(bmHandle, l.renderBuf, l.outputBuf)
	win.Close(bmHandle)
	return string(l.outputBuf.Bytes()), err
}
