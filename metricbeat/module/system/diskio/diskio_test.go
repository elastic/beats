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
}

func TestDataRegexp(t *testing.T) {
	oldFS := system.HostFS
	newFS := "_meta/testdata"
	system.HostFS = &newFS
	defer func() {
		system.HostFS = oldFS
	}()

	conf := getConfig()
	conf["diskio.name.regexp"] = "^sda"
	f := mbtest.NewEventsFetcher(t, getConfig())

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}

	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 7, len(data))
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
	}
}
