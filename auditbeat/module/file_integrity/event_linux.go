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
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/ebpf/sys"
	"github.com/elastic/ebpfevents"
)

// cgroupRegex captures 64-character lowercase hexadecimal container IDs found in cgroup paths.
var cgroupRegex = regexp.MustCompile(`[-/]([0-9a-f]{64})(\.scope)?$`)

// NewEventFromEbpfEvent creates a new Event from an ebpfevents.Event.
func NewEventFromEbpfEvent(
	ee ebpfevents.Event,
	maxFileSize uint64,
	hashTypes []HashType,
	fileParsers []FileParser,
	isExcludedPath func(string) bool,
) (Event, bool) {
	var (
		path, target, cgroupPath string
		action       Action
		metadata     Metadata
		process      Process
		err          error
		errors       = make([]error, 0)
	)
	switch ee.Type {
	case ebpfevents.EventTypeFileCreate:
		action = Created

		fileCreateEvent := ee.Body.(*ebpfevents.FileCreate)
		path = fileCreateEvent.Path
		if isExcludedPath(path) {
			event := Event{Path: path}
			return event, false
		}
		target = fileCreateEvent.SymlinkTargetPath

		metadata, err = metadataFromFileCreate(fileCreateEvent)
		if err != nil {
			errors = append(errors, err)
		}

		process, err = processFromFileCreate(fileCreateEvent)
		if err != nil {
			errors = append(errors, err)
		}

		cgroupPath = fileCreateEvent.CgroupPath
	case ebpfevents.EventTypeFileRename:
		action = Moved

		fileRenameEvent := ee.Body.(*ebpfevents.FileRename)
		path = fileRenameEvent.NewPath
		if isExcludedPath(path) {
			event := Event{Path: path}
			return event, false
		}
		target = fileRenameEvent.SymlinkTargetPath

		metadata, err = metadataFromFileRename(fileRenameEvent)
		if err != nil {
			errors = append(errors, err)
		}

		process, err = processFromFileRename(fileRenameEvent)
		if err != nil {
			errors = append(errors, err)
		}

		cgroupPath = fileRenameEvent.CgroupPath
	case ebpfevents.EventTypeFileDelete:
		action = Deleted

		fileDeleteEvent := ee.Body.(*ebpfevents.FileDelete)
		path = fileDeleteEvent.Path
		if isExcludedPath(path) {
			event := Event{Path: path}
			return event, false
		}
		target = fileDeleteEvent.SymlinkTargetPath

		process, err = processFromFileDelete(fileDeleteEvent)
		if err != nil {
			errors = append(errors, err)
		}

		cgroupPath = fileDeleteEvent.CgroupPath
	case ebpfevents.EventTypeFileModify:
		fileModifyEvent := ee.Body.(*ebpfevents.FileModify)

		switch fileModifyEvent.ChangeType {
		case ebpfevents.FileChangeTypeContent:
			action = Updated
		case ebpfevents.FileChangeTypePermissions, ebpfevents.FileChangeTypeOwner, ebpfevents.FileChangeTypeXattrs:
			action = AttributesModified
		}

		path = fileModifyEvent.Path
		if isExcludedPath(path) {
			event := Event{Path: path}
			return event, false
		}
		target = fileModifyEvent.SymlinkTargetPath

		metadata, err = metadataFromFileModify(fileModifyEvent)
		if err != nil {
			errors = append(errors, err)
		}

		process, err = processFromFileModify(fileModifyEvent)
		if err != nil {
			errors = append(errors, err)
		}

		cgroupPath = fileModifyEvent.CgroupPath
	}

	event := Event{
		Timestamp:   time.Now().UTC(),
		Path:        path,
		TargetPath:  target,
		Info:        &metadata,
		Source:      SourceEBPF,
		Action:      action,
		Process:     &process,
		ContainerID: containerIDFromCgroupPath(cgroupPath),
		errors:      errors,
	}

	if event.Action == Deleted {
		event.Info = nil
	} else {
		switch event.Info.Type {
		case FileType:
			fillHashes(&event, path, maxFileSize, hashTypes, fileParsers)
		case SymlinkType:
			event.TargetPath, err = filepath.EvalSymlinks(event.Path)
			if err != nil {
				event.errors = append(event.errors, err)
			}
		}
	}

	return event, true
}

