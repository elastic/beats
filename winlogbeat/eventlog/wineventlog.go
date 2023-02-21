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
	"errors"
	"expvar"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

var (
	detailSelector = "eventlog_detail"
	detailf        = logp.MakeDebug(detailSelector)

	// dropReasons contains counters for the number of dropped events for each
	// reason.
	dropReasons = expvar.NewMap("drop_reasons")

	// readErrors contains counters for the read error types that occur.
	readErrors = expvar.NewMap("read_errors")
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

	// metaTTL is the length of time a WinMeta value is valid in the cache.
	metaTTL = time.Hour
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

// query contains parameters used to customize the event log data that is
// queried from the log.
type query struct {
	IgnoreOlder time.Duration `config:"ignore_older"` // Ignore records older than this period of time.
	EventID     string        `config:"event_id"`     // White-list and black-list of events.
	Level       string        `config:"level"`        // Severity level.
	Provider    []string      `config:"provider"`     // Provider (source name).
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
	return fmt.Errorf("invalid no_more_events action: %v", v)
}

// String returns the name of the action.
func (a NoMoreEventsAction) String() string { return noMoreEventsActionNames[a] }

// defaultWinEventLogConfig is the default configuration for new wineventlog readers.
var defaultWinEventLogConfig = winEventLogConfig{
	BatchReadSize: 100,
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

	winMetaCache // Cached WinMeta tables by provider.

	logPrefix string // String to prefix on log messages.

	metrics *inputMetrics
}

func newEventLogging(options *conf.C) (EventLog, error) {
	cfgwarn.Deprecate("8.0.0", fmt.Sprintf("api %s is deprecated and %s will be used instead", eventLoggingAPIName, winEventLogAPIName))
	return newWinEventLog(options)
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(options *conf.C) (EventLog, error) {
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
		id:           id,
		config:       c,
		query:        xmlQuery,
		channelName:  c.Name,
		file:         filepath.IsAbs(c.Name),
		maxRead:      c.BatchReadSize,
		renderBuf:    make([]byte, renderBufferSize),
		outputBuf:    sys.NewByteBuffer(renderBufferSize),
		cache:        newMessageFilesCache(id, eventMetadataHandle, freeHandle),
		winMetaCache: newWinMetaCache(metaTTL),
		logPrefix:    fmt.Sprintf("WinEventLog[%s]", id),
		metrics:      newInputMetrics(c.Name, id),
	}

	// Forwarded events should be rendered using RenderEventXML. It is more
	// efficient and does not attempt to use local message files for rendering
	// the event's message.
	switch {
	case c.Forwarded == nil && c.Name == "ForwardedEvents",
		c.Forwarded != nil && *c.Forwarded:
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
	var bookmark win.EvtHandle
	var err error
	if len(state.Bookmark) > 0 {
		bookmark, err = win.CreateBookmarkFromXML(state.Bookmark)
	} else if state.RecordNumber > 0 && l.channelName != "" {
		bookmark, err = win.CreateBookmarkFromRecordID(l.channelName, state.RecordNumber)
	}
	if err != nil {
		l.metrics.logError(err)
		return err
	}
	defer win.Close(bookmark)

	if l.file {
		return l.openFile(state, bookmark)
	}
	return l.openChannel(bookmark)
}

func (l *winEventLog) openFile(state checkpoint.EventLogState, bookmark win.EvtHandle) error {
	path := l.channelName

	h, err := win.EvtQuery(0, path, "", win.EvtQueryFilePath|win.EvtQueryForwardDirection)
	if err != nil {
		l.metrics.logError(err)
		return fmt.Errorf("failed to get handle to event log file %v: %w", path, err)
	}

	if bookmark > 0 {
		debugf("%s Seeking to bookmark. timestamp=%v bookmark=%v",
			l.logPrefix, state.Timestamp, state.Bookmark)

		// This seeks to the last read event and strictly validates that the
		// bookmarked record number exists.
		if err = win.EvtSeek(h, 0, bookmark, win.EvtSeekRelativeToBookmark|win.EvtSeekStrict); err == nil {
			// Then we advance past the last read event to avoid sending that
			// event again. This won't fail if we're at the end of the file.
			if seekErr := win.EvtSeek(h, 1, bookmark, win.EvtSeekRelativeToBookmark); seekErr != nil {
				err = fmt.Errorf("failed to seek past bookmarked position: %w", seekErr)
			}
		} else {
			logp.Warn("%s Failed to seek to bookmarked location in %v (error: %v). "+
				"Recovering by reading the log from the beginning. (Did the file "+
				"change since it was last read?)", l.logPrefix, path, err)
			l.metrics.logError(err)
			if seekErr := win.EvtSeek(h, 0, 0, win.EvtSeekRelativeToFirst); seekErr != nil {
				err = fmt.Errorf("failed to seek to beginning of log: %w", seekErr)
			}
		}

		if err != nil {
			l.metrics.logError(err)
			return err
		}
	}

	l.subscription = h
	return nil
}

func (l *winEventLog) openChannel(bookmark win.EvtHandle) error {
	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		l.metrics.logError(err)
		return err
	}
	defer windows.CloseHandle(signalEvent) //nolint:errcheck // This is just a resource release.

	var flags win.EvtSubscribeFlag
	if bookmark > 0 {
		// Use EvtSubscribeStrict to detect when the bookmark is missing and be able to
		// subscribe again from the beginning.
		flags = win.EvtSubscribeStartAfterBookmark | win.EvtSubscribeStrict
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

	switch {
	case errors.Is(err, win.ERROR_NOT_FOUND), errors.Is(err, win.ERROR_EVT_QUERY_RESULT_STALE),
		errors.Is(err, win.ERROR_EVT_QUERY_RESULT_INVALID_POSITION):
		debugf("%s error subscribing (first chance): %v", l.logPrefix, err)
		// The bookmarked event was not found, we retry the subscription from the start.
		l.metrics.logError(err)
		incrementMetric(readErrors, err)
		subscriptionHandle, err = win.Subscribe(0, signalEvent, "", l.query, 0, win.EvtSubscribeStartAtOldestRecord)
	}

	if err != nil {
		l.metrics.logError(err)
		debugf("%s error subscribing (final): %v", l.logPrefix, err)
		return err
	}

	l.subscription = subscriptionHandle
	return nil
}

func (l *winEventLog) Read() ([]Record, error) {
	handles, _, err := l.eventHandles(l.maxRead)
	if err != nil || len(handles) == 0 {
		return nil, err
	}

	var records []Record
	defer func() {
		l.metrics.log(records)
		for _, h := range handles {
			win.Close(h)
		}
	}()
	detailf("%s EventHandles returned %d handles", l.logPrefix, len(handles))

	for _, h := range handles {
		l.outputBuf.Reset()
		err := l.render(h, l.outputBuf)
		var bufErr sys.InsufficientBufferError
		if errors.As(err, &bufErr) {
			detailf("%s Increasing render buffer size to %d", l.logPrefix,
				bufErr.RequiredSize)
			l.renderBuf = make([]byte, bufErr.RequiredSize)
			l.outputBuf.Reset()
			err = l.render(h, l.outputBuf)
		}
		l.metrics.logError(err)
		if err != nil && l.outputBuf.Len() == 0 {
			logp.Err("%s Dropping event with rendering error. %v", l.logPrefix, err)
			l.metrics.logDropped(err)
			incrementMetric(dropReasons, err)
			continue
		}

		r := l.buildRecordFromXML(l.outputBuf.Bytes(), err)
		r.Offset = checkpoint.EventLogState{
			Name:         l.id,
			RecordNumber: r.RecordID,
			Timestamp:    r.TimeCreated.SystemTime,
		}
		if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
			l.metrics.logError(err)
			logp.Warn("%s failed creating bookmark: %v", l.logPrefix, err)
		}
		if r.Message == "" {
			r.Message, err = l.message(h)
			if err != nil {
				l.metrics.logError(err)
				logp.Warn("%s error salvaging message (event id=%d qualifier=%d provider=%q created at %s will be included without a message): %v",
					l.logPrefix, r.EventIdentifier.ID, r.EventIdentifier.Qualifiers, r.Provider.Name, r.TimeCreated.SystemTime, err)
			}
		}
		records = append(records, r)
		l.lastRead = r.Offset
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *winEventLog) eventHandles(maxRead int) ([]win.EvtHandle, int, error) {
	handles, err := win.EventHandles(l.subscription, maxRead)
	switch err { //nolint:errorlint // This is an errno or nil.
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
		l.metrics.logError(err)
		if err := l.Close(); err != nil {
			return nil, 0, fmt.Errorf("failed to recover from RPC_S_INVALID_BOUND: %w", err)
		}
		if err := l.Open(l.lastRead); err != nil {
			return nil, 0, fmt.Errorf("failed to recover from RPC_S_INVALID_BOUND: %w", err)
		}
		return l.eventHandles(maxRead / 2)
	default:
		l.metrics.logError(err)
		incrementMetric(readErrors, err)
		logp.Warn("%s EventHandles returned error %v", l.logPrefix, err)
		return nil, 0, err
	}
}

func (l *winEventLog) buildRecordFromXML(x []byte, recoveredErr error) Record {
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
	winevent.EnrichRawValuesWithNames(l.winMeta(e.Provider.Name), &e)
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

	return r
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

func (l *winEventLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	l.metrics.close()
	return win.Close(l.subscription)
}

// winMetaCache retrieves and caches WinMeta tables by provider name.
// It is a cut down version of the PublisherMetadataStore caching in wineventlog.Renderer.
type winMetaCache struct {
	ttl    time.Duration
	logger *logp.Logger

	mu    sync.RWMutex
	cache map[string]winMetaCacheEntry
}

type winMetaCacheEntry struct {
	expire time.Time
	*winevent.WinMeta
}

func newWinMetaCache(ttl time.Duration) winMetaCache {
	return winMetaCache{cache: make(map[string]winMetaCacheEntry), ttl: ttl, logger: logp.L()}
}

func (c *winMetaCache) winMeta(provider string) *winevent.WinMeta {
	c.mu.RLock()
	e, ok := c.cache[provider]
	c.mu.RUnlock()
	if ok && time.Until(e.expire) > 0 {
		return e.WinMeta
	}

	// Upgrade lock.
	defer c.mu.Unlock()
	c.mu.Lock()

	// Did the cache get updated during lock upgrade?
	// No need to check expiry here since we must have a new entry
	// if there is an entry at all.
	if e, ok := c.cache[provider]; ok {
		return e.WinMeta
	}

	s, err := win.NewPublisherMetadataStore(win.NilHandle, provider, c.logger)
	if err != nil {
		// Return an empty store on error (can happen in cases where the
		// log was forwarded and the provider doesn't exist on collector).
		s = win.NewEmptyPublisherMetadataStore(provider, c.logger)
		logp.Warn("failed to load publisher metadata for %v (returning an empty metadata store): %v", provider, err)
	}
	s.Close()
	c.cache[provider] = winMetaCacheEntry{expire: time.Now().Add(c.ttl), WinMeta: &s.WinMeta}
	return &s.WinMeta
}

// incrementMetric increments a value in the specified expvar.Map. The key
// should be a windows syscall.Errno or a string. Any other types will be
// reported under the "other" key.
func incrementMetric(v *expvar.Map, key interface{}) {
	switch t := key.(type) {
	default:
		v.Add("other", 1)
	case string:
		v.Add(t, 1)
	case syscall.Errno:
		v.Add(strconv.Itoa(int(t)), 1)
	}
}

// inputMetrics handles event log metric reporting.
type inputMetrics struct {
	unregister func()

	lastBatch time.Time

	name        *monitoring.String // name of the provider being read
	events      *monitoring.Uint   // total number of events received
	dropped     *monitoring.Uint   // total number of discarded events
	errors      *monitoring.Uint   // total number of errors
	batchSize   metrics.Sample     // histogram of the number of events in each non-zero batch
	sourceLag   metrics.Sample     // histogram of the difference between timestamped event's creation and reading
	batchPeriod metrics.Sample     // histogram of the elapsed time between non-zero batch reads
}

// newInputMetrics returns an input metric for windows event logs. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(name, id string) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry(name, id, nil)
	out := &inputMetrics{
		unregister:  unreg,
		name:        monitoring.NewString(reg, "provider"),
		events:      monitoring.NewUint(reg, "received_events_total"),
		dropped:     monitoring.NewUint(reg, "discarded_events_total"),
		errors:      monitoring.NewUint(reg, "errors_total"),
		batchSize:   metrics.NewUniformSample(1024),
		sourceLag:   metrics.NewUniformSample(1024),
		batchPeriod: metrics.NewUniformSample(1024),
	}
	out.name.Set(name)
	_ = adapter.NewGoMetrics(reg, "received_events_count", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchSize))
	_ = adapter.NewGoMetrics(reg, "source_lag_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sourceLag))
	_ = adapter.NewGoMetrics(reg, "batch_read_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchPeriod))

	return out
}

// log logs metric for the given batch.
func (m *inputMetrics) log(batch []Record) {
	if m == nil {
		return
	}
	if len(batch) == 0 {
		return
	}

	now := time.Now()
	if !m.lastBatch.IsZero() {
		m.batchPeriod.Update(now.Sub(m.lastBatch).Nanoseconds())
	}
	m.lastBatch = now

	m.events.Add(uint64(len(batch)))
	m.batchSize.Update(int64(len(batch)))
	for _, r := range batch {
		m.sourceLag.Update(now.Sub(r.TimeCreated.SystemTime).Nanoseconds())
	}
}

// logError logs error metrics. Nil errors do not increment the error
// count but the err value is currently otherwise not used. It is included
// to allow easier extension of the metrics to include error stratification.
func (m *inputMetrics) logError(err error) {
	if m == nil {
		return
	}
	if err == nil {
		return
	}
	m.errors.Inc()
}

// logDropped logs dropped event metrics. Nil errors *do* increment the dropped
// count; the value is currently otherwise not used, but is included to allow
// easier extension of the metrics to include error stratification.
func (m *inputMetrics) logDropped(_ error) {
	if m == nil {
		return
	}
	m.dropped.Inc()
}

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	m.unregister()
}
