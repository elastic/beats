package file_integrity

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/metricbeat/mb"
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
	// SourceScan identifies events triggerd by a file system scan.
	SourceScan Source = iota
	// SourceFSNotify identifies events triggered by a notification from the
	// file system.
	SourceFSNotify
)

var sourceNames = map[Source]string{
	SourceScan:     "scan",
	SourceFSNotify: "fsnotify",
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
)

var typeNames = map[Type]string{
	FileType:    "file",
	DirType:     "dir",
	SymlinkType: "symlink",
}

// Digest is a output of a hash function.
type Digest []byte

// String returns the digest value in lower-case hexadecimal form.
func (d Digest) String() string {
	return hex.EncodeToString(d)
}

// MarshalText encodes the digest to a hexadecimal representation of itself.
func (d Digest) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// Event describe the filesystem change and includes metadata about the file.
type Event struct {
	Timestamp  time.Time           `json:"timestamp"`             // Time of event.
	Path       string              `json:"path"`                  // The path associated with the event.
	TargetPath string              `json:"target_path,omitempty"` // Target path for symlinks.
	Info       *Metadata           `json:"info"`                  // File metadata (if the file exists).
	Source     Source              `json:"source"`                // Source of the event.
	Action     Action              `json:"action"`                // Action (like created, updated).
	Hashes     map[HashType]Digest `json:"hash,omitempty"`        // File hashes.

	// Metadata
	rtt    time.Duration // Time taken to collect the info.
	errors []error       // Errors that occurred while collecting the info.
}

// Metadata contains file metadata.
type Metadata struct {
	Inode  uint64      `json:"inode"`
	UID    uint32      `json:"uid"`
	GID    uint32      `json:"gid"`
	SID    string      `json:"sid"`
	Owner  string      `json:"owner"`
	Group  string      `json:"group"`
	Size   uint64      `json:"size"`
	MTime  time.Time   `json:"mtime"`  // Last modification time.
	CTime  time.Time   `json:"ctime"`  // Last metadata change time.
	Type   Type        `json:"type"`   // File type (dir, file, symlink).
	Mode   os.FileMode `json:"mode"`   // Permissions
	SetUID bool        `json:"setuid"` // setuid bit (POSIX only)
	SetGID bool        `json:"setgid"` // setgid bit (POSIX only)
	Origin []string    `json:"origin"` // External origin info for the file (MacOS only)
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
		if event.Info.Size <= maxFileSize {
			hashes, err := hashFile(event.Path, hashTypes...)
			if err != nil {
				event.errors = append(event.errors, err)
			} else {
				event.Hashes = hashes
			}
		}
	case SymlinkType:
		event.TargetPath, _ = filepath.EvalSymlinks(event.Path)
	}

	return event
}

// NewEvent creates a new Event. Any errors that occur are included in the
// returned Event.
func NewEvent(
	path string,
	action Action,
	source Source,
	maxFileSize uint64,
	hashTypes []HashType,
) Event {
	info, err := os.Lstat(path)
	if err != nil && os.IsNotExist(err) {
		// deleted file is signaled by info == nil
		err = nil
	}
	err = errors.Wrap(err, "failed to lstat")
	return NewEventFromFileInfo(path, info, err, action, source, maxFileSize, hashTypes)
}

func buildMetricbeatEvent(e *Event, existedBefore bool) mb.Event {
	file := common.MapStr{
		"path": e.Path,
	}
	out := mb.Event{
		Timestamp: e.Timestamp,
		Took:      e.rtt,
		MetricSetFields: common.MapStr{
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
			file["size"] = info.Size
		}

		if info.Type != UnknownType {
			file["type"] = info.Type.String()
		}

		if runtime.GOOS == "windows" {
			if info.SID != "" {
				file["uid"] = info.SID
			}
		} else {
			file["uid"] = info.UID
			file["gid"] = info.GID
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
	}

	if len(e.Hashes) > 0 {
		hashes := make(common.MapStr, len(e.Hashes))
		for hashType, digest := range e.Hashes {
			hashes[string(hashType)] = digest
		}
		out.MetricSetFields.Put("hash", hashes)
	}

	if e.Action > 0 {
		actions := e.Action.InOrder(existedBefore, e.Info != nil).StringArray()
		out.MetricSetFields.Put("event.action", actions)
	}

	return out
}

// diffEvents returns true if the file info differs between the old event and
// the new event. Changes to the timestamp and action are ignored. If old
// contains a superset of new's hashes then false is returned.
func diffEvents(old, new *Event) (Action, bool) {
	if old == new {
		return 0, false
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
			o.Mode != n.Mode || o.Type != n.Type || o.SetUID != n.SetUID || o.SetGID != n.SetGID {
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

func hashFile(name string, hashType ...HashType) (map[HashType]Digest, error) {
	if len(hashType) == 0 {
		return nil, nil
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
		default:
			return nil, errors.Errorf("unknown hash type '%v'", name)
		}
	}

	f, err := file.ReadOpen(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file for hashing")
	}
	defer f.Close()

	hashWriter := multiWriter(hashes)
	if _, err := io.Copy(hashWriter, f); err != nil {
		return nil, errors.Wrap(err, "failed to calculate file hashes")
	}

	nameToHash := make(map[HashType]Digest, len(hashes))
	for i, h := range hashes {
		nameToHash[hashType[i]] = h.Sum(nil)
	}

	return nameToHash, nil
}

func multiWriter(hash []hash.Hash) io.Writer {
	writers := make([]io.Writer, 0, len(hash))
	for _, h := range hash {
		writers = append(writers, h)
	}
	return io.MultiWriter(writers...)
}
