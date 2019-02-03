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

// +build windows

package eventlog

import (
	"fmt"
	"io"
	"path/filepath"
	"syscall"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/sys"
	win "github.com/elastic/beats/winlogbeat/sys/wineventlog"
	"github.com/pkg/errors"
)

const (
	// winEventLogApiName is the name used to identify the Windows Event Log API
	// as both an event type and an API.
	winEventLogFileAPIName = "wineventlogfile"
)

var winEventLogFileConfigKeys = append(commonConfigKeys, "batch_read_size", "include_xml")

type winEventLogFileConfig struct {
	ConfigCommon  `config:",inline"`
	BatchReadSize int  `config:"batch_read_size"`
	IncludeXML    bool `config:"include_xml"`
}

// defaultWinEventLogConfig is the default configuration for new wineventlog readers.
var defaultWinEventLogFileConfig = winEventLogFileConfig{
	BatchReadSize: 1000,
}

type winEventFileLog struct {
	config    winEventLogFileConfig
	path      string
	evtHandle win.EvtHandle                                  // Handle to the query.
	maxRead   int                                            // Maximum number returned in one Read.
	lastRead  checkpoint.EventLogState                       // Record number of the last read event.
	render    func(event win.EvtHandle, out io.Writer) error // Function for rendering the event to XML.
	renderBuf []byte                                         // Buffer used for rendering event.
	outputBuf *sys.ByteBuffer                                // Buffer for receiving XML
	cache     *messageFilesCache                             // Cached mapping of source name to event message file handles.

	logPrefix string // String to prefix on log messages.
}

// Name returns the file path of the event log (i.e. Application, Security, etc.).
func (l *winEventFileLog) Name() string {
	return l.path
}

func (l *winEventFileLog) Open(state checkpoint.EventLogState) error {
	var bookmark win.EvtHandle
	var err error
	if len(state.Bookmark) > 0 {
		bookmark, err = win.CreateBookmarkFromXML(state.Bookmark)
	} else {
		bookmark, err = win.CreateBookmarkFromRecordID(l.path, state.RecordNumber)
	}
	if err != nil {
		return err
	}
	defer win.Close(bookmark)

	debugf("%s using EvtQuery and EvtSeek to read from file %s", l.logPrefix, l.path)

	queryHandle, err := win.EvtQuery(0, l.path, "", win.EvtQueryFilePath)
	if err != nil {
		return err
	}
	l.evtHandle = queryHandle
	err = win.EvtSeek(l.evtHandle, 0, bookmark)
	if err != nil {
		l.lastRead.Bookmark = ""
	}

	return nil
}

func (l *winEventFileLog) ReOpen(state checkpoint.EventLogState) error {
	var bookmark win.EvtHandle
	var err error
	if len(l.lastRead.Bookmark) > 0 {
		bookmark, err = win.CreateBookmarkFromXML(l.lastRead.Bookmark)
	} else {
		bookmark, err = win.CreateBookmarkFromRecordID(l.path, 0)
	}
	if err != nil {
		return err
	}
	defer win.Close(bookmark)

	debugf("%s using EvtQuery and EvtSeek to read from file %s", l.logPrefix, l.path)

	queryHandle, err := win.EvtQuery(0, l.path, "", win.EvtQueryFilePath)
	if err != nil {
		return err
	}
	l.evtHandle = queryHandle
	err = win.EvtSeek(l.evtHandle, 0, bookmark)
	if err != nil {
		l.lastRead.Bookmark = ""
	}

	return nil
}

