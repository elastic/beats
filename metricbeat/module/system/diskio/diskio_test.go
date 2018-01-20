// +build !integration
// +build darwin,cgo freebsd linux windows

package diskio

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/system"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}

	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 10, len(data))
}

func TestDataNameFilter(t *testing.T) {
	oldFS := system.HostFS
	newFS := "_meta/testdata"
	system.HostFS = &newFS
	defer func() {
		system.HostFS = oldFS
	}()

	conf := getConfig()
	conf["diskio.include_devices"] = []string{"sda", "sda1", "sda2"}
	f := mbtest.NewEventsFetcher(t, getConfig())

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}

	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(data))
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
	}
}
