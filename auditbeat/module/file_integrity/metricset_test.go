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
	"crypto/sha1"
	"github.com/elastic/beats/v7/auditbeat/ab"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/auditbeat/datastore"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	dir := t.TempDir()

	go func() {
		time.Sleep(100 * time.Millisecond)
		file := filepath.Join(dir, "file.data")
		require.NoError(t, os.WriteFile(file, []byte("hello world"), 0o600))
	}()

	ms := mbtest.NewPushMetricSetV2WithRegistry(t, getConfig(dir), ab.Registry)
	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	fullEvent := mbtest.StandardizeEvent(ms, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestActions(t *testing.T) {
	skipOnCIForDarwinAMD64(t)

	// Can be removed after https://github.com/elastic/ingest-dev/issues/3016 is solved
	skipOnBuildkiteWindows(t)
	// Can be removed after https://github.com/elastic/ingest-dev/issues/3076 is solved
	skipOnBuildkiteDarwinArm(t)

	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	// First directory
	dir := t.TempDir()

	// Second directory (to be reported with "initial_scan")
	newDir := t.TempDir()

	createdFilepath := filepath.Join(dir, "created.txt")
	updatedFilepath := filepath.Join(dir, "updated.txt")
	deletedFilepath := filepath.Join(dir, "deleted.txt")

	// Add first directory to db (so that files in it are not reported with "initial_scan")
	e := &Event{
		Timestamp: time.Now().UTC(),
		Path:      dir,
		Action:    InitialScan,
	}
	if err = store(bucket, e); err != nil {
		t.Fatal(err)
	}

	// Add fake event for non-existing file to db to simulate when a file has been deleted
	deletedFileEvent := &Event{
		Timestamp: time.Now().UTC(),
		Path:      deletedFilepath,
		Action:    Created,
	}
	if err = store(bucket, deletedFileEvent); err != nil {
		t.Fatal(err)
	}

	// Insert fake file event into db to simulate when a file has changed
	digest := sha1.New().Sum([]byte("different string"))
	updatedFileEvent := &Event{
		Timestamp: time.Now().UTC(),
		Path:      updatedFilepath,
		Action:    Created,
		Hashes:    map[HashType]Digest{SHA1: digest},
	}
	if err = store(bucket, updatedFileEvent); err != nil {
		t.Fatal(err)
	}

	// Create some files in first directory
	require.NoError(t, os.WriteFile(createdFilepath, []byte("hello world"), 0o600))
	require.NoError(t, os.WriteFile(updatedFilepath, []byte("hello world"), 0o600))

	ms := mbtest.NewPushMetricSetV2WithRegistry(t, getConfig(dir, newDir), ab.Registry)
	events := mbtest.RunPushMetricSetV2(10*time.Second, 5, ms)
	assert.Len(t, events, 5)

	for _, event := range events {
		if event.Error != nil {
			t.Fatalf("received error: %+v", event.Error)
		}

		actions, err := event.MetricSetFields.GetValue("event.action")
		path, err2 := event.MetricSetFields.GetValue("file.path")
		if assert.NoError(t, err) && assert.NoError(t, err2) {
			// Note: Actions reported for a file or directory will be different
			// depending on whether the scanner or the platform-dependent
			// filesystem event listener reported it. The subset of actions we test
			// for here should be consistent across all cases though.
			switch path.(string) {
			case newDir:
				assert.Contains(t, actions, "initial_scan")
			case dir:
				assert.Contains(t, actions, "attributes_modified")
			case deletedFilepath:
				assert.Contains(t, actions, "deleted")
			case createdFilepath:
				assert.Contains(t, actions, "created")
			case updatedFilepath:
				assert.Contains(t, actions, "updated")
				assert.Contains(t, actions, "attributes_modified")
			default:
				assert.Fail(t, "unexpected path", "path %v", path)
			}
		}
	}
}

func TestExcludedFiles(t *testing.T) {
	skipOnCIForDarwinAMD64(t)

	// Can be removed after https://github.com/elastic/ingest-dev/issues/3016 is solved
	skipOnBuildkiteWindows(t)
	// Can be removed after https://github.com/elastic/ingest-dev/issues/3076 is solved
	skipOnBuildkiteDarwinArm(t)

	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir := t.TempDir()

	ms := mbtest.NewPushMetricSetV2WithRegistry(t, getConfig(dir), ab.Registry)

	go func() {
		for _, f := range []string{"FILE.TXT", "FILE.TXT.SWP", "file.txt.swo", ".git/HEAD", ".gitignore"} {
			file := filepath.Join(dir, f)
			_ = os.WriteFile(file, []byte("hello world"), 0o600)
		}
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	wanted := map[string]bool{
		dir:                              true,
		filepath.Join(dir, "FILE.TXT"):   true,
		filepath.Join(dir, ".gitignore"): true,
	}
	if !assert.Len(t, events, len(wanted)) {
		return
	}
	for _, e := range events {
		event := e.MetricSetFields
		path, err := event.GetValue("file.path")
		if assert.NoError(t, err) {
			_, ok := wanted[path.(string)]
			assert.True(t, ok)
		}
	}
}

func TestIncludedExcludedFiles(t *testing.T) {
	skipOnCIForDarwinAMD64(t)

	// Can be removed after https://github.com/elastic/ingest-dev/issues/3016 is solved
	skipOnBuildkiteWindows(t)
	// Can be removed after https://github.com/elastic/ingest-dev/issues/3076 is solved
	skipOnBuildkiteDarwinArm(t)

	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir := t.TempDir()

	err = os.Mkdir(filepath.Join(dir, ".ssh"), 0o700)
	if err != nil {
		t.Fatal(err)
	}

	config := getConfig(dir)
	config["include_files"] = []string{`\.ssh`}
	config["recursive"] = true
	ms := mbtest.NewPushMetricSetV2WithRegistry(t, config, ab.Registry)

	for _, f := range []string{"FILE.TXT", ".ssh/known_hosts", ".ssh/known_hosts.swp"} {
		file := filepath.Join(dir, f)
		require.NoError(t, os.WriteFile(file, []byte("hello world"), 0o600))
	}

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	wanted := map[string]bool{
		dir:                                    true,
		filepath.Join(dir, ".ssh"):             true,
		filepath.Join(dir, ".ssh/known_hosts"): true,
	}
	if !assert.Len(t, events, len(wanted)) {
		return
	}

	got := map[string]bool{}
	for _, e := range events {
		event := e.MetricSetFields
		path, err := event.GetValue("file.path")
		if assert.NoError(t, err, "Failed to read file.path field") {
			got[path.(string)] = true
		}
	}
	assert.Equal(t, wanted, got)
}

func TestErrorReporting(t *testing.T) {
	if runtime.GOOS == "darwin" {
		// FSEvents doesn't generate write events during this test,
		// either it needs the file to be closed before generating
		// events or the non-readable permissions trick it somehow.
		t.Skip("Skip this test on Darwin")
	}
	if runtime.GOOS != "windows" && os.Getuid() == 0 {
		// There's no easy way to make a file unreadable by root
		// in UNIX/Linux OS.
		t.Skip("This test can't be run as root")
	}
	defer abtest.SetupDataDir(t)()

	dir := t.TempDir()

	path := filepath.Join(dir, "unreadable.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	makeFileNonReadable(t, path)

	config := getConfig(dir)
	config["scan_at_start"] = false
	ms := mbtest.NewPushMetricSetV2WithRegistry(t, config, ab.Registry)

	done := make(chan struct{}, 1)
	ready := make(chan struct{}, 1)
	go func() {
		for {
			_, err := f.WriteString("can't read this\n")
			require.NoError(t, err)
			require.NoError(t, f.Sync())
			select {
			case <-done:
				close(ready)
				return
			default:
				time.Sleep(time.Second / 10)
			}
		}
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 10, ms)
	close(done)
	<-ready

	getField := func(ev *mb.Event, field string) interface{} {
		v, _ := ev.MetricSetFields.GetValue(field)
		return v
	}
	match := func(ev *mb.Event) bool {
		return ev != nil &&
			reflect.DeepEqual(getField(ev, "event.type"), []string{"change"}) &&
			getField(ev, "file.type") == "file" &&
			getField(ev, "file.extension") == "txt"
	}

	var event *mb.Event
	for idx, ev := range events {
		ev := ev
		t.Log("event[", idx, "] = ", ev)
		if match(&ev) {
			event = &ev
			break
		}
	}

	if !assert.NotNil(t, event) {
		t.Fatal("target event not found")
	}

	if event.Error != nil {
		t.Fatalf("received error: %+v", event.Error)
	}

	errors := getField(event, "error.message")
	if !assert.NotNil(t, errors) {
		t.Fatal("no error.message in event")
	}

	var errList []string
	switch v := errors.(type) {
	case string:
		errList = []string{v}
	case []interface{}:
		for _, val := range v {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("Unexpected type %T in error.message list: %v", val, val)
			}
			errList = append(errList, str)
		}

	case []string:
		errList = v

	default:
		t.Fatalf("Unexpected type %T in error.message: %v", v, v)
	}

	found := false
	assert.NotEmpty(t, errList)
	for _, msg := range errList {
		if strings.Contains(msg, "hashing") {
			found = true
			break
		}
	}
	assert.True(t, found)
}

type testReporter struct {
	events []mb.Event
	errors []error
}

func (t *testReporter) Event(event mb.Event) bool {
	t.events = append(t.events, event)
	return true
}

func (t *testReporter) Error(err error) bool {
	t.errors = append(t.errors, err)
	return true
}

func (t *testReporter) Done() <-chan struct{} {
	return nil
}

func (t *testReporter) Clear() {
	t.events = nil
	t.errors = nil
}

func checkExpectedEvent(t *testing.T, ms *MetricSet, title string, input *Event, expected map[string]interface{}) {
	t.Helper()

	var reporter testReporter
	if !ms.reportEvent(&reporter, input) {
		t.Fatal("reportEvent failed", title)
	}
	if !assert.Empty(t, reporter.errors, title) {
		t.Fatal("errors during reportEvent", reporter.errors, title)
	}
	if expected == nil {
		assert.Empty(t, reporter.events)
		return
	}
	if !assert.NotEmpty(t, reporter.events) {
		t.Fatal("no event received", title)
	}
	if !assert.Len(t, reporter.events, 1) {
		t.Fatal("more than one event received", title)
	}
	ev := reporter.events[0]
	t.Log("got title=", title, "event=", ev)
	for k, v := range expected {
		iface, err := ev.MetricSetFields.GetValue(k)
		if v == nil {
			assert.Error(t, err, title)
			continue
		}
		if err != nil {
			t.Fatal("failed to fetch key", k, title)
		}
		assert.Equal(t, v, iface, title)
	}
}

type expectedEvent struct {
	title    string
	input    Event
	expected map[string]interface{}
}

func (e expectedEvent) validate(t *testing.T, ms *MetricSet) {
	checkExpectedEvent(t, ms, e.title, &e.input, e.expected)
}

type expectedEvents []expectedEvent

func (e expectedEvents) validate(t *testing.T) {
	store, err := os.CreateTemp("", "bucket")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	defer os.Remove(store.Name())

	ds := datastore.New(store.Name(), 0o644)
	bucket, err := ds.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()
	config := getConfig("somepath")
	config["hash_types"] = []string{"sha1"}
	ms, ok := mbtest.NewPushMetricSetV2WithRegistry(t, config, ab.Registry).(*MetricSet)
	if !assert.True(t, ok) {
		t.Fatal("can't create metricset")
	}
	ms.bucket = bucket.(datastore.BoltBucket)
	for _, ev := range e {
		ev.validate(t, ms)
	}
}

func TestEventFailedHash(t *testing.T) {
	baseTime := time.Now()
	t.Run("failed hash on update", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "creation event",
				input: Event{
					Timestamp: baseTime,
					Path:      "/some/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: []byte("11111111111111111111"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": Digest("11111111111111111111"),
				},
			},
			expectedEvent{
				title: "update with hash",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Updated,
					Hashes: map[HashType]Digest{
						SHA1: []byte("22222222222222222222"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"updated"},
					"event.type":     []string{"change"},
					"file.hash.sha1": Digest("22222222222222222222"),
				},
			},
			expectedEvent{
				title: "update with failed hash",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Source:     SourceFSNotify,
					Action:     Updated,
					hashFailed: true,
				},
				expected: map[string]interface{}{
					"event.action":   []string{"updated"},
					"event.type":     []string{"change"},
					"file.hash.sha1": nil,
				},
			},
			expectedEvent{
				title: "update again now with hash",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/path",
					Info: &Metadata{
						CTime: baseTime,
						MTime: baseTime,
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Updated,
					Hashes: map[HashType]Digest{
						SHA1: []byte("33333333333333333333"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"updated"},
					"event.type":     []string{"change"},
					"file.hash.sha1": Digest("33333333333333333333"),
				},
			},
			expectedEvent{
				title: "new modification time",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/path",
					Info: &Metadata{
						CTime: baseTime,
						MTime: baseTime.Add(time.Second),
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Updated,
					Hashes: map[HashType]Digest{
						SHA1: []byte("33333333333333333333"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"attributes_modified"},
					"event.type":     []string{"change"},
					"file.hash.sha1": Digest("33333333333333333333"),
				},
			},
		}.validate(t)
	})
	t.Run("failed hash on creation", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "creation event with failed hash",
				input: Event{
					Timestamp: baseTime,
					Path:      "/some/other/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action:     Created,
					Source:     SourceFSNotify,
					hashFailed: true,
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": nil,
				},
			},
			expectedEvent{
				title: "update with hash",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/other/path",
					Info: &Metadata{
						MTime: baseTime.Add(time.Second),
						CTime: baseTime,
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Updated,
					Hashes: map[HashType]Digest{
						SHA1: []byte("22222222222222222222"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"updated", "attributes_modified"},
					"event.type":     []string{"change"},
					"file.hash.sha1": Digest("22222222222222222222"),
				},
			},
		}.validate(t)
	})
	t.Run("delete", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "creation event",
				input: Event{
					Timestamp: baseTime,
					Path:      "/some/other/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: []byte("22222222222222222222"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": Digest("22222222222222222222"),
				},
			},
			expectedEvent{
				title: "delete",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/other/path",
					Info:      nil,
					Source:    SourceFSNotify,
					Action:    Deleted,
					Hashes:    nil,
				},
				expected: map[string]interface{}{
					"event.action":   []string{"deleted"},
					"event.type":     []string{"deletion"},
					"file.hash.sha1": nil,
				},
			},
		}.validate(t)
	})
	t.Run("move", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "creation event",
				input: Event{
					Timestamp: baseTime,
					Path:      "/some/other/path",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: []byte("22222222222222222222"),
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": Digest("22222222222222222222"),
				},
			},
			expectedEvent{
				title: "delete",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/some/other/path",
					Info:      nil,
					Source:    SourceFSNotify,
					// FSEvents likes to add extra flags to delete events.
					Action: Moved,
					Hashes: nil,
				},
				expected: map[string]interface{}{
					"event.action":   []string{"moved"},
					"event.type":     []string{"change"},
					"file.hash.sha1": nil,
				},
			},
		}.validate(t)
	})
}

