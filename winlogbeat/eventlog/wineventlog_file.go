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
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/sys"
	win "github.com/elastic/beats/winlogbeat/sys/wineventlog"
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
	BatchReadSize: 100,
}

type winEventFileLog struct {
	winEventLog
	config winEventLogFileConfig
	path   string
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
	l.subscription = queryHandle
	err = win.EvtSeek(l.subscription, 0, bookmark)
	// An error here occurrs if the bookmark was for an old log file that has been rotated.
	if err != nil {
		state.Bookmark = ""
		state.RecordNumber = 0
	}

	return nil
}

func (l *winEventFileLog) Read() ([]Record, error) {
	var handles, err = l.winEventLog.Read()
	if len(handles) == 0 {
		// If we have read everything from this log file, give other applications the ability to rotate log files if required.
		l.Close()
		time.Sleep(time.Second)
		l.Open(l.lastRead)
	}
	return handles, err
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

	logs := make([]EventLog, 0)
	for _, mf := range matchFiles {
		l := &winEventFileLog{
			config: c,
			path:   mf,
			winEventLog: winEventLog{
				channelName: mf,
				maxRead:     c.BatchReadSize,
				renderBuf:   make([]byte, renderBufferSize),
				outputBuf:   sys.NewByteBuffer(renderBufferSize),
				cache:       newMessageFilesCache(c.Name, eventMetadataHandle, freeHandle),
				logPrefix:   fmt.Sprintf("WinEventLogFile[%s]", c.Name),
			},
		}
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEvent(event, 0, l.renderBuf, l.cache.get, out)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func init() {
	// Register wineventlog API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogFileAPIName, 2, newWinEvetLogFile, win.Channels)
	}
}
