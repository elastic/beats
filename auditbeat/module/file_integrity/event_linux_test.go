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

//go:build linux && (amd64 || arm64) && cgo

package file_integrity

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	quark "github.com/elastic/go-quark"
)

const testContainerID = "d12fe576354a1805165303a4e34a69e5fe8db791ceb7e545f17811d1fbfba68f"

func quarkFileEvent(file *quark.File) quark.Event {
	return quark.Event{
		Events: quark.QUARK_EV_FILE,
		Process: quark.Process{
			Pid:    uint32(os.Getpid()), //nolint:gosec // pids fit in uint32
			Comm:   "auditbeat",
			Cgroup: "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/cri-containerd-" + testContainerID + ".scope",
			Proc: quark.Proc{
				Valid:    true,
				TimeBoot: 1_000_000_000,
				Euid:     uint32(os.Geteuid()), //nolint:gosec // uids fit in uint32
				Egid:     uint32(os.Getegid()), //nolint:gosec // gids fit in uint32
			},
		},
		File: file,
	}
}

func notExcluded(string) bool { return false }

func TestNewEventFromQuarkEventCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644), "write test file")

	qe := quarkFileEvent(&quark.File{
		Path:      path,
		SymTarget: "/bar",
		Inode:     1234,
		Mode:      syscall.S_IFREG | 0o644,
		Size:      5,
		Uid:       uint32(os.Geteuid()), //nolint:gosec // uids fit in uint32
		Gid:       uint32(os.Getegid()), //nolint:gosec // gids fit in uint32
		Mtime:     1_700_000_000_000_000_000,
		Ctime:     1_700_000_000_000_000_001,
		OpMask:    quark.QUARK_FILE_OP_CREATE,
	})

	event, ok := NewEventFromQuarkEvent(qe, 0, []HashType{}, []FileParser{}, notExcluded)
	require.True(t, ok, "event should be produced")

	assert.Equal(t, path, event.Path, "path")
	assert.Equal(t, "/bar", event.TargetPath, "target path")
	assert.Equal(t, Action(Created), event.Action, "action")
	assert.Equal(t, SourceEBPF, event.Source, "source")
	assert.Equal(t, testContainerID, event.ContainerID, "container id")

	require.NotNil(t, event.Info, "metadata")
	assert.Equal(t, uint64(1234), event.Info.Inode, "inode")
	assert.Equal(t, FileType, event.Info.Type, "type")
	assert.Equal(t, os.FileMode(0o644), event.Info.Mode, "mode keeps only permission bits")
	assert.False(t, event.Info.SetUID, "setuid")
	assert.False(t, event.Info.SetGID, "setgid")
	assert.Equal(t, uint64(5), event.Info.Size, "size")
	assert.Equal(t, int64(1_700_000_000_000_000_000), event.Info.MTime.UnixNano(), "mtime")
	assert.Equal(t, int64(1_700_000_000_000_000_001), event.Info.CTime.UnixNano(), "ctime")
	assert.NotEqual(t, "n/a", event.Info.Owner, "owner resolved")
	assert.NotEqual(t, "n/a", event.Info.Group, "group resolved")

	require.NotNil(t, event.Process, "process")
	assert.Equal(t, uint32(os.Getpid()), event.Process.PID, "pid") //nolint:gosec // pids fit in uint32
	assert.Equal(t, "auditbeat", event.Process.Name, "process name")
	assert.NotEmpty(t, event.Process.EntityID, "entity id")
	assert.Equal(t, uint32(os.Geteuid()), uint32(mustAtoi(t, event.Process.User.ID)), "euid")  //nolint:gosec // uids fit in uint32
	assert.Equal(t, uint32(os.Getegid()), uint32(mustAtoi(t, event.Process.Group.ID)), "egid") //nolint:gosec // gids fit in uint32
	assert.NotEqual(t, "n/a", event.Process.User.Name, "user name resolved")
	assert.NotEqual(t, "n/a", event.Process.Group.Name, "group name resolved")
	assert.Empty(t, event.errors, "no errors expected")
}

func TestNewEventFromQuarkEventDelete(t *testing.T) {
	qe := quarkFileEvent(&quark.File{
		Path:   "/gone",
		Mode:   syscall.S_IFREG | 0o644,
		OpMask: quark.QUARK_FILE_OP_CREATE | quark.QUARK_FILE_OP_MODIFY | quark.QUARK_FILE_OP_REMOVE,
	})

	event, ok := NewEventFromQuarkEvent(qe, 0, []HashType{}, []FileParser{}, notExcluded)
	require.True(t, ok, "event should be produced")
	assert.Equal(t, Action(Deleted), event.Action, "removal outranks aggregated create/modify")
	assert.Nil(t, event.Info, "deleted files carry no metadata")
	assert.Nil(t, event.Hashes, "deleted files are not hashed")
}

