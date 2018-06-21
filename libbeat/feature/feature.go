package feature

import (
	"fmt"
	"reflect"
)

//go:generate stringer -type=Stability

// Registry is the global plugin registry, this variable is mean to be temporary to move all the
// internal factory to receive a context that include the current beat registry.
var Registry = newRegistry()

// Featurable implements the description of a feature.
type Featurable interface {
	// Namespace returns the namespace of the Feature.
	Namespace() string

	// Name returns the name of the feature. The name must be unique for each namespace.
	Name() string

	// Factory returns the factory func.
	Factory() interface{}

	// Stability returns the stability of the feature.
	Stability() Stability

	// Equal returns true if the two object are equal.
	Equal(other Featurable) bool

	String() string
}

// Feature contains the information for a specific feature
type Feature struct {
	namespace string
	name      string
	factory   interface{}
	stability Stability
}

// Namespace return the namespace of the feature.
func (f *Feature) Namespace() string {
	return f.namespace
}

// Name returns the name of the feature.
func (f *Feature) Name() string {
	return f.name
}

// Factory returns the factory for the feature.
func (f *Feature) Factory() interface{} {
	return f.factory
}

// Stability returns the stability level of the feature, current: stable, beta, experimental.
func (f *Feature) Stability() Stability {
	return f.stability
}

// Equal return true if both object are equals.
func (f *Feature) Equal(other Featurable) bool {
	// There is no safe way to compare function in go,
	// but since the method are global it should be stable.
	if f.Name() == other.Name() &&
		f.Namespace() == other.Namespace() &&
		reflect.ValueOf(f.Factory()).Pointer() == reflect.ValueOf(other.Factory()).Pointer() {
		return true
	}

	return false
}

// String return the debug information
func (f *Feature) String() string {
	return fmt.Sprintf("%s/%s (stability: %s)", f.namespace, f.name, f.stability)
}

// Stability defines the stability of the feature, this value can be used to filter a bundler.
type Stability int

// List all the available stability for a feature.
const (
	Stable Stability = iota
	Beta
	Experimental
	Undefined
)

// New returns a new Feature.
func New(namespace, name string, factory interface{}, stability Stability) *Feature {
	return &Feature{
		namespace: namespace,
		name:      name,
		factory:   factory,
		stability: stability,
	}
}

// RegisterBundle registers a bundle of features.
func RegisterBundle(bundle Bundle) error {
	for _, f := range bundle.Features {
		Registry.Register(f)
	}
	return nil
}

// Register register a new feature on the global registry.
func Register(feature Featurable) error {
	return Registry.Register(feature)
}

// MustRegister register a new Feature on the global registry and panic on error.
func MustRegister(feature Featurable) {
	err := Register(feature)
	if err != nil {
		panic(err)
	}
}
