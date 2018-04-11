package add_host_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"runtime"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-sysinfo/types"
)

func TestRun(t *testing.T) {
	event := &beat.Event{
		Fields:    common.MapStr{},
		Timestamp: time.Now(),
	}
	p, err := newHostMetadataProcessor(nil)
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		assert.IsType(t, types.ErrNotImplemented, err)
		return
	}
	assert.NoError(t, err)

	newEvent, err := p.Run(event)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("host.os.family")
	assert.NoError(t, err)
	assert.NotNil(t, v)
}
