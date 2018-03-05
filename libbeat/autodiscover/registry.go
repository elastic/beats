package autodiscover

import "sync"

// Register of autodiscover providers
type registry struct {
	// Lock to control concurrent read/writes
	lock sync.RWMutex
	// A map of provider name to ProviderBuilder.
	providers map[string]ProviderBuilder
	// A map of builder name to BuilderConstructor.
	builders map[string]BuilderConstructor
}

// Registry holds all known autodiscover providers, they must be added to it to enable them for use
var Registry = NewRegistry()

// NewRegistry creates and returns a new Registry
func NewRegistry() *registry {
	return &registry{
		providers: make(map[string]ProviderBuilder, 0),
		builders:  make(map[string]BuilderConstructor, 0),
	}
}
