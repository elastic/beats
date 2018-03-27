package autodiscover

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

// Builder provides an interface by which configs can be built from provider metadata
type Builder interface {
	// CreateConfig creates a config from hints passed from providers
	CreateConfig(event bus.Event) []*common.Config
}

// Builders is a list of Builder objects
type Builders []Builder

// BuilderConstructor is a func used to generate a Builder object
type BuilderConstructor func(*common.Config) (Builder, error)

// AddBuilder registers a new BuilderConstructor
func (r *registry) AddBuilder(name string, builder BuilderConstructor) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("builder name is required")
	}

	_, exists := r.builders[name]
	if exists {
		return fmt.Errorf("builder '%s' is already registered", name)
	}

	if builder == nil {
		return fmt.Errorf("builder '%s' cannot be registered with a nil factory", name)
	}

	r.builders[name] = builder
	logp.Debug(debugK, "Builder registered: %s", name)
	return nil
}

// GetBuilder returns the provider with the giving name, nil if it doesn't exist
func (r *registry) GetBuilder(name string) BuilderConstructor {
	r.lock.RLock()
	defer r.lock.RUnlock()

	name = strings.ToLower(name)
	return r.builders[name]
}

// BuildBuilder reads provider configuration and instatiate one
func (r *registry) BuildBuilder(c *common.Config) (Builder, error) {
	var config BuilderConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	builder := r.GetBuilder(config.Type)
	if builder == nil {
		return nil, fmt.Errorf("unknown autodiscover builder %s", config.Type)
	}

	return builder(c)
}

// GetConfig creates configs for all builders initalized.
func (b Builders) GetConfig(event bus.Event) []*common.Config {
	var configs []*common.Config

	for _, builder := range b {
		if config := builder.CreateConfig(event); config != nil {
			configs = append(configs, config...)
		}
	}

	return configs
}

// NewBuilders instances the given list of builders. If hintsEnabled is true it will
// just enable the hints builder
func NewBuilders(bConfigs []*common.Config, hintsEnabled bool) (Builders, error) {
	var builders Builders
	if hintsEnabled {
		if len(bConfigs) > 0 {
			return nil, errors.New("hints.enabled is incompatible with manually defining builders")
		}

		hints, err := common.NewConfigFrom(map[string]string{"type": "hints"})
		if err != nil {
			return nil, err
		}

		bConfigs = append(bConfigs, hints)
	}

	for _, bcfg := range bConfigs {
		builder, err := Registry.BuildBuilder(bcfg)
		if err != nil {
			return nil, err
		}
		builders = append(builders, builder)
	}

	return builders, nil
}
