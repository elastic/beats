package composable

import "context"

// DynamicProvider is the interface that a dynamic provider must implement.
type DynamicProvider interface {
	// Run runs the inventory provider.
	Run(DynamicProviderComm) error
}

// DynamicProvider is the interface that a dynamic provider must implement.
type DynamicProviderSecrets interface {
	// Run runs the inventory provider.
	Run(DynamicProviderComm) error
	// Run runs the inventory provider.
	Fetch() error
}

// DynamicProviderComm is the interface that an dynamic provider uses to communicate back to Elastic Agent.
type DynamicProviderComm interface {
	context.Context

	// AddOrUpdate updates a mapping with given ID with latest mapping and processors.
	//
	// `priority` ensures that order is maintained when adding the mapping to the current state
	// for the processor. Lower priority mappings will always be sorted before higher priority mappings
	// to ensure that matching of variables occurs on the lower priority mappings first.
	AddOrUpdate(id string, priority int, mapping map[string]interface{}, processors []map[string]interface{}) error
	// Remove removes a mapping by given ID.
	Remove(id string)
}