func containerIDFromCgroupPath(path string) string {
	matches := cgroupRegex.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func metadataFromFileCreate(evt *ebpfevents.FileCreate) (Metadata, error) {
	var md Metadata
	fillExtendedAttributes(&md, evt.Path)
	err := fillFileInfo(&md, evt.Finfo)
	return md, err
}

func metadataFromFileRename(evt *ebpfevents.FileRename) (Metadata, error) {
	var md Metadata
	fillExtendedAttributes(&md, evt.NewPath)
	err := fillFileInfo(&md, evt.Finfo)
	return md, err
}

func metadataFromFileModify(evt *ebpfevents.FileModify) (Metadata, error) {
	var md Metadata
	fillExtendedAttributes(&md, evt.Path)
	err := fillFileInfo(&md, evt.Finfo)
	return md, err
}

func newProcess(pid uint32, start uint64, comm string, euid, egid uint32) (Process, error) {
	var (
		p   Process
		err error
	)

	t, err := sys.TimeFromNsSinceBoot(start)
	if err != nil {
		return p, err
	}

	p.EntityID, err = sys.EntityID(pid, t)
	if err != nil {
		return p, err
	}
	p.Name = comm
	p.PID = pid

	p.User.ID = strconv.FormatUint(uint64(euid), 10)
	u, err := user.LookupId(p.User.ID)
	if err == nil {
		p.User.Name = u.Username
	} else {
		p.User.Name = "n/a"
	}

	p.Group.ID = strconv.FormatUint(uint64(egid), 10)
	g, err := user.LookupGroupId(p.Group.ID)
	if err == nil {
		p.Group.Name = g.Name
	} else {
		p.Group.Name = "n/a"
	}

	return p, nil
}

func processFromFileCreate(evt *ebpfevents.FileCreate) (Process, error) {
	return newProcess(evt.Pids.Tgid, evt.Pids.StartTimeNs, evt.Comm, evt.Creds.Euid, evt.Creds.Egid)
}

func processFromFileRename(evt *ebpfevents.FileRename) (Process, error) {
	return newProcess(evt.Pids.Tgid, evt.Pids.StartTimeNs, evt.Comm, evt.Creds.Euid, evt.Creds.Egid)
}

func processFromFileModify(evt *ebpfevents.FileModify) (Process, error) {
	return newProcess(evt.Pids.Tgid, evt.Pids.StartTimeNs, evt.Comm, evt.Creds.Euid, evt.Creds.Egid)
}

func processFromFileDelete(evt *ebpfevents.FileDelete) (Process, error) {
	return newProcess(evt.Pids.Tgid, evt.Pids.StartTimeNs, evt.Comm, evt.Creds.Euid, evt.Creds.Egid)
}

func fillFileInfo(md *Metadata, finfo ebpfevents.FileInfo) error {
	md.Inode = finfo.Inode
	md.UID = finfo.Uid
	md.GID = finfo.Gid
	md.Size = finfo.Size
	md.MTime = finfo.Mtime
	md.CTime = finfo.Ctime
	md.Type = typeFromEbpfType(finfo.Type)
	md.Mode = finfo.Mode
	md.SetUID = finfo.Mode&os.ModeSetuid != 0
	md.SetGID = finfo.Mode&os.ModeSetgid != 0

	u, err := user.LookupId(strconv.FormatUint(uint64(finfo.Uid), 10))
	if err != nil {
		md.Owner = "n/a"
		md.Group = "n/a"
		return err
	}
	md.Owner = u.Username

	g, err := user.LookupGroupId(strconv.FormatUint(uint64(finfo.Gid), 10))
	if err != nil {
		md.Group = "n/a"
		return err
	}
	md.Group = g.Name

	return nil
}

func typeFromEbpfType(typ ebpfevents.FileType) Type {
	switch typ {
	case ebpfevents.FileTypeFile:
		return FileType
	case ebpfevents.FileTypeDir:
		return DirType
	case ebpfevents.FileTypeSymlink:
		return SymlinkType
	case ebpfevents.FileTypeCharDevice:
		return CharDeviceType
	case ebpfevents.FileTypeBlockDevice:
		return BlockDeviceType
	case ebpfevents.FileTypeNamedPipe:
		return FIFOType
	case ebpfevents.FileTypeSocket:
		return SocketType
	default:
		return UnknownType
	}
}
