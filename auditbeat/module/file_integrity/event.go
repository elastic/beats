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

package file_integrity

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Source identifies the source of an event (i.e. what triggered it).
type Source uint8

func (s Source) String() string {
	if name, found := sourceNames[s]; found {
		return name
	}
	return "unknown"
}

// MarshalText marshals the Source to a textual representation of itself.
func (s Source) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

const (
	// SourceScan identifies events triggered by a file system scan.
	SourceScan Source = iota
	// SourceFSNotify identifies events triggered by a notification from the
	// file system.
	SourceFSNotify
	// SourceEBPF identifies events triggered by an eBPF program.
	SourceEBPF
)

var sourceNames = map[Source]string{
	SourceScan:     "scan",
	SourceFSNotify: "fsnotify",
	SourceEBPF:     "ebpf",
}

// Type identifies the file type (e.g. dir, file, symlink).
type Type uint8

func (t Type) String() string {
	if name, found := typeNames[t]; found {
		return name
	}
	return "unknown"
}

// MarshalText marshals the Type to a textual representation of itself.
func (t Type) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// Enum of possible file.Types.
const (
	UnknownType Type = iota // Typically seen in deleted notifications where the object is gone.
	FileType
	DirType
	SymlinkType
	CharDeviceType
	BlockDeviceType
	FIFOType
	SocketType
)

var typeNames = map[Type]string{
	FileType:        "file",
	DirType:         "dir",
	SymlinkType:     "symlink",
	CharDeviceType:  "char_device",
	BlockDeviceType: "block_device",
	FIFOType:        "fifo",
	SocketType:      "socket",
}

// Digest is an output of a hash function.
type Digest []byte

// String returns the digest value in lower-case hexadecimal form.
func (d Digest) String() string {
	return hex.EncodeToString(d)
}

// MarshalText encodes the digest to a hexadecimal representation of itself.
func (d Digest) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// Event describes the filesystem change and includes metadata about the file.
type Event struct {
	Timestamp     time.Time           `json:"timestamp"`             // Time of event.
	Path          string              `json:"path"`                  // The path associated with the event.
	TargetPath    string              `json:"target_path,omitempty"` // Target path for symlinks.
	Info          *Metadata           `json:"info"`                  // File metadata (if the file exists).
	Source        Source              `json:"source"`                // Source of the event.
	Action        Action              `json:"action"`                // Action (like created, updated).
	Hashes        map[HashType]Digest `json:"hash,omitempty"`        // File hashes.
	ParserResults mapstr.M            `json:"file,omitempty"`        // Results from running file parsers.

	// Metadata
	rtt        time.Duration // Time taken to collect the info.
	errors     []error       // Errors that occurred while collecting the info.
	hashFailed bool          // Set when hashing the file failed.
}

// Metadata contains file metadata.
type Metadata struct {
	Inode          uint64      `json:"inode"`
	UID            uint32      `json:"uid"`
	GID            uint32      `json:"gid"`
	SID            string      `json:"sid"`
	Owner          string      `json:"owner"`
	Group          string      `json:"group"`
	Size           uint64      `json:"size"`
	MTime          time.Time   `json:"mtime"`            // Last modification time.
	CTime          time.Time   `json:"ctime"`            // Last metadata change time.
	Type           Type        `json:"type"`             // File type (dir, file, symlink).
	Mode           os.FileMode `json:"mode"`             // Permissions
	SetUID         bool        `json:"setuid"`           // setuid bit (POSIX only)
	SetGID         bool        `json:"setgid"`           // setgid bit (POSIX only)
	Origin         []string    `json:"origin"`           // External origin info for the file (macOS only)
	SELinux        string      `json:"selinux"`          // security.selinux xattr value (Linux only)
	POSIXACLAccess []byte      `json:"posix_acl_access"` // system.posix_acl_access xattr value (Linux only)
}

