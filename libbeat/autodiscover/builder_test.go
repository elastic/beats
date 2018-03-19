package autodiscover

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

type fakeBuilder struct{}

func (f *fakeBuilder) CreateConfig(event bus.Event) []*common.Config {
	return []*common.Config{common.NewConfig()}
}

func newFakeBuilder(_ *common.Config) (Builder, error) {
	return &fakeBuilder{}, nil
}

func TestBuilderRegistry(t *testing.T) {
	// Add a new builder
	reg := NewRegistry()
	reg.AddBuilder("fake", newFakeBuilder)

	// Check if that builder is available in registry
	b := reg.GetBuilder("fake")
	assert.NotNil(t, b)

	// Generate a config with type fake
	config := BuilderConfig{
		Type: "fake",
	}

	cfg, err := common.NewConfigFrom(&config)

	// Make sure that config building doesn't fail
	assert.Nil(t, err)

	builder, err := reg.BuildBuilder(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, builder)

	// Try to create a config with fake builder and assert length
	// of configs returned is one
	res := builder.CreateConfig(nil)
	assert.Equal(t, len(res), 1)

	builders := Builders{}
	builders = append(builders, builder)

	// Try using builders object for the same as above and expect
	// the same result
	res = builders.GetConfig(nil)
	assert.Equal(t, len(res), 1)
}
