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

	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

// lastEvent returns the last event held in the system event log.
func lastEvent(name string, work []byte, buf *sys.ByteBuffer) (winevent.Event, error) {
	h, err := win.EvtQuery(0, name, "", win.EvtQueryChannelPath|win.EvtQueryReverseDirection|win.EvtQueryTolerateQueryErrors)
	if err != nil {
		return winevent.Event{}, err
	}
	defer win.Close(h)

	// Seek to first since we have asked for the events in reverse order.
	err = win.EvtSeek(h, 0, 0, win.EvtSeekRelativeToFirst)
	if err != nil {
		return winevent.Event{}, err
	}

	buf.Reset()
	err = win.RenderEventXML(h, work, buf)
	var bufErr sys.InsufficientBufferError
	if errors.As(err, &bufErr) {
		// Don't retain work buffer that are over the 16kiB
		// allocation; we are calling this infrequently, and
		// mostly won't need to work above this value.
		work = make([]byte, bufErr.RequiredSize)
		buf.Reset()
		err = win.RenderEventXML(h, work, buf)
	}
	if err != nil && buf.Len() == 0 {
		return winevent.Event{}, err
	}
	return winevent.UnmarshalXML(buf.Bytes())
}