// NewEventFromFileInfo creates a new Event based on data from a os.FileInfo
// object that has already been created. Any errors that occur are included in
// the returned Event.
func NewEventFromFileInfo(
	path string,
	info os.FileInfo,
	err error,
	action Action,
	source Source,
	maxFileSize uint64,
	hashTypes []HashType,
	fileParsers []FileParser,
) Event {
	event := Event{
		Timestamp: time.Now().UTC(),
		Path:      path,
		Action:    action,
		Source:    source,
	}

	// err indicates that info is invalid.
	if err != nil {
		event.errors = append(event.errors, err)
		return event
	}

	// Deleted events will not have file info.
	if info == nil {
		return event
	}

	event.Info, err = NewMetadata(path, info)
	if err != nil {
		event.errors = append(event.errors, err)
	}
	if event.Info == nil {
		// This should never happen (only a change in Go could cause it).
		return event
	}

	switch event.Info.Type {
	case FileType:
		fillHashes(&event, path, maxFileSize, hashTypes, fileParsers)
	case SymlinkType:
		event.TargetPath, err = filepath.EvalSymlinks(event.Path)
		if err != nil {
			event.errors = append(event.errors, err)
		}
	}

	return event
}

func fillHashes(event *Event, path string, maxFileSize uint64, hashTypes []HashType, fileParsers []FileParser) {
	if event.Info.Size <= maxFileSize {
		hashes, nbytes, err := hashFile(event.Path, maxFileSize, hashTypes...)
		if err != nil {
			event.errors = append(event.errors, err)
			event.hashFailed = true
		} else if hashes != nil {
			// hashFile returns nil hashes and no error when:
			// - There's no hashes configured.
			// - File size at the time of hashing is larger than configured limit.
			event.Hashes = hashes
			event.Info.Size = nbytes
		}

		if len(fileParsers) != 0 && event.ParserResults == nil {
			event.ParserResults = make(mapstr.M)
		}
		for _, p := range fileParsers {
			if err = p.Parse(event.ParserResults, path); err != nil {
				event.errors = append(event.errors, err)
			}
		}
	}
}

// NewEvent creates a new Event. Any errors that occur are included in the
// returned Event.
func NewEvent(
	path string,
	action Action,
	source Source,
	maxFileSize uint64,
	hashTypes []HashType,
	fileParsers []FileParser,
) Event {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// deleted file is signaled by info == nil
			err = nil
		} else {
			err = fmt.Errorf("failed to lstat: %w", err)
		}
	}
	return NewEventFromFileInfo(path, info, err, action, source, maxFileSize, hashTypes, fileParsers)
}

func isASCIILetter(letter byte) bool {
	// It appears that Windows only allows ascii characters for drive letters
	// and that's what go checks for: https://golang.org/src/path/filepath/path_windows.go#L63
	// **If** Windows/go ever return multibyte utf16 characters we'll need to change
	// the drive letter mapping logic.
	return (letter >= 'a' && letter <= 'z') || (letter >= 'A' && letter <= 'Z')
}

func getDriveLetter(path string) string {
	volume := filepath.VolumeName(path)
	if len(volume) == 2 && volume[1] == ':' {
		if isASCIILetter(volume[0]) {
			return strings.ToUpper(volume[:1])
		}
	}
	return ""
}

