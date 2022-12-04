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
	"errors"
	"io"

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

// oldestEvent returns the oldest event held in the system event log.
func oldestEvent(work []byte, buf *sys.ByteBuffer) (winevent.Event, error) {
	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return winevent.Event{}, err
	}
	defer windows.CloseHandle(event)

	s, err := win.Subscribe(0, event, "", "", 0, win.EvtSubscribeStartAtOldestRecord)
	if err != nil {
		return winevent.Event{}, err
	}

	h, err := win.EventHandles(s, 1)
	switch err { //nolint:errorlint // This is an errno or nil.
	case nil:
	case win.ERROR_NO_MORE_ITEMS:
		// Shim to error that is not guarded by a go:build windows directive.
		return winevent.Event{}, io.EOF
	default:
		return winevent.Event{}, err
	}

	buf.Reset()
	err = win.RenderEventXML(h[0], work, buf)
	var bufErr sys.InsufficientBufferError
	if errors.As(err, &bufErr) {
		// Don't retain work buffer that are over the 16kiB
		// allocation; we are calling this infrequently, and
		// mostly won't need to work above this value.
		work = make([]byte, bufErr.RequiredSize)
		buf.Reset()
		err = win.RenderEventXML(h[0], work, buf)
	}
	if err != nil && buf.Len() == 0 {
		return winevent.Event{}, err
	}
	return winevent.UnmarshalXML(buf.Bytes())
}
