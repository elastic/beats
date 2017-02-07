// +build !windows

package file_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common/file"
	"github.com/stretchr/testify/assert"
)

func TestStat(t *testing.T) {
	f, err := ioutil.TempFile("", "teststat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	link := filepath.Join(os.TempDir(), "teststat-link")
	if err := os.Symlink(f.Name(), link); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(link)

	info, err := file.Stat(link)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, info.Mode().IsRegular())

	uid, err := info.UID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Geteuid(), uid)

	gid, err := info.GID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Getegid(), gid)
}

func TestLstat(t *testing.T) {
	link := filepath.Join(os.TempDir(), "link")
	if err := os.Symlink("dummy", link); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(link)

	info, err := file.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, info.Mode()&os.ModeSymlink > 0)

	uid, err := info.UID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Geteuid(), uid)

	gid, err := info.GID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Getegid(), gid)
}
