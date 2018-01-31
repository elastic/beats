// +build darwin

package file_integrity

import (
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	key   = "com.apple.metadata:kMDItemWhereFroms"
	value = `62 70 6C 69 73 74 30 30 A2 01 02 5F 10 5B 68 74
			 74 70 73 3A 2F 2F 61 72 74 69 66 61 63 74 73 2E
			 65 6C 61 73 74 69 63 2E 63 6F 2F 64 6F 77 6E 6C
			 6F 61 64 73 2F 62 65 61 74 73 2F 61 75 64 69 74
			 62 65 61 74 2F 61 75 64 69 74 62 65 61 74 2D 36
			 2E 31 2E 31 2D 64 61 72 77 69 6E 2D 78 38 36 5F
			 36 34 2E 74 61 72 2E 67 7A 5F 10 30 68 74 74 70
			 73 3A 2F 2F 77 77 77 2E 65 6C 61 73 74 69 63 2E
			 63 6F 2F 64 6F 77 6E 6C 6F 61 64 73 2F 62 65 61
			 74 73 2F 61 75 64 69 74 62 65 61 74 08 0B 69 00
			 00 00 00 00 00 01 01 00 00 00 00 00 00 00 03 00
			 00 00 00 00 00 00 00 00 00 00 00 00 00 00 9C`
)

func TestDarwinWhereFroms(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Unsupported platform")
	}
	f, err := ioutil.TempFile("", "wherefrom")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	t.Run("no origin", func(t *testing.T) {
		origin, err := GetFileOrigin(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, origin, 0)
	})
	t.Run("valid origin", func(t *testing.T) {
		err = exec.Command("xattr", "-w", "-x", key, value, f.Name()).Run()
		if err != nil {
			t.Fatal(err)
		}
		origin, err := GetFileOrigin(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, origin, 2)
		assert.Equal(t, "https://artifacts.elastic.co/downloads/beats/auditbeat/auditbeat-6.1.1-darwin-x86_64.tar.gz", origin[0])
		assert.Equal(t, "https://www.elastic.co/downloads/beats/auditbeat", origin[1])
	})
	t.Run("empty origin", func(t *testing.T) {
		err = exec.Command("xattr", "-w", "-x", key, "", f.Name()).Run()
		if err != nil {
			t.Fatal(err)
		}
		origin, err := GetFileOrigin(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, origin, 0)
	})
	t.Run("bad origin", func(t *testing.T) {
		err = exec.Command("xattr", "-w", "-x", key, "01 23 45 67", f.Name()).Run()
		if err != nil {
			t.Fatal(err)
		}
		_, err := GetFileOrigin(f.Name())
		assert.Error(t, err)
	})
}
