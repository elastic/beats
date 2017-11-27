package autodiscover

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

// Provider for autodiscover
type Provider interface {
	cfgfile.Runner
}

// ProviderRegistry holds all known autodiscover providers, they must be added to it to enable them for use
var ProviderRegistry = NewRegistry()

// ProviderBuilder creates a new provider based on the given config and returns it
type ProviderBuilder func(bus.Bus, *common.Config) (Provider, error)

// Register of autodiscover providers
type registry struct {
	// Lock to control concurrent read/writes
	lock sync.RWMutex
	// A map of provider name to ProviderBuilder.
	providers map[string]ProviderBuilder
}

// NewRegistry creates and returns a new Registry
func NewRegistry() *registry {
	return &registry{
		providers: make(map[string]ProviderBuilder, 0),
	}
}

// AddProvider registers a new ProviderBuilder
func (r *registry) AddProvider(name string, provider ProviderBuilder) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("provider name is required")
	}

	_, exists := r.providers[name]
	if exists {
		return fmt.Errorf("provider '%s' is already registered", name)
	}

	if provider == nil {
		return fmt.Errorf("provider '%s' cannot be registered with a nil factory", name)
	}

	r.providers[name] = provider
	logp.Debug(debugK, "Provider registered: %s", name)
	return nil
}

// GetProvider returns the provider with the giving name, nil if it doesn't exist
func (r *registry) GetProvider(name string) ProviderBuilder {
	r.lock.RLock()
	defer r.lock.RUnlock()

	name = strings.ToLower(name)
	return r.providers[name]
}

// BuildProvider reads provider configuration and instatiate one
func (r *registry) BuildProvider(bus bus.Bus, c *common.Config) (Provider, error) {
	var config ProviderConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	builder := r.GetProvider(config.Type)
	if builder == nil {
		return nil, fmt.Errorf("Unknown autodiscover provider %s", config.Type)
	}

	return builder(bus, c)
}
