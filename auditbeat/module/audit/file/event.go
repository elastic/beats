package file

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// Action is a description of the change that occurred.
type Action uint8

func (a Action) String() string {
	if name, found := actionNames[a]; found {
		return name
	}
	return "unknown"
}

// List of possible Actions.
const (
	Unknown Action = iota << 1
	AttributesModified
	Created
	Deleted
	Updated
	Moved
)

var actionNames = map[Action]string{
	AttributesModified: "attributes_modified",
	Created:            "created",
	Deleted:            "deleted",
	Updated:            "updated",
	Moved:              "moved",
}

// Event describe the filesystem change and includes metadata about the file.
type Event struct {
	Timestamp  time.Time         // Time of event.
	Path       string            // The path associated with the event.
	TargetPath string            // Target path for symlinks.
	Action     string            // Action (like created, updated).
	Info       *Metadata         // File metadata (if the file exists).
	Hashes     map[string]string // File hashes.

	errors []error
}

// Metadata contains file metadata.
type Metadata struct {
	Inode uint64
	UID   uint32
	GID   uint32
	SID   string
	Owner string
	Group string
	Mode  os.FileMode
	Size  int64
	ATime time.Time // Last access time.
	MTime time.Time // Last modification time.
	CTime time.Time // Last status change time.
	Type  string    // File type (dir, file, symlink).
}

func buildMapStr(e *Event) common.MapStr {
	m := common.MapStr{
		"@timestamp": e.Timestamp,
		"path":       e.Path,
		"action":     e.Action,
		"hashed":     len(e.Hashes) > 0,
	}

	if e.TargetPath != "" {
		m["target_path"] = e.TargetPath
	}

	if e.Info != nil {
		info := e.Info
		m["inode"] = strconv.FormatUint(info.Inode, 10)
		m["size"] = info.Size
		m["atime"] = info.ATime
		m["mtime"] = info.MTime
		m["ctime"] = info.CTime

		if info.Type != "" {
			m["type"] = info.Type
		}

		if runtime.GOOS == "windows" {
			if info.SID != "" {
				m["sid"] = info.SID
			}
		} else {
			m["uid"] = info.UID
			m["gid"] = info.GID
			m["mode"] = fmt.Sprintf("%#04o", uint32(info.Mode))
		}

		if info.Owner != "" {
			m["owner"] = info.Owner
		}
		if info.Group != "" {
			m["group"] = info.Group
		}
	}

	for name, hash := range e.Hashes {
		m[name] = hash
	}

	return m
}

func hashFile(name string, hashType ...string) (map[string]string, error) {
	if len(hashType) == 0 {
		return nil, nil
	}

	var hashes []hash.Hash
	for _, name := range hashType {
		switch name {
		case "md5":
			hashes = append(hashes, md5.New())
		case "sha1":
			hashes = append(hashes, sha1.New())
		case "sha224":
			hashes = append(hashes, sha256.New224())
		case "sha256":
			hashes = append(hashes, sha256.New())
		case "sha384":
			hashes = append(hashes, sha512.New384())
		case "sha512":
			hashes = append(hashes, sha512.New())
		case "sha512_224":
			hashes = append(hashes, sha512.New512_224())
		case "sha512_256":
			hashes = append(hashes, sha512.New512_256())
		default:
			return nil, errors.Errorf("unknown hash type '%v'", name)
		}
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file for hashing")
	}
	defer f.Close()

	hashWriter := multiWriter(hashes)
	if _, err := io.Copy(hashWriter, f); err != nil {
		return nil, errors.Wrap(err, "failed to calculate file hashes")
	}

	nameToHash := make(map[string]string, len(hashes))
	for i, h := range hashes {
		nameToHash[hashType[i]] = hex.EncodeToString(h.Sum(nil))
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
