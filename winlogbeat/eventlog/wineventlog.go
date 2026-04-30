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
	"github.com/elastic/elastic-agent-libs/monitoring"
	wininfo "github.com/elastic/go-sysinfo/providers/windows"
)

var errRecordIDGap = errors.New("record ID gap detected")
var errRenderNoEvent = errors.New("rendering error without partial event")

const renderNoEventRetryLimit = 3
const recordIDGapRetryLimit = 3

type gapDetectedError struct {
	channel  string
	previous uint64
	current  uint64
	bookmark string
}

func (e *gapDetectedError) Error() string {
	return fmt.Sprintf("%v in channel %q (previous=%d current=%d)",
		errRecordIDGap, e.channel, e.previous, e.current)
}

func (e *gapDetectedError) Unwrap() error { return errRecordIDGap }
func (e *gapDetectedError) Bookmark() string {
	return e.bookmark
}

func (e *gapDetectedError) RetryKey() string {
	return fmt.Sprintf("%s:%d:%d", e.channel, e.previous, e.current)
}

type renderNoEventError struct {
	cause    error
	bookmark string
}

func (e *renderNoEventError) Error() string {
	if e.cause == nil {
		return errRenderNoEvent.Error()
	}
	return fmt.Sprintf("%v: %v", errRenderNoEvent, e.cause)
}

func (e *renderNoEventError) Unwrap() error { return errRenderNoEvent }
func (e *renderNoEventError) Bookmark() string {
	return e.bookmark
}

func (e *renderNoEventError) RetryKey() string {
	if e.bookmark != "" {
		return e.bookmark
	}
	return fmt.Sprintf("no-bookmark:%v", e.cause)
}