func (l *winEventFileLog) Read() ([]Record, error) {
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
	var records []Record
	for _, h := range handles {
		l.outputBuf.Reset()
		err := l.render(h, l.outputBuf)
		if bufErr, ok := err.(sys.InsufficientBufferError); ok {
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

		r, err := l.buildRecordFromXML(l.outputBuf.Bytes(), err)
		if err != nil {
			logp.Err("%s Dropping event. %v", l.logPrefix, err)
			incrementMetric(dropReasons, err)
			continue
		}

		r.Offset = checkpoint.EventLogState{
			Name:         l.path,
			RecordNumber: r.RecordID,
			Timestamp:    r.TimeCreated.SystemTime,
		}
		if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
			logp.Warn("%s failed creating bookmark: %v", l.logPrefix, err)
		}
		records = append(records, r)
		l.lastRead = r.Offset
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil

}

func (l *winEventFileLog) eventHandles(maxRead int) ([]win.EvtHandle, int, error) {
	handles, err := win.EventHandles(l.evtHandle, maxRead, 0)
	switch err {
	case nil:
		if l.maxRead > maxRead {
			debugf("%s Recovered from RPC_S_INVALID_BOUND error (errno 1734) "+
				"by decreasing batch_read_size to %v", l.logPrefix, maxRead)
		}
		return handles, maxRead, nil
	case win.ERROR_NO_MORE_ITEMS:
		detailf("%s No more events", l.logPrefix)
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

func (l *winEventFileLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return win.Close(l.evtHandle)
}

func (l *winEventFileLog) buildRecordFromXML(x []byte, recoveredErr error) (Record, error) {
	e, err := sys.UnmarshalEventXML(x)
	if err != nil {
		return Record{}, fmt.Errorf("Failed to unmarshal XML='%s'. %v", x, err)
	}

	err = sys.PopulateAccount(&e.User)
	if err != nil {
		debugf("%s SID %s account lookup failed. %v", l.logPrefix,
			e.User.Identifier, err)
	}

	if e.RenderErrorCode != 0 {
		// Convert the render error code to an error message that can be
		// included in the "message_error" field.
		e.RenderErr = syscall.Errno(e.RenderErrorCode).Error()
	} else if recoveredErr != nil {
		e.RenderErr = recoveredErr.Error()
	}

	if e.Level == "" {
		// Fallback on LevelRaw if the Level is not set in the RenderingInfo.
		e.Level = win.EventLevel(e.LevelRaw).String()
	}

	if logp.IsDebug(detailSelector) {
		detailf("%s XML=%s Event=%+v", l.logPrefix, string(x), e)
	}

	r := Record{
		API:   winEventLogAPIName,
		Event: e,
	}

	if l.config.IncludeXML {
		r.XML = string(x)
	}

	return r, nil
}

func newWinEvetLogFile(options *common.Config) ([]EventLog, error) {
	c := defaultWinEventLogFileConfig
	if err := readConfig(options, &c, winEventLogFileConfigKeys); err != nil {
		return nil, err
	}

	eventMetadataHandle := func(providerName, sourceName string) sys.MessageFiles {
		mf := sys.MessageFiles{SourceName: sourceName}
		h, err := win.OpenPublisherMetadata(0, sourceName, 0)
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

	//c.Name must be a filepath, allow contains Go glob
	matchFiles, err := filepath.Glob(c.Name)
	if err != nil {
		return nil, err
	}

	retMe := make([]EventLog, 0)
	for _, mf := range matchFiles {
		l := &winEventFileLog{
			config:    c,
			path:      mf,
			maxRead:   c.BatchReadSize,
			renderBuf: make([]byte, renderBufferSize),
			outputBuf: sys.NewByteBuffer(renderBufferSize),
			cache:     newMessageFilesCache(c.Name, eventMetadataHandle, freeHandle),
			logPrefix: fmt.Sprintf("WinEventLogFile[%s]", c.Name),
		}
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEvent(event, 0, l.renderBuf, l.cache.get, out)
		}
		retMe = append(retMe, l)
	}
	return retMe, nil
}

func (l *winEventFileLog) createBookmarkFromEvent(evtHandle win.EvtHandle) (string, error) {
	bmHandle, err := win.CreateBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", err
	}
	l.outputBuf.Reset()
	err = win.RenderBookmarkXML(bmHandle, l.renderBuf, l.outputBuf)
	win.Close(bmHandle)
	return string(l.outputBuf.Bytes()), err
}

func init() {
	// Register wineventlog API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogFileAPIName, 2, newWinEvetLogFile, win.Channels)
	}
}
