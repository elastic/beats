package feature

// Bundleable merges featurable and bundle interface together.
type bundleable interface {
	Features() []Featurable
}

// Bundle defines a list of features available in the current beat.
type Bundle struct {
	features []Featurable
}

// NewBundle creates a new Bundle of feature to be registered.
func NewBundle(features ...Featurable) *Bundle {
	return &Bundle{features: features}
}

// Filter creates a new bundle with only the feature matching the requested stability.
func (b *Bundle) Filter(stabilities ...Stability) *Bundle {
	var filtered []Featurable

	for _, feature := range b.features {
		for _, stability := range stabilities {
			if feature.Stability() == stability {
				filtered = append(filtered, feature)
				break
			}
		}
	}
	return NewBundle(filtered...)
}

// Features returns the interface features slice so
func (b *Bundle) Features() []Featurable {
	return b.features
}

// MustBundle takes existing bundle or features and create a new Bundle with all the merged Features.
func MustBundle(bundle ...bundleable) *Bundle {
	var merged []Featurable
	for _, feature := range bundle {
		merged = append(merged, feature.Features()...)
	}
	return NewBundle(merged...)
}
