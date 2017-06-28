package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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
