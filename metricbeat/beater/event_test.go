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
	moduleName    = "mymodule"
	metricSetName = "mymetricset"
	host          = "localhost"
	elapsed       = time.Duration(500 * time.Millisecond)
	tag           = "alpha"
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
	// processors
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
	assert.Equal(t, common.Time(startTime), event["@timestamp"])

	metricset := event["metricset"].(common.MapStr)
	assert.Equal(t, moduleName, metricset["module"])
	assert.Equal(t, metricSetName, metricset["name"])
	assert.Equal(t, int64(500000), metricset["rtt"])
	assert.Equal(t, host, metricset["host"])
	assert.Equal(t, common.MapStr{}, event[moduleName].(common.MapStr)[metricSetName])
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
