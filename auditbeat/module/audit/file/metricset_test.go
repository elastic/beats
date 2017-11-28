package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
		"file.paths": []string{path},
	}
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

	ms := mbtest.NewPushMetricSet(t, getConfig(dir))
	events, errs := mbtest.RunPushMetricSet(time.Second, ms)
	if len(errs) > 0 {
		t.Fatalf("received errors: %+v", errs)
	}

	if !assert.Len(t, events, 2) {
		return
	}
	event := events[0]
	assert.Equal(t, dir, event["path"])
	assert.Equal(t, []string{"created"}, event["action"])
	event = events[1]
	assert.Equal(t, e.Path, event["path"])
	assert.Equal(t, []string{"deleted"}, event["action"])
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
