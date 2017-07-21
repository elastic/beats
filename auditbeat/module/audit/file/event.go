package file

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	Timestamp  time.Time // Time of event.
	Path       string    // The path associated with the event.
	TargetPath string    // Target path for symlinks.
	Action     string    // Action (like created, updated).
	Info       *Metadata // File metadata (if the file exists).
	Hashed     bool      // True if hashed.
	MD5        string
	SHA1       string
	SHA256     string

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
		"hashed":     e.Hashed,
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

	if e.Hashed {
		m["md5"] = e.MD5
		m["sha1"] = e.SHA1
		m["sha256"] = e.SHA256
	}

	return m
}

func hashFile(name string) (md5sum, sha1sum, sha256sum string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to open file for hashing")
	}
	defer f.Close()

	m5 := md5.New()
	s1 := sha1.New()
	s256 := sha256.New()

	hashWriter := io.MultiWriter(m5, s1, s256)
	if _, err := io.Copy(hashWriter, f); err != nil {
		return "", "", "", errors.Wrap(err, "failed to calculate file hashes")
	}

	return hex.EncodeToString(m5.Sum(nil)),
		hex.EncodeToString(s1.Sum(nil)),
		hex.EncodeToString(s256.Sum(nil)),
		nil
}
