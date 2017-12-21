package file_integrity

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetadata(t *testing.T) {
	f, err := ioutil.TempFile("", "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.WriteString("metadata test")
	if err != nil {
		t.Fatal(err)
	}
	f.Sync()
	f.Close()

	info, err := os.Lstat(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	meta, err := NewMetadata(f.Name(), info)
	if err != nil {
		t.Fatal(err)
	}

	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotZero(t, meta.Inode)

	if runtime.GOOS == "windows" {
		// The owner can differ from the creator if the GPO for
		// "System object Default owner for objects created by members of the Administrators group"
		// is set to "administrators group" rather than "object creator".
		if meta.Owner == u.Username {
			assert.Equal(t, u.Uid, meta.SID)
		} else if meta.Owner == `BUILTIN\Administrators` {
			// Well-known SID for BUILTIN_ADMINISTRATORS.
			assert.Equal(t, "S-1-5-32-544", meta.SID)
		} else {
			t.Error("unexpected owner", meta.Owner)
		}
		assert.Zero(t, meta.UID)
		assert.Zero(t, meta.GID)
		assert.Empty(t, meta.Group)
	} else {
		group, err := user.LookupGroupId(u.Gid)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, u.Uid, strconv.Itoa(int(meta.UID)))
		assert.Equal(t, u.Gid, strconv.Itoa(int(meta.GID)))
		assert.Equal(t, u.Username, meta.Owner)
		assert.Equal(t, group.Name, meta.Group)
		assert.Empty(t, meta.SID)

		assert.EqualValues(t, 0600, meta.Mode)
	}

	assert.EqualValues(t, len("metadata test"), meta.Size, "size")
	assert.NotZero(t, meta.MTime, "mtime")
	assert.NotZero(t, meta.CTime, "ctime")
	assert.Equal(t, FileType, meta.Type, "type")
}

func TestSetUIDSetGIDBits(t *testing.T) {
	f, err := ioutil.TempFile("", "setuid")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.WriteString("metadata test")
	if err != nil {
		t.Fatal(err)
	}
	f.Sync()
	f.Close()

	info, err := os.Lstat(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	meta, err := NewMetadata(f.Name(), info)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, meta.SetUID)
	assert.False(t, meta.SetGID)

	if runtime.GOOS == "windows" {
		t.Skip("No setuid/setgid bits on Windows")
	}

	for _, flags := range []os.FileMode{
		0600 | os.ModeSetuid,
		0600 | os.ModeSetgid,
		0600 | os.ModeSetuid | os.ModeSetuid,
	} {
		msg := fmt.Sprintf("checking flags %04o", flags)
		if err = os.Chmod(f.Name(), flags); err != nil {
			t.Fatal(err, msg)
		}

		info, err = os.Lstat(f.Name())
		if err != nil {
			t.Fatal(err, msg)
		}

		meta, err = NewMetadata(f.Name(), info)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, flags&os.ModeSetuid != 0, meta.SetUID)
		assert.Equal(t, flags&os.ModeSetgid != 0, meta.SetGID)
	}
}