func (l *winEventLog) newRenderNoEventError(handle win.EvtHandle, cause error) *renderNoEventError {
	bookmark, bookmarkErr := l.createBookmarkFromEvent(handle)
	return &renderNoEventError{
		cause:    errors.Join(cause, bookmarkErr),
		bookmark: bookmark,
	}
}

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	config      config
	query       string
	filter      *recordFilter
	id          string                   // Identifier of this event log.
	channelName string                   // Name of the channel from which to read.
	file        bool                     // Reading from file rather than channel.
	maxRead     int                      // Maximum number returned in one Read.
	lastRead    checkpoint.EventLogState // Record number of the last read event.
	log         *logp.Logger

	iterator *win.EventIterator
	renderer win.EventRenderer

	metrics *inputMetrics

	renderNoEventKey   string
	renderNoEventCount int
	gapRetryKey        string
	gapRetryCount      int
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(options *conf.C) (EventLog, error) {
	var err error

	c := defaultConfig()
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
		if l.hasWin2025ForwardedBugRisk() {
			l.log.Warn("using a custom XML query with Windows Server 2025 forwarded events can hit a known Event Log API issue")
		}
		l.query = c.XMLQuery
	} else {
		l.log = l.log.With("channel", c.Name)
		if info, err := os.Stat(c.Name); err == nil && info.Mode().IsRegular() {
			path, err := filepath.Abs(c.Name)
			if err != nil {
				return nil, err
			}
			l.file = true
			l.channelName = path
		}

		l.filter, err = newRecordFilter(c.SimpleQuery)
		if err != nil {
			return nil, err
		}

		// Always use an unfiltered query and apply configured filters in Go.
		l.query = "*"
	}

	switch c.IncludeXML || l.isForwarded() {
	case true:
		l.renderer = win.NewXMLRenderer(
			c.EventLanguage,
			l.isForwarded(),
			win.NilHandle, l.log)
	case false:
		l.renderer, err = win.NewRenderer(
			c.EventLanguage,
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

func (l *winEventLog) shouldDetectGap(prevRecordID, currentRecordID uint64) bool {
	if l.file || l.isForwarded() || prevRecordID == 0 {
		return false
	}
	return currentRecordID > prevRecordID+1
}

func (l *winEventLog) hasWin2025ForwardedBugRisk() bool {
	if !l.isForwarded() {
		return false
	}
	osinfo, err := wininfo.OperatingSystem()
	if err != nil {
		l.log.Warnf("failed to get OS info while checking known issue conditions: %v", err)
		return false
	}
	return strings.Contains(osinfo.Name, "2025")
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

// IgnoreMissingChannel returns true if missing channels should be ignored.
func (l *winEventLog) IgnoreMissingChannel() bool {
	return !l.file && (l.config.IgnoreMissingChannel == nil || *l.config.IgnoreMissingChannel)
}

func (l *winEventLog) Open(state checkpoint.EventLogState, metricsRegistry *monitoring.Registry) error {
	l.lastRead = state
	// we need to defer metrics initialization since when the event log
	// is used from winlog input it would register it twice due to CheckConfig calls
	if l.metrics == nil && l.id != "" {
		l.metrics = newInputMetrics(l.channelName, metricsRegistry, l.log)
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
	channelPath := ""
	if l.config.XMLQuery == "" {
		channelPath = l.channelName
	}
	h, err := win.Subscribe(
		0, // Session - nil for localhost
		signalEvent,
		channelPath,
		l.query,                 // Query - nil means all events
		win.EvtHandle(bookmark), // Bookmark - for resuming from a specific event
		flags)

	if err == nil {
		return h, nil
	}
	if errors.Is(err, win.ERROR_NOT_FOUND) ||
		errors.Is(err, win.ERROR_EVT_QUERY_RESULT_STALE) ||
		errors.Is(err, win.ERROR_EVT_QUERY_RESULT_INVALID_POSITION) {
		// The bookmarked event was not found, we retry the subscription from the start.
		incrementMetric(readErrors, err)
		// Clear persisted checkpoint fields before restarting at oldest so stale
		// state does not produce synthetic gap checks on the next records.
		l.resetLastRead()
		return win.Subscribe(0, signalEvent, channelPath, l.query, 0, win.EvtSubscribeStartAtOldestRecord)
	}
	return 0, err
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
			if returnErr := l.handleProcessError(err); returnErr != nil {
				return records, returnErr
			}
			continue
		}
		l.resetRenderNoEventRetry()
		// Any successfully processed event breaks a previous gap retry streak.
		l.resetGapRetry()
		if l.filter != nil && !l.filter.match(record) {
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

func (l *winEventLog) handleProcessError(err error) error {
	var renderErr *renderNoEventError
	if errors.As(err, &renderErr) {
		// Render-no-event and gap retries are independent counters; reset gap
		// state while handling render failures to avoid cross-error pollution.
		l.resetGapRetry()
		l.metrics.logError(err)
		retryCount := l.incrementRenderNoEventRetry(renderErr.RetryKey())
		if retryCount <= renderNoEventRetryLimit {
			return err
		}

		l.log.Errorw("Dropping poison event after repeated render failures.",
			"channel", l.channelName,
			"retry_count", retryCount,
			"retry_limit", renderNoEventRetryLimit)
		l.metrics.logDropped(err)
		incrementMetric(dropReasons, errRenderNoEvent.Error())
		if bookmark := renderErr.Bookmark(); bookmark != "" {
			l.lastRead.Bookmark = bookmark
		} else {
			l.log.Errorw("Dropping poison event without bookmark after repeated render failures.",
				"channel", l.channelName,
				"retry_count", retryCount,
				"retry_limit", renderNoEventRetryLimit)
			l.resetRenderNoEventRetry()
			return nil
		}
		// We advanced by bookmark only; clear record number so gap detection
		// does not compare against stale numeric state.
		l.lastRead.RecordNumber = 0
		l.resetRenderNoEventRetry()
		return nil
	}

	l.resetRenderNoEventRetry()
	var gapErr *gapDetectedError
	if errors.As(err, &gapErr) {
		// Gap errors are retried first (runner reset + backoff) because in-flight
		// events can arrive and fill the gap shortly after detection.
		l.metrics.logError(err)
		retryCount := l.incrementGapRetry(gapErr.RetryKey())
		if retryCount <= recordIDGapRetryLimit {
			return err
		}

		// After repeated retries on the same gap boundary, accept the gap and
		// advance state to avoid an infinite reset loop.
		l.log.Errorw("Accepting record ID gap after repeated retries.",
			"channel", l.channelName,
			"retry_count", retryCount,
			"retry_limit", recordIDGapRetryLimit,
			"previous_record_id", gapErr.previous,
			"current_record_id", gapErr.current,
			"missing", gapErr.current-gapErr.previous-1)
		l.metrics.logDropped(err)
		incrementMetric(dropReasons, errRecordIDGap.Error())
		if bookmark := gapErr.Bookmark(); bookmark != "" {
			l.lastRead.Bookmark = bookmark
		}
		l.lastRead.RecordNumber = gapErr.current
		// Gap was handled (accepted/dropped), so clear retry state for the next boundary.
		l.resetGapRetry()
		return nil
	}

	// Different/non-gap error path; discard any stale gap retry context.
	l.resetGapRetry()
	if errors.Is(err, errRecordIDGap) || errors.Is(err, errRenderNoEvent) {
		l.metrics.logError(err)
		return err
	}

	l.metrics.logError(err)
	l.log.Warnw("Dropping event due to rendering error.", "error", err)
	l.metrics.logDropped(err)
	incrementMetric(dropReasons, err)
	return nil
}

func (l *winEventLog) processHandle(h win.EvtHandle) (*Record, error) {
	defer h.Close()

	// NOTE: Render can return an error and a partial event.
	evt, xml, err := l.renderer.Render(h)
	if evt == nil {
		return nil, l.newRenderNoEventError(h, err)
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

	prevRecordID := l.lastRead.RecordNumber
	if l.shouldDetectGap(prevRecordID, r.RecordID) {
		// Gap detection is channel-only. File reads can legitimately contain
		// non-contiguous record IDs and should not trigger recovery. Forwarded
		// events can also be non-contiguous.
		l.log.Warnw("Record ID gap detected, resetting subscription.",
			"channel", l.channelName,
			"previous_record_id", prevRecordID,
			"current_record_id", r.RecordID,
			"missing", r.RecordID-prevRecordID-1)
		return nil, l.newGapDetectedError(h, prevRecordID, r.RecordID)
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

func (l *winEventLog) newGapDetectedError(handle win.EvtHandle, previousRecordID, currentRecordID uint64) *gapDetectedError {
	// Capture the current event bookmark so the gap circuit-breaker can skip
	// this boundary if retries are exhausted.
	bookmark, err := l.createBookmarkFromEvent(handle)
	if err != nil {
		l.metrics.logError(err)
		l.log.Warnw("Failed creating bookmark for record ID gap recovery.", "error", err)
	}

	return &gapDetectedError{
		channel:  l.channelName,
		previous: previousRecordID,
		current:  currentRecordID,
		bookmark: bookmark,
	}
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
	// Only close the iterator, keep the renderer alive to avoid
	// unnecessarily recreating render contexts. The renderer's
	// systemContext and userContext should remain valid across
	// session resets since they were created independently.
	if l.iterator == nil {
		return nil
	}
	err := l.iterator.Close()
	l.iterator = nil
	return err
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

func (l *winEventLog) incrementRenderNoEventRetry(bookmark string) int {
	if bookmark == l.renderNoEventKey && l.renderNoEventCount > 0 {
		l.renderNoEventCount++
		return l.renderNoEventCount
	}

	l.renderNoEventKey = bookmark
	l.renderNoEventCount = 1
	return l.renderNoEventCount
}

func (l *winEventLog) resetRenderNoEventRetry() {
	l.renderNoEventKey = ""
	l.renderNoEventCount = 0
}

func (l *winEventLog) incrementGapRetry(key string) int {
	if key == l.gapRetryKey && l.gapRetryCount > 0 {
		l.gapRetryCount++
		return l.gapRetryCount
	}

	l.gapRetryKey = key
	l.gapRetryCount = 1
	return l.gapRetryCount
}

func (l *winEventLog) resetGapRetry() {
	l.gapRetryKey = ""
	l.gapRetryCount = 0
}

func (l *winEventLog) resetLastRead() {
	// Keep this scoped to fields used by resubscribe/gap logic.
	l.lastRead.Bookmark = ""
	l.lastRead.RecordNumber = 0
}