func TestNewEventFromQuarkEventExcluded(t *testing.T) {
	qe := quarkFileEvent(&quark.File{Path: "/excluded/foo", OpMask: quark.QUARK_FILE_OP_CREATE})

	event, ok := NewEventFromQuarkEvent(qe, 0, nil, nil, func(string) bool { return true })
	assert.False(t, ok, "excluded paths produce no event")
	assert.Equal(t, "/excluded/foo", event.Path, "path is still reported for logging")

	_, ok = NewEventFromQuarkEvent(quark.Event{Events: quark.QUARK_EV_FILE}, 0, nil, nil, notExcluded)
	assert.False(t, ok, "events without a file payload are dropped")
}

func TestNewEventFromQuarkEventUnknownProcess(t *testing.T) {
	qe := quarkFileEvent(&quark.File{Path: "/gone", OpMask: quark.QUARK_FILE_OP_REMOVE})
	qe.Process.Proc.Valid = false

	event, ok := NewEventFromQuarkEvent(qe, 0, nil, nil, notExcluded)
	require.True(t, ok, "event should still be produced")
	require.NotNil(t, event.Process, "process")
	assert.Equal(t, uint32(os.Getpid()), event.Process.PID, "pid is known even without cached state") //nolint:gosec // pids fit in uint32
	assert.Empty(t, event.Process.EntityID, "entity id needs the process start time")
	assert.Len(t, event.errors, 1, "missing process state is recorded as an error")
}

func TestActionFromQuarkFile(t *testing.T) {
	tests := []struct {
		name       string
		opMask     uint32
		changeMask uint32
		want       Action
	}{
		{name: "create", opMask: quark.QUARK_FILE_OP_CREATE, want: Created},
		{name: "move", opMask: quark.QUARK_FILE_OP_MOVE, want: Moved},
		{name: "remove", opMask: quark.QUARK_FILE_OP_REMOVE, want: Deleted},
		{name: "modify content", opMask: quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_CONTENT, want: Updated},
		{name: "modify perms", opMask: quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_PERMS, want: AttributesModified},
		{name: "modify owner", opMask: quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_OWNER, want: AttributesModified},
		{name: "modify xattrs", opMask: quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_XATTRS, want: AttributesModified},
		{name: "modify content and perms", opMask: quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_CONTENT | quark.QUARK_FILE_CH_PERMS, want: Updated | AttributesModified},
		{name: "modify without change mask", opMask: quark.QUARK_FILE_OP_MODIFY, want: Updated},
		{name: "aggregated create and modify", opMask: quark.QUARK_FILE_OP_CREATE | quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_CONTENT, want: Created},
		{name: "aggregated move and modify", opMask: quark.QUARK_FILE_OP_MOVE | quark.QUARK_FILE_OP_MODIFY, changeMask: quark.QUARK_FILE_CH_CONTENT, want: Moved},
		{name: "aggregated create and remove", opMask: quark.QUARK_FILE_OP_CREATE | quark.QUARK_FILE_OP_REMOVE, want: Deleted},
		{name: "no op", want: None},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := actionFromQuarkFile(&quark.File{OpMask: tc.opMask, ChangeMask: tc.changeMask})
			assert.Equal(t, tc.want, got, "action for op=%#x change=%#x", tc.opMask, tc.changeMask)
		})
	}
}

func TestTypeFromStatMode(t *testing.T) {
	tests := map[uint32]Type{
		syscall.S_IFREG | 0o644:  FileType,
		syscall.S_IFDIR | 0o755:  DirType,
		syscall.S_IFLNK | 0o777:  SymlinkType,
		syscall.S_IFCHR | 0o620:  CharDeviceType,
		syscall.S_IFBLK | 0o660:  BlockDeviceType,
		syscall.S_IFIFO | 0o600:  FIFOType,
		syscall.S_IFSOCK | 0o755: SocketType,
		0o644:                    UnknownType,
	}
	for mode, want := range tests {
		assert.Equal(t, want, typeFromStatMode(mode), "type for mode %#o", mode)
	}
}

func TestMetadataFromQuarkFileSetuid(t *testing.T) {
	md, _ := metadataFromQuarkFile(&quark.File{
		Path: filepath.Join(t.TempDir(), "missing"),
		Mode: syscall.S_IFREG | syscall.S_ISUID | syscall.S_ISGID | 0o755,
	})
	assert.True(t, md.SetUID, "setuid bit")
	assert.True(t, md.SetGID, "setgid bit")
	assert.Equal(t, os.FileMode(0o755), md.Mode, "mode excludes the setuid/setgid bits")
}

func TestContainerIDFromCgroupPath(t *testing.T) {
	tests := map[string]string{
		"/kubepods.slice/kubepods-burstable.slice/cri-containerd-" + testContainerID + ".scope": testContainerID,
		"/docker/" + testContainerID:                  testContainerID,
		"/user.slice/user-1000.slice/session-5.scope": "",
		"": "",
	}
	for path, want := range tests {
		assert.Equal(t, want, containerIDFromCgroupPath(path), "container id for %q", path)
	}
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	var n int
	for _, c := range s {
		require.True(t, c >= '0' && c <= '9', "expected numeric id, got %q", s)
		n = n*10 + int(c-'0')
	}
	return n
}
