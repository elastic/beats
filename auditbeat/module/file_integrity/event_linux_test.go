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

//go:build linux

package file_integrity

import (
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/ebpfevents"
)

func TestNewEventFromEbpfEvent(t *testing.T) {
	ebpfEvent := ebpfevents.Event{
		Header: ebpfevents.Header{
			Type: ebpfevents.EventTypeFileCreate,
		},
		Body: &ebpfevents.FileCreate{
			Finfo: ebpfevents.FileInfo{
				Type:  ebpfevents.FileTypeFile,
				Inode: 1234,
				Mode:  os.FileMode(0o644),
				Size:  2345,
				Uid:   3456,
				Gid:   4567,
			},
			Path:              "/foo",
			SymlinkTargetPath: "/bar",
		},
	}
	expectedEvent := Event{
		Action:     Created,
		Path:       "/foo",
		TargetPath: "/bar",
		Info: &Metadata{
			Type:  FileType,
			Inode: 1234,
			UID:   3456,
			GID:   4567,
			Size:  2345,
			Owner: "n/a",
			Group: "n/a",
			Mode:  os.FileMode(0o644),
		},
		Source: SourceEBPF,
		errors: []error{user.UnknownUserIdError(3456)},
	}

	event, ok := NewEventFromEbpfEvent(
		ebpfEvent, 0, []HashType{}, []FileParser{}, func(path string) bool { return false })
	assert.True(t, ok)
	event.Timestamp = expectedEvent.Timestamp

	assert.Equal(t, expectedEvent, event)
}
