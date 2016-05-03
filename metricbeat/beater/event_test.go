// +build !integration

package beater

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const (
	moduleName     = "mymodule"
	metricSetName  = "mymetricset"
	eventFieldName = moduleName + "-" + metricSetName
	host           = "localhost"
	elapsed        = time.Duration(500 * time.Millisecond)
	tag            = "alpha"
)

var (
	startTime = time.Now()
	errFetch  = errors.New("error fetching data")
	tags      = []string{tag}
)

var builder = eventBuilder{
	moduleName:    moduleName,
	metricSetName: metricSetName,
	// host
	startTime:     startTime,
	fetchDuration: elapsed,
	// event
	// fetchErr
	// filters
	// metadata
}

func TestEventBuilder(t *testing.T) {
	b := builder
	b.host = host
	event, err := b.build()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, defaultType, event["type"])
	assert.Equal(t, moduleName, event["module"])
	assert.Equal(t, metricSetName, event["metricset"])
	assert.Equal(t, common.Time(startTime), event["@timestamp"])
	assert.Equal(t, int64(500000), event["rtt"])
	assert.Equal(t, host, event["metricset-host"])
	assert.Equal(t, common.MapStr{}, event[eventFieldName])
	assert.Nil(t, event["error"])
}

func TestEventBuilderError(t *testing.T) {
	b := builder
	b.fetchErr = errFetch
	event, err := b.build()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, errFetch.Error(), event["error"])
}

func TestEventBuilderNoHost(t *testing.T) {
	b := builder
	event, err := b.build()
	if err != nil {
		t.Fatal(err)
	}

	_, found := event["metricset-host"]
	assert.False(t, found)
}
