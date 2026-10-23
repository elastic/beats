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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	quark "github.com/elastic/go-quark"

	"github.com/elastic/beats/v7/libbeat/ebpf/sys"
)

// cgroupRegex captures 64-character lowercase hexadecimal container IDs found in cgroup paths.
var cgroupRegex = regexp.MustCompile(`[-/]([0-9a-f]{64})(\.scope)?$`)

// NewEventFromQuarkEvent creates an Event from a quark file event. It returns
// false when the event has no file payload, targets an excluded path, or does
// not describe a file operation the module reports on.
func NewEventFromQuarkEvent(
	qe quark.Event,
	maxFileSize uint64,
	hashTypes []HashType,
	fileParsers []FileParser,
	isExcludedPath func(string) bool,
) (Event, bool) {
	file := qe.File
	if file == nil || file.Path == "" {
		return Event{}, false
	}
	if isExcludedPath(file.Path) {
		return Event{Path: file.Path}, false
	}

	action := actionFromQuarkFile(file)
	if action == None {
		return Event{Path: file.Path}, false
	}

	var errs []error
	process, err := processFromQuark(qe.Process)
	if err != nil {
		errs = append(errs, err)
	}

	event := Event{
		Timestamp:   time.Now().UTC(),
		Path:        file.Path,
		TargetPath:  file.SymTarget,
		Source:      SourceEBPF,
		Action:      action,
		Process:     &process,
		ContainerID: containerIDFromCgroupPath(qe.Process.Cgroup),
	}

	// A removed file has no metadata to report, mirroring the other backends.
	if action&Deleted == 0 {
		md, err := metadataFromQuarkFile(file)
		if err != nil {
			errs = append(errs, err)
		}
		event.Info = &md

		switch md.Type {
		case FileType:
			fillHashes(&event, file.Path, maxFileSize, hashTypes, fileParsers)
		case SymlinkType:
			event.TargetPath, err = filepath.EvalSymlinks(event.Path)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	event.errors = errs
	return event, true
}

// actionFromQuarkFile maps quark's operation and change masks to an Action.
// quark aggregates operations on the same inode that happen close together, so
// several bits can be set at once. The final state of the file wins: a removal
// or move outranks the create or modifications that preceded it.
func actionFromQuarkFile(file *quark.File) Action {
	switch {
	case file.OpMask&quark.QUARK_FILE_OP_REMOVE != 0:
		return Deleted
	case file.OpMask&quark.QUARK_FILE_OP_MOVE != 0:
		return Moved
	case file.OpMask&quark.QUARK_FILE_OP_CREATE != 0:
		return Created
	case file.OpMask&quark.QUARK_FILE_OP_MODIFY != 0:
		var action Action
		if file.ChangeMask&quark.QUARK_FILE_CH_CONTENT != 0 {
			action |= Updated
		}
		if file.ChangeMask&(quark.QUARK_FILE_CH_PERMS|quark.QUARK_FILE_CH_OWNER|quark.QUARK_FILE_CH_XATTRS) != 0 {
			action |= AttributesModified
		}
		if action == None {
			// quark reported a modification without saying what changed.
			action = Updated
		}
		return action
	default:
		return None
	}
}

func containerIDFromCgroupPath(path string) string {
	matches := cgroupRegex.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// metadataFromQuarkFile fills Metadata from the stat data quark captured in
// the kernel at the time of the event. Mode keeps only the permission bits so
// that file.mode matches what the fsnotify backend reports.
func metadataFromQuarkFile(file *quark.File) (Metadata, error) {
	md := Metadata{
		Inode:  file.Inode,
		UID:    file.Uid,
		GID:    file.Gid,
		Size:   file.Size,
		MTime:  time.Unix(0, int64(file.Mtime)), //nolint:gosec // nanosecond timestamps fit in int64
		CTime:  time.Unix(0, int64(file.Ctime)), //nolint:gosec // nanosecond timestamps fit in int64
		Type:   typeFromStatMode(file.Mode),
		Mode:   os.FileMode(file.Mode) & os.ModePerm,
		SetUID: file.Mode&syscall.S_ISUID != 0,
		SetGID: file.Mode&syscall.S_ISGID != 0,
	}
	fillExtendedAttributes(&md, file.Path)

	u, err := user.LookupId(strconv.FormatUint(uint64(file.Uid), 10))
	if err != nil {
		md.Owner = "n/a"
		md.Group = "n/a"
		return md, err
	}
	md.Owner = u.Username

	g, err := user.LookupGroupId(strconv.FormatUint(uint64(file.Gid), 10))
	if err != nil {
		md.Group = "n/a"
		return md, err
	}
	md.Group = g.Name

	return md, nil
}

// typeFromStatMode maps the file type bits of a stat(2) st_mode to a Type.
func typeFromStatMode(mode uint32) Type {
	switch mode & syscall.S_IFMT {
	case syscall.S_IFREG:
		return FileType
	case syscall.S_IFDIR:
		return DirType
	case syscall.S_IFLNK:
		return SymlinkType
	case syscall.S_IFCHR:
		return CharDeviceType
	case syscall.S_IFBLK:
		return BlockDeviceType
	case syscall.S_IFIFO:
		return FIFOType
	case syscall.S_IFSOCK:
		return SocketType
	default:
		return UnknownType
	}
}

// processFromQuark builds the Process attached to a file event from quark's
// process cache entry. The entity ID uses the same boot-relative start time
// reduction as the rest of auditbeat so IDs stay consistent across sources.
func processFromQuark(p quark.Process) (Process, error) {
	proc := Process{
		PID:  p.Pid,
		Name: p.Comm,
	}
	if !p.Proc.Valid {
		return proc, fmt.Errorf("quark has no process state for pid %d", p.Pid)
	}

	start, err := sys.TimeFromNsSinceBoot(p.Proc.TimeBoot)
	if err != nil {
		return proc, err
	}
	proc.EntityID, err = sys.EntityID(p.Pid, start)
	if err != nil {
		return proc, err
	}

	proc.User.ID = strconv.FormatUint(uint64(p.Proc.Euid), 10)
	if u, err := user.LookupId(proc.User.ID); err == nil {
		proc.User.Name = u.Username
	} else {
		proc.User.Name = "n/a"
	}

	proc.Group.ID = strconv.FormatUint(uint64(p.Proc.Egid), 10)
	if g, err := user.LookupGroupId(proc.Group.ID); err == nil {
		proc.Group.Name = g.Name
	} else {
		proc.Group.Name = "n/a"
	}

	return proc, nil
}
