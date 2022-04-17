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

package wineventlog

import (
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/menderesk/beats/v7/winlogbeat/sys"
)

// Bookmark is a handle to an event log bookmark.
type Bookmark EvtHandle

// Close closes the bookmark handle.
func (b Bookmark) Close() error {
	return EvtHandle(b).Close()
}

// XML returns the bookmark's value as XML.
func (b Bookmark) XML() (string, error) {
	var bufferUsed uint32

	err := _EvtRender(NilHandle, EvtHandle(b), EvtRenderBookmark, 0, nil, &bufferUsed, nil)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return "", errors.Wrap(err, "failed to determine necessary buffer size for EvtRender")
	}

	bb := sys.NewPooledByteBuffer()
	bb.Reserve(int(bufferUsed * 2))
	defer bb.Free()

	err = _EvtRender(NilHandle, EvtHandle(b), EvtRenderBookmark, uint32(bb.Len()), bb.PtrAt(0), &bufferUsed, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to render bookmark XML")
	}

	return sys.UTF16BytesToString(bb.Bytes())
}

// NewBookmarkFromEvent returns a Bookmark pointing to the given event record.
// The returned handle must be closed.
func NewBookmarkFromEvent(eventHandle EvtHandle) (Bookmark, error) {
	h, err := _EvtCreateBookmark(nil)
	if err != nil {
		return 0, err
	}
	if err = _EvtUpdateBookmark(h, eventHandle); err != nil {
		h.Close()
		return 0, err
	}
	return Bookmark(h), nil
}

// NewBookmarkFromXML returns a Bookmark created from an XML bookmark.
// The returned handle must be closed.
func NewBookmarkFromXML(xml string) (Bookmark, error) {
	utf16, err := syscall.UTF16PtrFromString(xml)
	if err != nil {
		return 0, err
	}
	h, err := _EvtCreateBookmark(utf16)
	return Bookmark(h), err
}
