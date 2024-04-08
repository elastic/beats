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
				Uid:   uint32(os.Geteuid()),
				Gid:   uint32(os.Getegid()),
			},
			Path:              "/foo",
			SymlinkTargetPath: "/bar",
			Creds: ebpfevents.CredInfo{
				Ruid: 1,
				Rgid: 2,
				Euid: uint32(os.Geteuid()),
				Egid: uint32(os.Getegid()),
				Suid: 5,
				Sgid: 6,
			},
		},
	}
	event, ok := NewEventFromEbpfEvent(
		ebpfEvent, 0, []HashType{}, []FileParser{}, func(path string) bool { return false })
	assert.True(t, ok)

	expectedEvent := Event{
		Action:     Created,
		Path:       "/foo",
		TargetPath: "/bar",
		Info: &Metadata{
			Type:  FileType,
			Inode: 1234,
			UID:   uint32(os.Geteuid()),
			GID:   uint32(os.Getegid()),
			Size:  2345,
			Owner: event.Info.Owner,
			Group: event.Info.Group,
			Mode:  os.FileMode(0o644),
		},
		Process: event.Process, // 1:1 copy this as it changes on every machine
		Source:  SourceEBPF,
		errors:  nil,
	}
	event.Timestamp = expectedEvent.Timestamp

	assert.Equal(t, expectedEvent, event)
	assert.NotEqual(t, "", event.Process.EntityID)
	assert.NotEqual(t, 0, event.Process.PID)
	assert.NotEqual(t, 0, event.Process.User.ID)
	assert.NotEqual(t, "", event.Process.User.Name)
}
