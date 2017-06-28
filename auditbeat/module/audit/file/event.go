package file

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

type Action uint8

func (a Action) String() string {
	if name, found := actionNames[a]; found {
		return name
	}
	return "unknown"
}

const (
	Unknown = iota << 1
	AttributesModified
	Created
	Deleted
	Updated
	MovedTo
	CollisionWithin
	Unmounted
	RootChanged
)

var actionNames = map[Action]string{
	AttributesModified: "attributes_modified",
	Created:            "created",
	Deleted:            "deleted",
	Updated:            "updated",
	MovedTo:            "moved_to",
	CollisionWithin:    "collision_within",
	Unmounted:          "unmounted",
	RootChanged:        "root_changed",
}

type Event struct {
	Path      string    `structs:"path"`   // The path associated with the event.
	Action    string    `structs:"action"` // Action (like created, updated).
	Inode     uint64    `structs:"inode"`
	UID       int32     `structs:"uid"`
	GID       int32     `structs:"gid"`
	Owner     string    `structs:"owner,omitempty"`
	Group     string    `structs:"group,omitempty"`
	Mode      string    `structs:"mode"`
	Size      int64     `structs:"size"`
	ATime     time.Time `structs:"atime"` // Last access time.
	MTime     time.Time `structs:"mtime"` // Last modification time.
	CTime     time.Time `structs:"ctime"` // Last status change time.
	MD5       string    `structs:"md5,omitempty"`
	SHA1      string    `structs:"sha1,omitempty"`
	SHA256    string    `structs:"sha256,omitempty"`
	Hashed    bool      `structs:"hashed"`     // True if hashed.
	Timestamp time.Time `structs:"@timestamp"` // Time of event.
}

func buildMapStr(e *Event) common.MapStr {
	m := common.MapStr{
		"@timestamp": e.Timestamp,
		"path":       e.Path,
		"action":     e.Action,
		"inode":      e.Inode,
		"uid":        e.UID,
		"gid":        e.GID,
		"mode":       e.Mode,
		"size":       e.Size,
		"atime":      e.ATime,
		"mtime":      e.MTime,
		"ctime":      e.CTime,
		"hashed":     e.Hashed,
	}

	if e.Owner != "" {
		m["owner"] = e.Owner
	}
	if e.Group != "" {
		m["group"] = e.Group
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