func TestEventDelete(t *testing.T) {
	store, err := os.CreateTemp("", "bucket")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	defer os.Remove(store.Name())

	ds := datastore.New(store.Name(), 0o644)
	bucket, err := ds.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()
	config := getConfig("somepath")
	config["hash_types"] = []string{"sha1"}
	ms, ok := mbtest.NewPushMetricSetV2WithRegistry(t, config, ab.Registry).(*MetricSet)
	if !assert.True(t, ok) {
		t.Fatal("can't create metricset")
	}
	ms.bucket = bucket.(datastore.BoltBucket)

	baseTime := time.Now()
	sha := Digest("22222222222222222222")
	t.Run("delete event for file missing on disk", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "creation event",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": sha,
				},
			},
			expectedEvent{
				title: "delete",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/file",
					Source:    SourceFSNotify,
					Action:    Deleted,
				},
				expected: map[string]interface{}{
					"event.action": []string{"deleted"},
					"event.type":   []string{"deletion"},
				},
			},
			expectedEvent{
				title: "creation event",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": sha,
				},
			},
		}.validate(t)
	})

	// This tests getting a DELETE followed by a CREATE, but by the time we observe the former the file already
	// exists on disk.
	shaNext := Digest("22222222222222222223")
	t.Run("delete event for file present on disk (different contents)", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "create",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": sha,
				},
			},
			expectedEvent{
				title: "delete",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Deleted,
					Hashes: map[HashType]Digest{
						SHA1: shaNext,
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"updated"},
					"event.type":     []string{"change"},
					"file.hash.sha1": shaNext,
				},
			},
			expectedEvent{
				title: "re-create",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: shaNext,
					},
				},
				expected: nil, // Already observed during handling of previous event.
			},
		}.validate(t)
	})

	t.Run("delete event for file present on disk (same contents)", func(t *testing.T) {
		expectedEvents{
			expectedEvent{
				title: "create",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				expected: map[string]interface{}{
					"event.action":   []string{"created"},
					"event.type":     []string{"creation"},
					"file.hash.sha1": sha,
				},
			},
			expectedEvent{
				title: "delete",
				input: Event{
					Timestamp: time.Now(),
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Source: SourceFSNotify,
					Action: Deleted,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				// No event because it has the same contents as before.
				expected: nil,
			},
			expectedEvent{
				title: "re-create",
				input: Event{
					Timestamp: baseTime,
					Path:      "/file",
					Info: &Metadata{
						MTime: baseTime,
						CTime: baseTime,
						Type:  FileType,
					},
					Action: Created,
					Source: SourceFSNotify,
					Hashes: map[HashType]Digest{
						SHA1: sha,
					},
				},
				// No event because it has the same contents as before.
				expected: nil,
			},
		}.validate(t)
	})
}

func getConfig(path ...string) map[string]interface{} {
	return map[string]interface{}{
		"module":        "file_integrity",
		"paths":         path,
		"exclude_files": []string{`(?i)\.sw[nop]$`, `[/\\]\.git([/\\]|$)`},
	}
}

func skipOnCIForDarwinAMD64(t testing.TB) {
	if os.Getenv("CI") == "true" && runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		t.Skip("Skip test on CI for darwin/amd64")
	}
}

func skipOnBuildkiteWindows(t testing.TB) {
	if os.Getenv("BUILDKITE") == "true" && runtime.GOOS == "windows" {
		t.Skip("Skip on Buildkite Windows: Shortened TMP problem")
	}
}

func skipOnBuildkiteDarwinArm(t testing.TB) {
	if os.Getenv("BUILDKITE") == "true" && runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		t.Skip("Skip test on Buldkite: unexpected path error")
	}
}