func buildMetricbeatEvent(e *Event, existedBefore bool) mb.Event {
	file := mapstr.M{
		"path": e.Path,
	}
	out := mb.Event{
		Timestamp: e.Timestamp,
		Took:      e.rtt,
		MetricSetFields: mapstr.M{
			"file": file,
		},
	}

	if e.TargetPath != "" {
		file["target_path"] = e.TargetPath
	}

	if e.Info != nil {
		info := e.Info
		file["inode"] = strconv.FormatUint(info.Inode, 10)
		file["mtime"] = info.MTime
		file["ctime"] = info.CTime

		if e.Info.Type == FileType {
			if extension := filepath.Ext(e.Path); extension != "" {
				file["extension"] = strings.TrimLeft(extension, ".")
			}
			if mimeType := getMimeType(e.Path); mimeType != "" {
				file["mime_type"] = mimeType
			}
			file["size"] = info.Size
		}

		if info.Type != UnknownType {
			file["type"] = info.Type.String()
		}

		if runtime.GOOS == "windows" {
			if drive := getDriveLetter(e.Path); drive != "" {
				file["drive_letter"] = drive
			}
			if info.SID != "" {
				file["uid"] = info.SID
			}
		} else {
			file["uid"] = strconv.Itoa(int(info.UID))
			file["gid"] = strconv.Itoa(int(info.GID))
			file["mode"] = fmt.Sprintf("%#04o", uint32(info.Mode))
		}

		if info.Owner != "" {
			file["owner"] = info.Owner
		}
		if info.Group != "" {
			file["group"] = info.Group
		}
		if info.SetUID {
			file["setuid"] = true
		}
		if info.SetGID {
			file["setgid"] = true
		}
		if len(info.Origin) > 0 {
			file["origin"] = info.Origin
		}
		if info.SELinux != "" {
			file["selinux"] = info.SELinux
		}
		if len(info.POSIXACLAccess) != 0 {
			a, err := aclText(info.POSIXACLAccess)
			if err == nil {
				file["posix_acl_access"] = a
			}
		}
	}

	if len(e.Hashes) > 0 {
		hashes := make(mapstr.M, len(e.Hashes))
		for hashType, digest := range e.Hashes {
			hashes[string(hashType)] = digest
		}
		file["hash"] = hashes
	}
	for k, v := range e.ParserResults {
		file[k] = v
	}

	out.MetricSetFields.Put("event.kind", "event")
	out.MetricSetFields.Put("event.category", []string{"file"})
	if e.Action > 0 {
		actions := e.Action.InOrder(existedBefore, e.Info != nil)
		out.MetricSetFields.Put("event.type", actions.ECSTypes())
		out.MetricSetFields.Put("event.action", actions.StringArray())
	} else {
		out.MetricSetFields.Put("event.type", None.ECSTypes())
	}

	if n := len(e.errors); n > 0 {
		errors := make([]string, n)
		for idx, err := range e.errors {
			errors[idx] = err.Error()
		}
		if n == 1 {
			out.MetricSetFields.Put("error.message", errors[0])
		} else {
			out.MetricSetFields.Put("error.message", errors)
		}
	}
	return out
}

func aclText(b []byte) ([]string, error) {
	if (len(b)-4)%8 != 0 {
		return nil, fmt.Errorf("unexpected ACL length: %d", len(b))
	}
	b = b[4:] // The first four bytes is the version, discard it.
	a := make([]string, 0, len(b)/8)
	for len(b) != 0 {
		tag := binary.LittleEndian.Uint16(b)
		perm := binary.LittleEndian.Uint16(b[2:])
		qual := binary.LittleEndian.Uint32(b[4:])
		a = append(a, fmt.Sprintf("%s:%s:%s", tags[tag], qualString(qual, tag), modeString(perm)))
		b = b[8:]
	}
	return a, nil
}

var tags = map[uint16]string{
	0x00: "undefined",
	0x01: "user",
	0x02: "user",
	0x04: "group",
	0x08: "group",
	0x10: "mask",
	0x20: "other",
}

func qualString(qual uint32, tag uint16) string {
	if qual == math.MaxUint32 {
		// 0xffffffff is undefined ID, so return zero.
		return ""
	}
	const (
		tagUser  = 0x02
		tagGroup = 0x08
	)
	id := strconv.Itoa(int(qual))
	switch tag {
	case tagUser:
		u, err := user.LookupId(id)
		if err == nil {
			return u.Username
		}
	case tagGroup:
		g, err := user.LookupGroupId(id)
		if err == nil {
			return g.Name
		}
	}
	// Fallback to the numeric ID if we can't get a name
	// or the tag is other than user/group.
	return id
}

func modeString(perm uint16) string {
	var buf [3]byte
	w := 0
	const rwx = "rwx"
	for i, c := range rwx {
		if perm&(1<<uint(len(rwx)-1-i)) != 0 {
			buf[w] = byte(c)
		} else {
			buf[w] = '-'
		}
		w++
	}
	return string(buf[:w])
}

