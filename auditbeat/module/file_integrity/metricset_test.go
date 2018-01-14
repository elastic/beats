package file_integrity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/paths"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer setup(t)()

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

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir))
	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	fullEvent := mbtest.StandardizeEvent(ms, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent)
}

func TestDetectDeletedFiles(t *testing.T) {
	defer setup(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := &Event{
		Timestamp: time.Now().UTC(),
		Path:      filepath.Join(dir, "ghost.file"),
		Action:    Created,
	}
	if err = store(bucket, e); err != nil {
		t.Fatal(err)
	}

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir))
	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	if !assert.Len(t, events, 2) {
		return
	}
	event := events[0].MetricSetFields
	path, err := event.GetValue("file.path")
	if assert.NoError(t, err) {
		assert.Equal(t, dir, path)
	}

	action, err := event.GetValue("event.action")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"created"}, action)
	}

	event = events[1].MetricSetFields
	path, err = event.GetValue("file.path")
	if assert.NoError(t, err) {
		assert.Equal(t, e.Path, path)
	}

	action, err = event.GetValue("event.action")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"deleted"}, action)
	}
}

func TestExcludedFiles(t *testing.T) {
	defer setup(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir))

	go func() {
		for _, f := range []string{"FILE.TXT", "FILE.TXT.SWP", "file.txt.swo", ".git/HEAD", ".gitignore"} {
			file := filepath.Join(dir, f)
			ioutil.WriteFile(file, []byte("hello world"), 0600)
		}
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	wanted := map[string]bool{
		dir: true,
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

func setup(t testing.TB) func() {
	// path.data should be set so that the DB is written to a predictable location.
	var err error
	paths.Paths.Data, err = ioutil.TempDir("", "beat-data-dir")
	if err != nil {
		t.Fatal()
	}
	return func() { os.RemoveAll(paths.Paths.Data) }
}

func getConfig(path string) map[string]interface{} {
	return map[string]interface{}{
		"module":        "file_integrity",
		"paths":         []string{path},
		"exclude_files": []string{`(?i)\.sw[nop]$`, `[/\\]\.git([/\\]|$)`},
	}
}
