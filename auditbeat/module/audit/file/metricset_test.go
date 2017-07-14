package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	go func() {
		time.Sleep(100 * time.Millisecond)
		file := filepath.Join(dir, "file.data")
		ioutil.WriteFile(file, []byte("hello world"), 0600)
	}()

	ms := mbtest.NewPushMetricSet(t, getConfig(dir))
	events, errs := mbtest.RunPushMetricSet(time.Second, ms)
	if len(errs) > 0 {
		t.Fatalf("received errors: %+v", errs)
	}
	if len(events) == 0 {
		t.Fatal("received no events")
	}

	fullEvent := mbtest.CreateFullEvent(ms, events[len(events)-1])
	mbtest.WriteEventToDataJSON(t, fullEvent)
}

func getConfig(path string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "audit",
		"metricsets": []string{"file"},
		"file.paths": map[string][]string{
			"binaries": {path},
		},
	}
}

func TestEventReader(t *testing.T) {
	// Make dir to monitor.
	dir, err := ioutil.TempDir("", "audit")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a new EventReader.
	config := defaultConfig
	config.Paths = map[string][]string{
		"testdir": {dir},
	}
	r, err := NewEventReader(config)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)
	events, err := r.Start(done)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new file.
	txt1 := filepath.Join(dir, "test1.txt")
	var fileMode os.FileMode = 0640
	t.Run("created", func(t *testing.T) {
		if err = ioutil.WriteFile(txt1, []byte("hello"), fileMode); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assert.Equal(t, Created.String(), event.Action)
		assertSameFile(t, txt1, event.Path)
		if runtime.GOOS != "windows" {
			assert.EqualValues(t, fileMode, event.Info.Mode)
		}
	})

	// Rename the file.
	txt2 := filepath.Join(dir, "test2.txt")
	t.Run("move", func(t *testing.T) {
		if err = os.Rename(txt1, txt2); err != nil {
			t.Fatal(err)
		}

		if runtime.GOOS == "windows" {
			updated := readTimeout(t, events)
			assert.Equal(t, Updated.String(), updated.Action)
		}

		// The order isn't guaranteed.
		created := readTimeout(t, events)
		moved := readTimeout(t, events)
		if created.Action != Created.String() {
			tmp := moved
			moved = created
			created = tmp
		}

		assert.Equal(t, Moved.String(), moved.Action)
		assert.Equal(t, txt1, moved.Path)

		assert.Equal(t, Created.String(), created.Action)
		assertSameFile(t, txt2, created.Path)
	})

	// Chmod the file.
	t.Run("attributes modified", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip()
		}

		if err = os.Chmod(txt2, 0644); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.Equal(t, AttributesModified.String(), event.Action)
		assert.EqualValues(t, 0644, event.Info.Mode)
	})

	// Append data to the file.
	t.Run("updated", func(t *testing.T) {
		f, err := os.OpenFile(txt2, os.O_RDWR|os.O_APPEND, fileMode)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(" world!")
		f.Sync()
		f.Close()

		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.Equal(t, Updated.String(), event.Action)
		if runtime.GOOS != "windows" {
			assert.EqualValues(t, 0644, event.Info.Mode)
		}
	})

	// Change the GID of the file.
	t.Run("chown", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skip chown on windows")
		}

		gid := changeGID(t, txt2)
		event := readTimeout(t, events)
		assertSameFile(t, txt2, event.Path)
		assert.Equal(t, AttributesModified.String(), event.Action)
		assert.EqualValues(t, gid, event.Info.GID)
	})

	t.Run("deleted", func(t *testing.T) {
		if err = os.Remove(txt2); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assert.Equal(t, Deleted.String(), event.Action)
	})

	// Create a sub-directory.
	subDir := filepath.Join(dir, "subdir")
	t.Run("dir created", func(t *testing.T) {
		if err = os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assertSameFile(t, subDir, event.Path)
	})

	// Test that it does not monitor recursively.
	subFile := filepath.Join(subDir, "foo.txt")
	t.Run("non-recursive", func(t *testing.T) {
		if err = ioutil.WriteFile(subFile, []byte("foo"), fileMode); err != nil {
			t.Fatal(err)
		}

		assertNoEvent(t, events)
	})

	// Test moving a file into the monitored dir from outside.
	var moveInOrig string
	moveIn := filepath.Join(dir, "test3.txt")
	t.Run("move in", func(t *testing.T) {
		f, err := ioutil.TempFile("", "test3.txt")
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("move-in")
		f.Sync()
		f.Close()
		moveInOrig = f.Name()

		if err = os.Rename(moveInOrig, moveIn); err != nil {
			t.Fatal(err)
		}

		event := readTimeout(t, events)
		assert.Equal(t, Created.String(), event.Action)
		assertSameFile(t, moveIn, event.Path)
	})

	// Test moving a file out of the monitored dir.
	t.Run("move out", func(t *testing.T) {
		if err = os.Rename(moveIn, moveInOrig); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(moveInOrig)

		event := readTimeout(t, events)
		assertSameFile(t, moveIn, event.Path)
		if runtime.GOOS == "windows" {
			assert.Equal(t, Deleted.String(), event.Action)
		} else {
			assert.Equal(t, Moved.String(), event.Action)
		}
	})
}

// readTimeout reads one event from the channel and returns it. If it does
// not receive an event after one second it will time-out and fail the test.
func readTimeout(t testing.TB, events <-chan Event) Event {
	select {
	case e, ok := <-events:
		if !ok {
			t.Fatal("failed reading from event channel")
		}
		t.Logf("%+v", buildMapStr(&e).StringToPrint())
		return e
	case <-time.After(time.Second):
		t.Fatalf("%+v", errors.Errorf("timed-out waiting for event"))
	}

	return Event{}
}

// assertNoEvent asserts that no event is received on the channel. It waits for
// 250ms.
func assertNoEvent(t testing.TB, events <-chan Event) {
	select {
	case e := <-events:
		t.Fatal("received unexpected event", e)
	case <-time.After(250 * time.Millisecond):
	}
}

// assertSameFile asserts that two files are the same.
func assertSameFile(t testing.TB, f1, f2 string) {
	if f1 == f2 {
		return
	}

	info1, err := os.Lstat(f1)
	if err != nil {
		t.Error(err)
		return
	}

	info2, err := os.Lstat(f2)
	if err != nil {
		t.Error(err)
		return
	}

	assert.True(t, os.SameFile(info1, info2), "%v and %v are not the same file", f1, f2)
}

// changeGID changes the GID of a file using chown. It uses the second group
// that the user is a member of. If the user is only a member of one group then
// it will skip the test.
func changeGID(t testing.TB, file string) int {
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal("failed to get groups", err)
	}

	if len(groups) <= 1 {
		t.Skip("no group that we can change to")
	}

	// The second one will be a non-default group.
	gid := groups[1]
	if err = os.Chown(file, -1, gid); err != nil {
		t.Fatal(err)
	}

	return gid
}
