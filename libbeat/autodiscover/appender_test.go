package autodiscover

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

type fakeAppender struct{}

func (f *fakeAppender) Append(event bus.Event) {
	event["foo"] = "bar"
}

func newFakeAppender(_ *common.Config) (Appender, error) {
	return &fakeAppender{}, nil
}

func TestAppenderrRegistry(t *testing.T) {
	// Add a new builder
	reg := NewRegistry()
	reg.AddAppender("fake", newFakeAppender)

	// Check if that appender is available in registry
	b := reg.GetAppender("fake")
	assert.NotNil(t, b)

	// Generate a config with type fake
	config := AppenderConfig{
		Type: "fake",
	}

	cfg, err := common.NewConfigFrom(&config)

	// Make sure that config building doesn't fail
	assert.Nil(t, err)

	appender, err := reg.BuildAppender(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, appender)

	// Try to append onto an event using fakeAppender and the result should have one item
	event := bus.Event{}
	appender.Append(event)
	assert.Equal(t, len(event), 1)
	assert.Equal(t, event["foo"], "bar")

	appenders := Appenders{}
	appenders = append(appenders, appender)

	// Try using appenders object for the same as above and expect
	// the same result
	event = bus.Event{}
	appenders.Append(event)
	assert.Equal(t, len(event), 1)
	assert.Equal(t, event["foo"], "bar")
}