// diffEvents returns true if the file info differs between the old event and
// the new event. Changes to the timestamp and action are ignored. If old
// contains a superset of new's hashes then false is returned.
func diffEvents(old, new *Event) (Action, bool) {
	if old == new {
		return None, false
	}

	if old == nil && new != nil {
		return Created, true
	}

	if old != nil && new == nil {
		return Deleted, true
	}

	if old.Path != new.Path {
		return Moved, true
	}

	result := None

	// Test if new.Hashes is a subset of old.Hashes.
	hasAllHashes := true
	for hashType, newValue := range new.Hashes {

		oldValue, found := old.Hashes[hashType]
		if !found {
			hasAllHashes = false
			continue
		}

		// The Updated action takes precedence over a new hash type being configured.
		if !bytes.Equal(oldValue, newValue) {
			result |= Updated
			break
		}
	}

	if old.TargetPath != new.TargetPath ||
		(old.Info == nil && new.Info != nil) ||
		(old.Info != nil && new.Info == nil) {
		result |= AttributesModified
	}

	// Test if metadata has changed.
	if o, n := old.Info, new.Info; o != nil && n != nil {
		// The owner and group names are ignored (they aren't persisted).
		if o.Inode != n.Inode || o.UID != n.UID || o.GID != n.GID || o.SID != n.SID ||
			o.Mode != n.Mode || o.Type != n.Type || o.SetUID != n.SetUID || o.SetGID != n.SetGID ||
			o.SELinux != n.SELinux || !bytes.Equal(o.POSIXACLAccess, n.POSIXACLAccess) {
			result |= AttributesModified
		}

		// For files consider mtime and size.
		if n.Type == FileType && (!o.MTime.Equal(n.MTime) || o.Size != n.Size) {
			result |= AttributesModified
		}
	}

	// The old event didn't have all the requested hash types.
	if !hasAllHashes {
		result |= ConfigChange
	}

	return result, result != None
}

func hashFile(name string, maxSize uint64, hashType ...HashType) (nameToHash map[HashType]Digest, nbytes uint64, err error) {
	if len(hashType) == 0 {
		return nil, 0, nil
	}

	var hashes []hash.Hash
	for _, name := range hashType {
		switch name {
		case BLAKE2B_256:
			h, _ := blake2b.New256(nil)
			hashes = append(hashes, h)
		case BLAKE2B_384:
			h, _ := blake2b.New384(nil)
			hashes = append(hashes, h)
		case BLAKE2B_512:
			h, _ := blake2b.New512(nil)
			hashes = append(hashes, h)
		case MD5:
			hashes = append(hashes, md5.New())
		case SHA1:
			hashes = append(hashes, sha1.New())
		case SHA224:
			hashes = append(hashes, sha256.New224())
		case SHA256:
			hashes = append(hashes, sha256.New())
		case SHA384:
			hashes = append(hashes, sha512.New384())
		case SHA3_224:
			hashes = append(hashes, sha3.New224())
		case SHA3_256:
			hashes = append(hashes, sha3.New256())
		case SHA3_384:
			hashes = append(hashes, sha3.New384())
		case SHA3_512:
			hashes = append(hashes, sha3.New512())
		case SHA512:
			hashes = append(hashes, sha512.New())
		case SHA512_224:
			hashes = append(hashes, sha512.New512_224())
		case SHA512_256:
			hashes = append(hashes, sha512.New512_256())
		case XXH64:
			hashes = append(hashes, xxhash.New())
		default:
			return nil, 0, fmt.Errorf("unknown hash type '%v'", name)
		}
	}

	f, err := file.ReadOpen(name)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer f.Close()

	hashWriter := multiWriter(hashes)
	// Make sure it hashes up to the limit in case the file is growing
	// since its size was checked.
	validSizeLimit := maxSize < math.MaxInt64-1
	var r io.Reader = f
	if validSizeLimit {
		r = io.LimitReader(r, int64(maxSize+1))
	}
	written, err := io.Copy(hashWriter, r)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to calculate file hashes: %w", err)
	}

	// The file grew larger than configured limit.
	if validSizeLimit && written > int64(maxSize) {
		return nil, 0, nil
	}

	nameToHash = make(map[HashType]Digest, len(hashes))
	for i, h := range hashes {
		nameToHash[hashType[i]] = h.Sum(nil)
	}

	return nameToHash, uint64(written), nil
}

func multiWriter(hash []hash.Hash) io.Writer {
	writers := make([]io.Writer, 0, len(hash))
	for _, h := range hash {
		writers = append(writers, h)
	}
	return io.MultiWriter(writers...)
}
